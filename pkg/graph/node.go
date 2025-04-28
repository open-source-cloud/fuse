// Package graph provides a graph interface
package graph

import "github.com/open-source-cloud/fuse/pkg/workflow"

type (
	// NodeInputMapping provides the structure for node input mapping
	NodeInputMapping struct {
		Source  string
		Origin  any
		Mapping string
	}
	// NodeConfig describes a Node's configuration
	NodeConfig interface {
		InputMapping() []NodeInputMapping
		AddInputMapping(source string, origin any, mapping string)
	}
	// Node describes an executable Node object
	Node interface {
		ID() string
		NodeRef() workflow.Node
		Config() NodeConfig
		InputEdges() []Edge
		OutputEdges() map[string]Edge
		IsOutputConditional() bool
		AddInputEdge(edge Edge)
		AddOutputEdge(edgeID string, edge Edge)
	}
)
