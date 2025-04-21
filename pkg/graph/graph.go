// Package graph provides a graph interface
package graph

// Graph is the interface for a graph
type Graph interface {
	Root() Node
	FindNode(nodeID string) Node
	AddNode(parentNodeID string, edgeID string, node Node)
	AddNodeMultipleParents(parentNodeIDs []string, edgeID string, node Node)
}
