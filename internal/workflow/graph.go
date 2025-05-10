package workflow

import (
	"fmt"
	"gopkg.in/yaml.v3"
)

func NewGraphSchemaFromJSON(jsonSpec []byte) (*GraphSchema, error) {
	var schema GraphSchema
	err := yaml.Unmarshal(jsonSpec, &schema)
	if err != nil {
		return nil, err
	}
	return &schema, nil
}

func NewGraphSchemaFromYAML(yamlSpec []byte) (*GraphSchema, error) {
	var schema GraphSchema
	err := yaml.Unmarshal(yamlSpec, &schema)
	if err != nil {
		return nil, err
	}
	return &schema, nil
}

func NewGraphFromJSON(jsonSpec []byte) (*Graph, error) {
	schema, err := NewGraphSchemaFromJSON(jsonSpec)
	if err != nil {
		return nil, err
	}
	return NewGraphFromSchema(schema)
}

func NewGraphFromYAML(yamlSpec []byte) (*Graph, error) {
	schema , err := NewGraphSchemaFromYAML(yamlSpec)
	if err != nil {
		return nil, err
	}
	return NewGraphFromSchema(schema)
}

func NewGraphFromSchema(schema *GraphSchema) (*Graph, error) {
	graph := &Graph{
		schema: schema,
		nodes:  make(map[string]*Node),
		edges:  make(map[string]*Edge),
	}

	for i, nodeDef := range schema.Nodes {
		node := newNode(nodeDef)
		graph.nodes[node.ID()] = node
		if i == 0 {
			graph.root = node
		}
	}
	for _, edgeDef := range schema.Edges {
		fromNode, err := graph.FindNode(edgeDef.From)
		if err != nil {
			return nil, err
		}
		toNode, err := graph.FindNode(edgeDef.To)
		if err != nil {
			return nil, err
		}
		edge := newEdge(edgeDef.ID, fromNode, toNode, edgeDef)
		graph.edges[edge.ID()] = edge
		fromNode.AddOutputEdge(edge.ID(), edge)
		toNode.AddInputEdge(edge)
	}

	return graph, nil
}

type (
	Graph struct {
		root   *Node
		schema *GraphSchema
		nodes  map[string]*Node
		edges  map[string]*Edge
	}
)

func (g *Graph) ID() string {
	return g.schema.ID
}

// Root returns the root Node of the Graph
func (g *Graph) Root() *Node {
	return g.root
}

// FindNode finds a Node by ID
func (g *Graph) FindNode(nodeID string) (*Node, error) {
	if g.root.ID() == nodeID {
		return g.root, nil
	}

	if nodeRef, ok := g.nodes[nodeID]; ok {
		return nodeRef, nil
	}

	return nil, fmt.Errorf("node %s not found", nodeID)
}
