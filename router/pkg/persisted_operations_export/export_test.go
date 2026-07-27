package persisted_operations_export

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/wundergraph/cosmo/router/internal/persistedoperation"
	"github.com/wundergraph/cosmo/router/internal/persistedoperation/operationstorage/fs"
)

// The plan_generator package already ships a fixture set for the classic Cosmo
// demo graph (employees + products subgraphs); reuse it rather than duplicating
// execution config / operation fixtures.
func getTestDataDir() string {
	_, filename, _, _ := runtime.Caller(0)
	return path.Join(filepath.Dir(filename), "..", "plan_generator", "testdata")
}

func TestExport(t *testing.T) {
	t.Run("checks queries path exists", func(t *testing.T) {
		tempDir := t.TempDir()

		cfg := ExportConfig{
			SourceDir:       path.Join(getTestDataDir(), "queries", "notexistant"),
			OutDir:          tempDir,
			ExecutionConfig: path.Join(getTestDataDir(), "execution_config", "base.json"),
			Timeout:         "30s",
		}

		err := Export(context.Background(), cfg)
		assert.ErrorContains(t, err, "failed to read queries directory:")
	})

	t.Run("checks filter file exists", func(t *testing.T) {
		tempDir := t.TempDir()

		cfg := ExportConfig{
			SourceDir:       path.Join(getTestDataDir(), "queries", "base"),
			OutDir:          tempDir,
			ExecutionConfig: path.Join(getTestDataDir(), "execution_config", "base.json"),
			Filter:          path.Join(getTestDataDir(), "not_existant", "filter.txt"),
			Timeout:         "30s",
		}

		err := Export(context.Background(), cfg)
		assert.ErrorContains(t, err, "failed to read filter file:")
	})

	t.Run("fail if execution config doesn't exist", func(t *testing.T) {
		tempDir := t.TempDir()

		cfg := ExportConfig{
			SourceDir:       path.Join(getTestDataDir(), "queries", "base"),
			OutDir:          tempDir,
			ExecutionConfig: path.Join(getTestDataDir(), "not_existant", "base.json"),
			Timeout:         "30s",
		}

		err := Export(context.Background(), cfg)
		assert.ErrorContains(t, err, "failed to create plan generator:")
	})

	t.Run("fail with invalid execution config", func(t *testing.T) {
		tempDir := t.TempDir()

		cfg := ExportConfig{
			SourceDir:       path.Join(getTestDataDir(), "queries", "base"),
			OutDir:          tempDir,
			ExecutionConfig: path.Join(getTestDataDir(), "execution_config", "wrong.json"),
			Timeout:         "30s",
		}

		err := Export(context.Background(), cfg)
		assert.ErrorContains(t, err, "unexpected EOF")
	})

	t.Run("fails with wrong timeout duration", func(t *testing.T) {
		tempDir := t.TempDir()

		cfg := ExportConfig{
			SourceDir:       path.Join(getTestDataDir(), "queries", "base"),
			OutDir:          tempDir,
			ExecutionConfig: path.Join(getTestDataDir(), "execution_config", "base.json"),
			Timeout:         "30as",
		}

		err := Export(context.Background(), cfg)
		assert.ErrorContains(t, err, "failed to parse timeout:")
	})

	t.Run("exports one file per unique subgraph document, readable by the fs storage client", func(t *testing.T) {
		tempDir := t.TempDir()

		cfg := ExportConfig{
			SourceDir:       path.Join(getTestDataDir(), "queries", "base"),
			OutDir:          tempDir,
			ExecutionConfig: path.Join(getTestDataDir(), "execution_config", "base.json"),
			Timeout:         "30s",
			OutputReport:    true,
			Logger:          zap.NewNop(),
		}

		err := Export(context.Background(), cfg)
		require.NoError(t, err)

		// 1.graphql in the shared fixture set spans both the employees and
		// products subgraphs, so both output directories must exist.
		entries, err := os.ReadDir(tempDir)
		require.NoError(t, err)

		subgraphDirs := make(map[string]bool)
		for _, e := range entries {
			if e.IsDir() {
				subgraphDirs[e.Name()] = true
			}
		}
		assert.True(t, subgraphDirs["employees"], "expected an employees subgraph output directory")
		assert.True(t, subgraphDirs["products"], "expected a products subgraph output directory")

		for subgraph := range subgraphDirs {
			subgraphDir := path.Join(tempDir, subgraph)
			files, err := os.ReadDir(subgraphDir)
			require.NoError(t, err)
			require.NotEmpty(t, files, "expected at least one persisted operation file for %s", subgraph)

			for _, f := range files {
				hashFromName := f.Name()[:len(f.Name())-len(".json")]

				content, err := os.ReadFile(path.Join(subgraphDir, f.Name()))
				require.NoError(t, err)

				var po persistedoperation.PersistedOperation
				require.NoError(t, json.Unmarshal(content, &po))
				assert.Equal(t, 1, po.Version)
				assert.NotEmpty(t, po.Body)

				sum := sha256.Sum256([]byte(po.Body))
				assert.Equal(t, hex.EncodeToString(sum[:]), hashFromName, "file name must be the sha256 hash of its body")

				// The critical end-to-end proof: the exact client a subgraph's own
				// router uses in production must be able to read this file back by hash.
				fsClient, err := fs.NewClient(subgraphDir, &fs.Options{})
				require.NoError(t, err)
				body, err := fsClient.PersistedOperation(context.Background(), "", hashFromName)
				require.NoError(t, err)
				assert.Equal(t, po.Body, string(body))
			}
		}
	})

	t.Run("is deterministic across repeated runs", func(t *testing.T) {
		tempDir1 := t.TempDir()
		tempDir2 := t.TempDir()

		cfg1 := ExportConfig{
			SourceDir:       path.Join(getTestDataDir(), "queries", "base"),
			OutDir:          tempDir1,
			ExecutionConfig: path.Join(getTestDataDir(), "execution_config", "base.json"),
			Timeout:         "30s",
			Logger:          zap.NewNop(),
		}
		cfg2 := cfg1
		cfg2.OutDir = tempDir2

		require.NoError(t, Export(context.Background(), cfg1))
		require.NoError(t, Export(context.Background(), cfg2))

		assertDirsEqual(t, tempDir1, tempDir2)
	})
}

func assertDirsEqual(t *testing.T, dir1, dir2 string) {
	t.Helper()

	entries1, err := os.ReadDir(dir1)
	require.NoError(t, err)
	entries2, err := os.ReadDir(dir2)
	require.NoError(t, err)
	require.Equal(t, len(entries1), len(entries2))

	for _, e := range entries1 {
		p1 := path.Join(dir1, e.Name())
		p2 := path.Join(dir2, e.Name())

		if e.IsDir() {
			assertDirsEqual(t, p1, p2)
			continue
		}

		c1, err := os.ReadFile(p1)
		require.NoError(t, err)
		c2, err := os.ReadFile(p2)
		require.NoError(t, err)
		assert.Equal(t, c1, c2)
	}
}
