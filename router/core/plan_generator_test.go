package core

import (
	"path"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wundergraph/graphql-go-tools/v2/pkg/ast"
	"github.com/wundergraph/graphql-go-tools/v2/pkg/engine/plan"
	"go.uber.org/zap"
)

func TestPlanOperationPanic(t *testing.T) {
	// Create a minimal plan configuration
	planConfig := &plan.Configuration{}

	// Create a planner with minimal configuration
	planner, err := NewPlanner(planConfig, &ast.Document{}, &ast.Document{})
	if err != nil {
		t.Fatalf("Failed to create planner: %v", err)
	}

	// Create an invalid operation document that will cause a panic
	invalidOperation := &ast.Document{
		RootNodes: []ast.Node{
			{
				Kind: ast.NodeKindOperationDefinition,
				Ref:  0,
			},
		},
	}

	assert.NotPanics(t, func() {
		_, _, err = planner.PlanPreparedOperation(invalidOperation)
		assert.Error(t, err)
	})
}

func TestValidateOperationPanic(t *testing.T) {
	// Create a minimal plan configuration
	planConfig := &plan.Configuration{}

	// Create a planner with minimal configuration
	planner, err := NewPlanner(planConfig, &ast.Document{}, &ast.Document{})
	if err != nil {
		t.Fatalf("Failed to create planner: %v", err)
	}

	// Create an invalid operation document that will cause a panic
	invalidOperation := &ast.Document{
		RootNodes: []ast.Node{
			{
				Kind: ast.NodeKindOperationDefinition,
				Ref:  0,
			},
		},
	}

	// Attempt to validate the operation - this should panic
	assert.NotPanics(t, func() {
		err = planner.validateOperation(invalidOperation)
		assert.Error(t, err)
	})
}

func planGeneratorTestDataDir() string {
	_, filename, _, _ := runtime.Caller(0)
	return path.Join(filepath.Dir(filename), "..", "pkg", "plan_generator", "testdata")
}

func TestSubgraphOperations(t *testing.T) {
	pg, err := NewPlanGenerator(
		path.Join(planGeneratorTestDataDir(), "execution_config", "base.json"),
		zap.NewNop(),
		0,
	)
	require.NoError(t, err)

	planForFile := func(t *testing.T) *PlanWrapper {
		planner, err := pg.GetPlanner()
		require.NoError(t, err)

		operation, _, err := planner.ParseAndPrepareOperation(
			path.Join(planGeneratorTestDataDir(), "queries", "base", "1.graphql"),
		)
		require.NoError(t, err)

		planWrapper, _, err := planner.PlanPreparedOperation(operation)
		require.NoError(t, err)

		return planWrapper
	}

	// 1.graphql selects `employees` and `products` fields, so it must
	// produce fetches to both subgraphs.
	ops := planForFile(t).SubgraphOperations()
	require.NotEmpty(t, ops)

	subgraphs := make(map[string]bool)
	for _, op := range ops {
		assert.NotEmpty(t, op.Query, "subgraph %s: document text must not be empty", op.SubgraphName)
		subgraphs[op.SubgraphName] = true
	}
	assert.True(t, subgraphs["employees"], "expected a fetch to the employees subgraph")
	assert.True(t, subgraphs["products"], "expected a fetch to the products subgraph")

	// Planners must not be reused, but re-planning the same operation from
	// scratch must produce byte-identical document text: the exported
	// persisted-operation hash depends on this being stable.
	opsAgain := planForFile(t).SubgraphOperations()
	assert.Equal(t, ops, opsAgain)
}
