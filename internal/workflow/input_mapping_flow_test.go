package workflow

import (
	"testing"

	"github.com/open-source-cloud/fuse/internal/packages"
	"github.com/open-source-cloud/fuse/pkg/store"
	pkgworkflow "github.com/open-source-cloud/fuse/pkg/workflow"
	"github.com/stretchr/testify/require"
)

func TestInputMapping_SourceFlow_sliceDestinationValidatesUpstreamScalar(t *testing.T) {
	sumMeta := &packages.FunctionMetadata{
		Input: packages.FunctionInputMetadata{
			CustomParameters: false,
			Parameters: map[string]pkgworkflow.ParameterSchema{
				"values": {Name: "values", Type: "[]float64", Required: true},
			},
		},
	}
	randMeta := &packages.FunctionMetadata{
		Output: packages.FunctionOutputMetadata{
			Parameters: map[string]pkgworkflow.ParameterSchema{
				"rand": {Name: "rand", Type: "int"},
			},
		},
	}

	sumNode := &Node{
		schema:           &NodeSchema{ID: "logic-sum", Function: "fuse/pkg/logic/sum"},
		functionMetadata: sumMeta,
	}
	randNode := &Node{
		schema:           &NodeSchema{ID: "logic-rand-1", Function: "fuse/pkg/logic/rand"},
		functionMetadata: randMeta,
	}

	edge := &Edge{
		id:     "e1",
		from:   randNode,
		to:     sumNode,
		schema: &EdgeSchema{ID: "e1"},
	}
	edge.schema.Input = []InputMapping{{
		Source:   SourceFlow,
		Variable: "logic-rand-1.rand",
		MapTo:    "values",
	}}

	out := store.New()
	out.Set("logic-rand-1.rand", 42)

	wf := &Workflow{aggregatedOutput: out}
	args := wf.inputMapping(edge, edge.Input())

	val, ok := args["values"].([]float64)
	require.True(t, ok, "got %T", args["values"])
	require.Len(t, val, 1)
	require.InEpsilon(t, 42.0, val[0], 0.001)
}

func TestResolveJoinInputs_parallelRandToSum_mergesFloat64Slice(t *testing.T) {
	sumMeta := &packages.FunctionMetadata{
		Input: packages.FunctionInputMetadata{
			CustomParameters: false,
			Parameters: map[string]pkgworkflow.ParameterSchema{
				"values": {Name: "values", Type: "[]float64", Required: true},
			},
		},
	}
	randMeta := &packages.FunctionMetadata{
		Output: packages.FunctionOutputMetadata{
			Parameters: map[string]pkgworkflow.ParameterSchema{
				"rand": {Name: "rand", Type: "int"},
			},
		},
	}

	sumNode := &Node{
		schema:           &NodeSchema{ID: "logic-sum", Function: "fuse/pkg/logic/sum"},
		functionMetadata: sumMeta,
	}
	r1 := &Node{
		schema:           &NodeSchema{ID: "logic-rand-1", Function: "fuse/pkg/logic/rand"},
		functionMetadata: randMeta,
		thread:           1,
	}
	r2 := &Node{
		schema:           &NodeSchema{ID: "logic-rand-2", Function: "fuse/pkg/logic/rand"},
		functionMetadata: randMeta,
		thread:           2,
	}

	e1 := &Edge{id: "e1", from: r1, to: sumNode, schema: &EdgeSchema{
		ID: "e1",
		Input: []InputMapping{{
			Source: SourceFlow, Variable: "logic-rand-1.rand", MapTo: "values",
		}},
	}}
	e2 := &Edge{id: "e2", from: r2, to: sumNode, schema: &EdgeSchema{
		ID: "e2",
		Input: []InputMapping{{
			Source: SourceFlow, Variable: "logic-rand-2.rand", MapTo: "values",
		}},
	}}
	sumNode.AddInputEdge(e1)
	sumNode.AddInputEdge(e2)

	out := store.New()
	out.Set("logic-rand-1.rand", 10)
	out.Set("logic-rand-2.rand", 32)

	wf := &Workflow{aggregatedOutput: out}
	merged := wf.resolveJoinInputs(sumNode)

	val, ok := merged["values"].([]float64)
	require.True(t, ok, "got %T", merged["values"])
	require.Len(t, val, 2)
	require.InEpsilon(t, 10.0, val[0], 0.001)
	require.InEpsilon(t, 32.0, val[1], 0.001)
}
