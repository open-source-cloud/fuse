package workflow

import (
	"testing"

	"github.com/open-source-cloud/fuse/internal/packages"
	"github.com/open-source-cloud/fuse/pkg/store"
	"github.com/open-source-cloud/fuse/pkg/workflow"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// buildConditionalNode creates a test node with conditional output metadata and output edges
func buildConditionalNode(nodeID, conditionalField string, edges []*Edge) *Node {
	return &Node{
		schema: &NodeSchema{ID: nodeID, Function: "test/func"},
		functionMetadata: &packages.FunctionMetadata{
			Output: packages.FunctionOutputMetadata{
				ConditionalOutput:      true,
				ConditionalOutputField: conditionalField,
				Edges: map[string]packages.FunctionOutputEdgeMetadata{
					testBranchA: {Name: testBranchA},
					testBranchB: {Name: testBranchB},
				},
			},
		},
		outputEdges: edges,
	}
}

func buildNonConditionalNode(nodeID string, edges []*Edge) *Node {
	return &Node{
		schema: &NodeSchema{ID: nodeID, Function: "test/func"},
		functionMetadata: &packages.FunctionMetadata{
			Output: packages.FunctionOutputMetadata{
				ConditionalOutput: false,
			},
		},
		outputEdges: edges,
	}
}

func buildEdge(id string, condition *EdgeCondition, toNode *Node) *Edge {
	return &Edge{
		id: id,
		schema: &EdgeSchema{
			ID:          id,
			Conditional: condition,
		},
		to: toNode,
	}
}

func TestFilterOutputEdgesByConditionals_NonConditionalNode(t *testing.T) {
	target1 := &Node{schema: &NodeSchema{ID: "t1"}}
	target2 := &Node{schema: &NodeSchema{ID: "t2"}}
	e1 := buildEdge("e1", nil, target1)
	e2 := buildEdge("e2", nil, target2)
	node := buildNonConditionalNode("source", []*Edge{e1, e2})

	output := store.New()
	wf := &Workflow{aggregatedOutput: output}

	result := wf.filterOutputEdgesByConditionals(node)

	assert.Len(t, result, 2)
}

func TestFilterOutputEdgesByConditionals_ExactMatch(t *testing.T) {
	targetA := &Node{schema: &NodeSchema{ID: "branch-a-node"}}
	targetB := &Node{schema: &NodeSchema{ID: "branch-b-node"}}
	edgeA := buildEdge("ea", &EdgeCondition{Name: testBranchA, Type: ConditionExact, Value: "a"}, targetA)
	edgeB := buildEdge("eb", &EdgeCondition{Name: testBranchB, Type: ConditionExact, Value: "b"}, targetB)
	node := buildConditionalNode("src", "result", []*Edge{edgeA, edgeB})

	output := store.New()
	output.Set("src", map[string]any{"result": "a"})
	wf := &Workflow{aggregatedOutput: output}

	result := wf.filterOutputEdgesByConditionals(node)

	require.Len(t, result, 1)
	assert.Equal(t, "ea", result[0].ID())
}

func TestFilterOutputEdgesByConditionals_ExpressionMatch(t *testing.T) {
	targetHigh := &Node{schema: &NodeSchema{ID: "high"}}
	targetLow := &Node{schema: &NodeSchema{ID: "low"}}
	edgeHigh := buildEdge("eh", &EdgeCondition{Name: "high-value", Type: ConditionExpression, Expression: "output.amount > 1000"}, targetHigh)
	edgeLow := buildEdge("el", &EdgeCondition{Name: "low-value", Type: ConditionExpression, Expression: "output.amount <= 1000"}, targetLow)
	node := buildConditionalNode("calc", "amount", []*Edge{edgeHigh, edgeLow})

	output := store.New()
	output.Set("calc", map[string]any{"amount": 5000})
	wf := &Workflow{aggregatedOutput: output}

	result := wf.filterOutputEdgesByConditionals(node)

	require.Len(t, result, 1)
	assert.Equal(t, "eh", result[0].ID())
}

func TestFilterOutputEdgesByConditionals_DefaultFallback(t *testing.T) {
	targetA := &Node{schema: &NodeSchema{ID: "a"}}
	targetDefault := &Node{schema: &NodeSchema{ID: "default"}}
	edgeA := buildEdge("ea", &EdgeCondition{Name: "match-a", Type: ConditionExact, Value: "a"}, targetA)
	edgeDefault := buildEdge("ed", &EdgeCondition{Name: "fallback", Type: ConditionDefault}, targetDefault)
	node := buildConditionalNode("src", "result", []*Edge{edgeA, edgeDefault})

	output := store.New()
	output.Set("src", map[string]any{"result": "no-match"}) // doesn't match "a"
	wf := &Workflow{aggregatedOutput: output}

	result := wf.filterOutputEdgesByConditionals(node)

	require.Len(t, result, 1)
	assert.Equal(t, "ed", result[0].ID())
}

func TestFilterOutputEdgesByConditionals_DefaultNotUsedWhenOtherMatches(t *testing.T) {
	targetA := &Node{schema: &NodeSchema{ID: "a"}}
	targetDefault := &Node{schema: &NodeSchema{ID: "default"}}
	edgeA := buildEdge("ea", &EdgeCondition{Name: "match-a", Type: ConditionExact, Value: "a"}, targetA)
	edgeDefault := buildEdge("ed", &EdgeCondition{Name: "fallback", Type: ConditionDefault}, targetDefault)
	node := buildConditionalNode("src", "result", []*Edge{edgeA, edgeDefault})

	output := store.New()
	output.Set("src", map[string]any{"result": "a"})
	wf := &Workflow{aggregatedOutput: output}

	result := wf.filterOutputEdgesByConditionals(node)

	require.Len(t, result, 1)
	assert.Equal(t, "ea", result[0].ID())
}

func TestFilterOutputEdgesByConditionals_ExpressionError_EdgeSkipped(t *testing.T) {
	targetBad := &Node{schema: &NodeSchema{ID: "bad"}}
	targetGood := &Node{schema: &NodeSchema{ID: "good"}}
	edgeBad := buildEdge("ebad", &EdgeCondition{Name: "bad-expr", Type: ConditionExpression, Expression: "invalid @@@ syntax"}, targetBad)
	edgeGood := buildEdge("egood", &EdgeCondition{Name: "good-expr", Type: ConditionExpression, Expression: "output.ok == true"}, targetGood)
	node := buildConditionalNode("src", "result", []*Edge{edgeBad, edgeGood})

	output := store.New()
	output.Set("src", map[string]any{"ok": true})
	wf := &Workflow{aggregatedOutput: output}

	result := wf.filterOutputEdgesByConditionals(node)

	// Bad expression is skipped, good one matches
	require.Len(t, result, 1)
	assert.Equal(t, "egood", result[0].ID())
}

func TestFilterOutputEdgesByConditionals_NoConditionOnEdge_PassesThrough(t *testing.T) {
	target := &Node{schema: &NodeSchema{ID: "target"}}
	edge := buildEdge("e1", nil, target) // nil condition
	node := buildConditionalNode("src", "result", []*Edge{edge})

	output := store.New()
	wf := &Workflow{aggregatedOutput: output}

	result := wf.filterOutputEdgesByConditionals(node)

	require.Len(t, result, 1)
	assert.Equal(t, "e1", result[0].ID())
}

func TestFilterOutputEdgesByConditionals_MultipleExpressionMatches(t *testing.T) {
	target1 := &Node{schema: &NodeSchema{ID: "t1"}}
	target2 := &Node{schema: &NodeSchema{ID: "t2"}}
	// Both expressions match
	edge1 := buildEdge("e1", &EdgeCondition{Name: "check1", Type: ConditionExpression, Expression: "output.val > 5"}, target1)
	edge2 := buildEdge("e2", &EdgeCondition{Name: "check2", Type: ConditionExpression, Expression: "output.val < 20"}, target2)
	node := buildConditionalNode("src", "result", []*Edge{edge1, edge2})

	output := store.New()
	output.Set("src", map[string]any{"val": 10})
	wf := &Workflow{aggregatedOutput: output}

	result := wf.filterOutputEdgesByConditionals(node)

	// Both should match — results in parallel execution
	assert.Len(t, result, 2)
}

func buildJoinNodeWithMeta(id string, merge *MergeConfig) *Node {
	return &Node{
		schema: &NodeSchema{
			ID:       id,
			Function: "test/join",
			Merge:    merge,
		},
		functionMetadata: &packages.FunctionMetadata{
			Input: packages.FunctionInputMetadata{
				Parameters: map[string]workflow.ParameterSchema{
					"data":  {Name: "data", Type: "string"},
					"count": {Name: "count", Type: "int"},
				},
			},
		},
		thread:     2,
		inputEdges: make([]*Edge, 0),
	}
}

func buildJoinEdge(id, mapTo string, fromNode, toNode *Node) *Edge {
	return &Edge{
		id: id,
		schema: &EdgeSchema{
			ID:   id,
			From: fromNode.ID(),
			To:   toNode.ID(),
			Input: []InputMapping{
				{Source: SourceSchema, Value: "result-" + id, MapTo: mapTo},
			},
		},
		from: fromNode,
		to:   toNode,
	}
}

func TestResolveJoinInputs_WithMergeStrategy(t *testing.T) {
	joinNode := buildJoinNodeWithMeta("join", &MergeConfig{Strategy: MergeKeyed})
	fromNode1 := &Node{schema: &NodeSchema{ID: "branch1"}, thread: 0}
	fromNode2 := &Node{schema: &NodeSchema{ID: "branch2"}, thread: 1}

	edge1 := buildJoinEdge("e1", "data", fromNode1, joinNode)
	edge2 := buildJoinEdge("e2", "count", fromNode2, joinNode)
	joinNode.inputEdges = []*Edge{edge1, edge2}

	wf := &Workflow{aggregatedOutput: store.New()}

	result := wf.resolveJoinInputs(joinNode)

	// With keyed strategy, results are grouped by edge ID
	assert.Contains(t, result, "e1")
	assert.Contains(t, result, "e2")
}

func TestResolveJoinInputs_DefaultAppendStrategy(t *testing.T) {
	joinNode := buildJoinNodeWithMeta("join", nil) // nil = default append
	fromNode1 := &Node{schema: &NodeSchema{ID: "b1"}, thread: 0}
	fromNode2 := &Node{schema: &NodeSchema{ID: "b2"}, thread: 1}

	edge1 := buildJoinEdge("e1", "data", fromNode1, joinNode)
	edge2 := buildJoinEdge("e2", "data", fromNode2, joinNode)
	joinNode.inputEdges = []*Edge{edge1, edge2}

	wf := &Workflow{aggregatedOutput: store.New()}

	result := wf.resolveJoinInputs(joinNode)

	// Default append: same key from multiple branches becomes an array
	data, exists := result["data"]
	require.True(t, exists)
	arr, ok := data.([]any)
	require.True(t, ok)
	assert.Len(t, arr, 2)
}

func TestWorkflow_StateCancelled(t *testing.T) {
	schema := &GraphSchema{
		ID:   "test",
		Name: "test",
		Nodes: []*NodeSchema{
			{ID: "n1", Function: "debug/nil"},
			{ID: "n2", Function: "debug/nil"},
		},
		Edges: []*EdgeSchema{
			{ID: "e1", From: "n1", To: "n2"},
		},
	}

	graph, err := NewGraph(schema)
	require.NoError(t, err)

	wf := New(workflow.NewID(), graph)

	wf.SetState(StateRunning)
	assert.Equal(t, StateRunning, wf.State())

	wf.SetState(StateCancelled)
	assert.Equal(t, StateCancelled, wf.State())

	// Journal should contain state change entries
	entries := wf.Journal().Entries()
	require.GreaterOrEqual(t, len(entries), 2)

	// Last entry should be the cancelled state
	lastEntry := entries[len(entries)-1]
	assert.Equal(t, JournalStateChanged, lastEntry.Type)
	assert.Equal(t, StateCancelled, lastEntry.State)
}
