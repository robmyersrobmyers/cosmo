// Package persisted_operations_export plans GraphQL operations offline against a
// static execution config and writes out, per subgraph, the persisted-operation
// files a subgraph's own router would need to enforce
// security.block_non_persisted_operations against the exact documents the
// gateway sends it.
package persisted_operations_export

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"go.uber.org/zap"

	"github.com/wundergraph/cosmo/router/core"
	"github.com/wundergraph/cosmo/router/internal/persistedoperation"
)

const ReportFileName = "report.json"

type ExportConfig struct {
	ExecutionConfig                    string
	SourceDir                          string
	OutDir                             string
	Concurrency                        int
	Filter                             string
	Timeout                            string
	OutputReport                       bool
	FailOnPlanError                    bool
	FailFast                           bool
	LogLevel                           string
	Logger                             *zap.Logger
	MaxDataSourceCollectorsConcurrency uint
}

type ExportResults struct {
	Files []ExportResult `json:"files,omitempty"`
	Error string         `json:"error,omitempty"`
}

type ExportResult struct {
	FileName  string   `json:"file_name,omitempty"`
	Subgraphs []string `json:"subgraphs,omitempty"`
	Error     string   `json:"error,omitempty"`
	Warning   string   `json:"warning,omitempty"`
}

// Export reads GraphQL operation files from cfg.SourceDir, plans each of them
// offline against cfg.ExecutionConfig, and writes one <hash>.json persisted
// operation file per unique (subgraph, document) pair into
// cfg.OutDir/<subgraph-name>/<hash>.json. The output layout matches what
// internal/persistedoperation/operationstorage/fs.Client reads, so each
// subgraph's output directory can be pointed at directly by a file_system
// storage provider on that subgraph's own router, with no CDN/S3/control-plane
// access required on either side.
func Export(ctx context.Context, cfg ExportConfig) error {
	if cfg.Concurrency == 0 {
		cfg.Concurrency = runtime.GOMAXPROCS(0)
	}

	queriesPath, err := filepath.Abs(cfg.SourceDir)
	if err != nil {
		return fmt.Errorf("failed to get absolute path for queries: %w", err)
	}

	outPath, err := filepath.Abs(cfg.OutDir)
	if err != nil {
		return fmt.Errorf("failed to get absolute path for output: %w", err)
	}
	if err := os.MkdirAll(outPath, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	executionConfigPath, err := filepath.Abs(cfg.ExecutionConfig)
	if err != nil {
		return fmt.Errorf("failed to get absolute path for execution config: %w", err)
	}

	var filter []string
	if cfg.Filter != "" {
		filterContent, err := os.ReadFile(cfg.Filter)
		if err != nil {
			return fmt.Errorf("failed to read filter file: %w", err)
		}
		filter = strings.Split(string(filterContent), "\n")
	}

	queries, err := os.ReadDir(queriesPath)
	if err != nil {
		return fmt.Errorf("failed to read queries directory: %w", err)
	}

	queriesQueue := make(chan os.DirEntry, len(queries))
	for _, queryFile := range queries {
		queriesQueue <- queryFile
	}
	close(queriesQueue)

	var results []ExportResult
	var resultsMux sync.Mutex

	duration, parseErr := time.ParseDuration(cfg.Timeout)
	if parseErr != nil {
		return fmt.Errorf("failed to parse timeout: %w", parseErr)
	}
	ctx, cancel := context.WithTimeout(ctx, duration)
	defer cancel()
	ctxError, cancelError := context.WithCancelCause(ctx)
	defer cancelError(nil)

	pg, err := core.NewPlanGenerator(executionConfigPath, cfg.Logger, cfg.MaxDataSourceCollectorsConcurrency)
	if err != nil {
		return fmt.Errorf("failed to create plan generator: %w", err)
	}

	var planError atomic.Bool
	wg := sync.WaitGroup{}
	wg.Add(cfg.Concurrency)
	for i := 0; i < cfg.Concurrency; i++ {
		go func(i int) {
			defer wg.Done()
			for {
				select {
				case <-ctxError.Done():
					return
				case queryFile, ok := <-queriesQueue:
					if !ok {
						return
					}

					if !slices.Contains([]string{".graphql", ".gql", ".graphqls"}, filepath.Ext(queryFile.Name())) {
						continue
					}

					if len(filter) > 0 && !slices.Contains(filter, queryFile.Name()) {
						continue
					}

					queryFilePath := filepath.Join(queriesPath, queryFile.Name())
					res := exportOperation(pg, queryFilePath, queryFile.Name(), outPath)

					resultsMux.Lock()
					results = append(results, res)
					resultsMux.Unlock()

					if res.Error != "" {
						planError.Store(true)
						if cfg.FailFast {
							cancel()
						}
					}
				}
			}
		}(i)
	}
	wg.Wait()

	if cfg.OutputReport {
		if err := writeReport(outPath, results, ctxError); err != nil {
			return err
		}
	}

	if cfg.FailOnPlanError && planError.Load() {
		return fmt.Errorf("some operations failed to export")
	}

	return context.Cause(ctxError)
}

// exportOperation plans a single operation file and writes its distinct
// per-subgraph persisted operation files. Planners must not be reused across
// operations.
func exportOperation(pg *core.PlanGenerator, queryFilePath, fileName, outPath string) ExportResult {
	res := ExportResult{FileName: fileName}

	planner, err := pg.GetPlanner()
	if err != nil {
		res.Error = fmt.Sprintf("failed to get a planner: %v", err)
		return res
	}

	operation, _, err := planner.ParseAndPrepareOperation(queryFilePath)
	if err != nil {
		if _, ok := err.(*core.PlannerOperationValidationError); ok {
			res.Warning = err.Error()
		} else {
			res.Error = err.Error()
		}
		return res
	}

	planWrapper, _, err := planner.PlanPreparedOperation(operation)
	if err != nil {
		res.Error = fmt.Sprintf("failed to plan operation: %v", err)
		return res
	}

	subgraphOps := planWrapper.SubgraphOperations()
	subgraphNames := make(map[string]struct{}, len(subgraphOps))

	for _, op := range subgraphOps {
		subgraphNames[op.SubgraphName] = struct{}{}

		if err := writePersistedOperation(outPath, op); err != nil {
			res.Error = err.Error()
			return res
		}
	}

	res.Subgraphs = make([]string, 0, len(subgraphNames))
	for name := range subgraphNames {
		res.Subgraphs = append(res.Subgraphs, name)
	}
	slices.Sort(res.Subgraphs)

	return res
}

// writePersistedOperation writes <outPath>/<subgraphName>/<sha256Hash>.json in
// the format internal/persistedoperation/operationstorage/fs.Client reads.
// Concurrent writes of the same (subgraph, hash) pair from different operation
// files are safe to leave unsynchronized: the document text is deterministic
// for a given (schema, operation), so a benign last-writer-wins race can never
// produce mismatched content for the same file name.
func writePersistedOperation(outPath string, op core.SubgraphOperation) error {
	hash := sha256.Sum256([]byte(op.Query))
	hashHex := hex.EncodeToString(hash[:])

	subgraphDir := filepath.Join(outPath, op.SubgraphName)
	if err := os.MkdirAll(subgraphDir, 0755); err != nil {
		return fmt.Errorf("failed to create subgraph output directory: %w", err)
	}

	data, err := json.Marshal(persistedoperation.PersistedOperation{
		Version: 1,
		Body:    op.Query,
	})
	if err != nil {
		return fmt.Errorf("failed to marshal persisted operation: %w", err)
	}

	outFile := filepath.Join(subgraphDir, hashHex+".json")
	if err := os.WriteFile(outFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write persisted operation file: %w", err)
	}

	return nil
}

func writeReport(outPath string, results []ExportResult, ctxError context.Context) error {
	reportFilePath := filepath.Join(outPath, ReportFileName)
	reportFile, err := os.Create(reportFilePath)
	if err != nil {
		return fmt.Errorf("failed to create results file: %w", err)
	}
	defer func() {
		_ = reportFile.Close()
	}()

	slices.SortFunc(results, func(a, b ExportResult) int {
		return strings.Compare(a.FileName, b.FileName)
	})
	resultData := ExportResults{
		Files: results,
	}
	if ctxError.Err() != nil {
		resultData.Error = context.Cause(ctxError).Error()
	}

	data, jsonErr := json.Marshal(resultData)
	if jsonErr != nil {
		return fmt.Errorf("failed to marshal result: %w", jsonErr)
	}
	if _, err := fmt.Fprintf(reportFile, "%s\n", data); err != nil {
		return fmt.Errorf("failed to write result: %w", err)
	}

	return nil
}
