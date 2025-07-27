// Package mocks contains mock utilities for the fuse package.
package mocks

import (
	"github.com/open-source-cloud/fuse/internal/workflow"
)

// SmallTestGraphSchema returns a small test graph schema
// The debug-nil node is used to debug the graph.
// The logic-rand-1 node is used to generate a random number between 10 and 100.
// The logic-rand-2 node is used to generate a random number between 10 and 100.
// The logic-sum node is used to sum the two random numbers.
func SmallTestGraphSchema() *workflow.GraphSchema {
	return &workflow.GraphSchema{
		ID:   "test",
		Name: "test",
		Nodes: []*workflow.NodeSchema{
			{
				ID:       "debug-nil",
				Function: "fuse/pkg/debug/nil",
			},
			{
				ID:       "logic-rand-1",
				Function: "fuse/pkg/logic/rand",
			},
			{
				ID:       "logic-rand-2",
				Function: "fuse/pkg/logic/rand",
			},
			{
				ID:       "logic-sum",
				Function: "fuse/pkg/logic/sum",
			},
		},
		Edges: []*workflow.EdgeSchema{
			{
				ID:   "debug-nil-to-logic-rand-1",
				From: "debug-nil",
				To:   "logic-rand-1",
				Input: []workflow.InputMapping{
					{
						Source: "schema",
						Value:  10,
						MapTo:  "min",
					},
					{
						Source: "schema",
						Value:  100,
						MapTo:  "max",
					},
				},
			},
			{
				ID:   "debug-nil-to-logic-rand-2",
				From: "debug-nil",
				To:   "logic-rand-2",
				Input: []workflow.InputMapping{
					{
						Source: "schema",
						Value:  10,
						MapTo:  "min",
					},
					{
						Source: "schema",
						Value:  100,
						MapTo:  "max",
					},
				},
			},
			{
				ID:   "logic-rand-1-to-logic-sum",
				From: "logic-rand-1",
				To:   "logic-sum",
				Input: []workflow.InputMapping{
					{
						Source:   "flow",
						Variable: "logic-rand-1.rand",
						MapTo:    "values",
					},
				},
			},
			{
				ID:   "logic-rand-2-to-logic-sum",
				From: "logic-rand-2",
				To:   "logic-sum",
				Input: []workflow.InputMapping{
					{
						Source:   "flow",
						Variable: "logic-rand-2.rand",
						MapTo:    "values",
					},
				},
			},
		},
		Metadata: map[string]string{
			"test": "test",
		},
		Tags: map[string]string{
			"test": "test",
		},
	}
}
