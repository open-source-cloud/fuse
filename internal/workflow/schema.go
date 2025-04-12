package workflow

import "github.com/open-source-cloud/fuse/internal/graph"

type Schema interface {
	ID() string
	RootNode() graph.Node
	FindNodeByIndex(index int) graph.Node
}

type schema struct {
	id    string
	graph graph.Graph
}

func LoadSchema(id string, graph graph.Graph) Schema {
	return &schema{
		id:    id,
		graph: graph,
	}
}

func (s *schema) ID() string {
	return s.id
}

func (s *schema) RootNode() graph.Node {
	return s.graph.Root()
}

func (s *schema) FindNodeByIndex(index int) graph.Node {
	return s.graph.Root()
}
