// Package graph provides a memory graph implementation
package graph

import (
	"fmt"
	"github.com/open-source-cloud/fuse/internal/graph/schema"
	"github.com/rs/zerolog/log"
)

// NewGraphFromSchema creates a new Graph from a schema definition
func NewGraphFromSchema(graphDef *schema.Graph) (Graph, error) {
	newGraph := &graph{
		nodes: make(map[string]Node),
		edges: make(map[string]Edge),
	}
	err := newGraph.processGraphFromSchema(graphDef)
	if err != nil {
		log.Error().Err(err).Msg("Failed to create graph from schema")
		return nil, err
	}
	return newGraph, nil
}

type (
	// ParentNodeWithCondition represents a parent node with a condition to be added
	ParentNodeWithCondition struct {
		NodeID    string
		Condition *schema.EdgeCondition
		input     []schema.InputMapping
	}
	// Graph is the interface for a graph
	Graph interface {
		Root() Node
		FindNode(nodeID string) (Node, error)
		AddNode(parentNodeID string, edgeID string, node Node, condition *schema.EdgeCondition, input []schema.InputMapping) error
		AddNodeMultipleParents(parentNodeIDs []ParentNodeWithCondition, edgeID string, node Node) error
	}

	graph struct {
		root  Node
		nodes map[string]Node
		edges map[string]Edge
	}
)

// Root returns the root node of the graph
func (g *graph) Root() Node {
	return g.root
}

// FindNode finds a node by ID
func (g *graph) FindNode(nodeID string) (Node, error) {
	if g.root.ID() == nodeID {
		return g.root, nil
	}

	if nodeRef, ok := g.nodes[nodeID]; ok {
		return nodeRef, nil
	}

	return nil, fmt.Errorf("node %s not found", nodeID)
}

// AddNode adds a node to the graph
func (g *graph) AddNode(parentNodeID string, edgeID string, node Node, condition *schema.EdgeCondition, input []schema.InputMapping) error {
	parentNode, err := g.FindNode(parentNodeID)
	if err != nil {
		return err
	}
	newEdge := NewEdge(edgeID, parentNode, node, condition, input)
	parentNode.AddOutputEdge(edgeID, newEdge)
	node.AddInputEdge(newEdge)
	return nil
}

// AddNodeMultipleParents adds a node to the graph with multiple parents
func (g *graph) AddNodeMultipleParents(parentNodeIDs []ParentNodeWithCondition, edgeID string, node Node) error {
	for _, parent := range parentNodeIDs {
		err := g.AddNode(parent.NodeID, edgeID, node, parent.Condition, parent.input)
		if err != nil {
			return err
		}
	}
	return nil
}

func (g *graph) processGraphFromSchema(schemaDef *schema.Graph) error {
	for i, nodeDef := range schemaDef.Nodes {
		newNode := NewNode(nodeDef)
		g.nodes[newNode.ID()] = newNode
		if i == 0 {
			g.root = newNode
		}
	}
	for _, edgeDef := range schemaDef.Edges {
		fromNode, err := g.FindNode(edgeDef.From)
		if err != nil {
			return err
		}
		toNode, err := g.FindNode(edgeDef.To)
		if err != nil {
			return err
		}
		newEdge := NewEdge(edgeDef.ID, fromNode, toNode, edgeDef.Conditional, edgeDef.Input)
		g.edges[newEdge.ID()] = newEdge
		fromNode.AddOutputEdge(newEdge.ID(), newEdge)
		toNode.AddInputEdge(newEdge)
	}

	return nil
}
