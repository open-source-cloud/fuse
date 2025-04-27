// Package graph provides a graph interface
package graph

type (
	// ParentNodeWithCondition represents a parent node with a condition to be added
	ParentNodeWithCondition struct {
		NodeID    string
		Condition *EdgeCondition
	}

	// Graph is the interface for a graph
	Graph interface {
		Root() Node
		FindNode(nodeID string) (Node, error)
		AddNode(parentNodeID string, edgeID string, node Node, condition *EdgeCondition) error
		AddNodeMultipleParents(parentNodeIDs []ParentNodeWithCondition, edgeID string, node Node) error
	}
)
