// Package graph provides a graph interface
package graph

// Graph is the interface for a graph
type (
	ParentNodeWithCondition struct {
		NodeID    string
		Condition *EdgeCondition
	}

	Graph interface {
		Root() Node
		FindNode(nodeID string) (Node, error)
		AddNode(parentNodeID string, edgeID string, node Node, condition *EdgeCondition) error
		AddNodeMultipleParents(parentNodeIDs []ParentNodeWithCondition, edgeID string, node Node) error
	}
)
