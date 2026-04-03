package workflow

import (
	"testing"

	"github.com/open-source-cloud/fuse/internal/packages"
	"github.com/open-source-cloud/fuse/pkg/store"
	"github.com/open-source-cloud/fuse/pkg/workflow"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testBranchA = "branch-a"
	testBranchB = "branch-b"
)

func newTestNodeWithConditionalOutput(id, conditionalField string) *Node {
	node := &Node{
		schema: &NodeSchema{ID: id, Function: "test/func"},
		functionMetadata: &packages.FunctionMetadata{
			Output: packages.FunctionOutputMetadata{
				ConditionalOutput:      true,
				ConditionalOutputField: conditionalField,
				Edges: map[string]packages.FunctionOutputEdgeMetadata{
					testBranchA: {Name: testBranchA, ConditionalEdge: workflow.ConditionalEdgeMetadata{Value: "a"}},
					testBranchB: {Name: testBranchB, ConditionalEdge: workflow.ConditionalEdgeMetadata{Value: "b"}},
				},
			},
		},
	}
	return node
}

func TestEvaluateCondition_ExactMatch(t *testing.T) {
	node := newTestNodeWithConditionalOutput("node1", "result")
	output := store.New()
	output.Set("node1", map[string]any{"result": "a"})

	condition := &EdgeCondition{Name: testBranchA, Type: ConditionExact, Value: "a"}
	result, err := EvaluateCondition(condition, output, node)

	require.NoError(t, err)
	assert.True(t, result)
}

func TestEvaluateCondition_ExactMatch_NoMatch(t *testing.T) {
	node := newTestNodeWithConditionalOutput("node1", "result")
	output := store.New()
	output.Set("node1", map[string]any{"result": "a"})

	condition := &EdgeCondition{Name: testBranchB, Type: ConditionExact, Value: "b"}
	result, err := EvaluateCondition(condition, output, node)

	require.NoError(t, err)
	assert.False(t, result)
}

func TestEvaluateCondition_EmptyType_LegacyExactMatch(t *testing.T) {
	node := newTestNodeWithConditionalOutput("node1", "result")
	output := store.New()
	output.Set("node1", map[string]any{"result": true})

	condition := &EdgeCondition{Name: "if-true", Value: true}
	result, err := EvaluateCondition(condition, output, node)

	require.NoError(t, err)
	assert.True(t, result)
}

func TestEvaluateCondition_Expression_SimpleComparison(t *testing.T) {
	node := newTestNodeWithConditionalOutput("node1", "result")
	output := store.New()
	output.Set("node1", map[string]any{"amount": 1500})

	condition := &EdgeCondition{Name: "high-value", Type: ConditionExpression, Expression: "output.amount > 1000"}
	result, err := EvaluateCondition(condition, output, node)

	require.NoError(t, err)
	assert.True(t, result)
}

func TestEvaluateCondition_Expression_FalseResult(t *testing.T) {
	node := newTestNodeWithConditionalOutput("node1", "result")
	output := store.New()
	output.Set("node1", map[string]any{"amount": 500})

	condition := &EdgeCondition{Name: "high-value", Type: ConditionExpression, Expression: "output.amount > 1000"}
	result, err := EvaluateCondition(condition, output, node)

	require.NoError(t, err)
	assert.False(t, result)
}

func TestEvaluateCondition_Expression_MultipleConditions(t *testing.T) {
	node := newTestNodeWithConditionalOutput("node1", "result")
	output := store.New()
	output.Set("node1", map[string]any{"status": "active", "tier": "premium"})

	condition := &EdgeCondition{
		Name:       "premium-active",
		Type:       ConditionExpression,
		Expression: `output.status == "active" && output.tier == "premium"`,
	}
	result, err := EvaluateCondition(condition, output, node)

	require.NoError(t, err)
	assert.True(t, result)
}

func TestEvaluateCondition_Expression_CompileError(t *testing.T) {
	node := newTestNodeWithConditionalOutput("checker", "status")
	output := store.New()

	condition := &EdgeCondition{Name: "bad", Type: ConditionExpression, Expression: "invalid @@@ syntax"}
	_, err := EvaluateCondition(condition, output, node)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to compile expression")
}

func TestEvaluateCondition_Default(t *testing.T) {
	node := newTestNodeWithConditionalOutput("validator", "valid")
	output := store.New()

	condition := &EdgeCondition{Name: "fallback", Type: ConditionDefault}
	result, err := EvaluateCondition(condition, output, node)

	require.NoError(t, err)
	assert.True(t, result)
}
