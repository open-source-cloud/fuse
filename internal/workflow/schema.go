package workflow

import "github.com/open-source-cloud/fuse/internal/graph"

// NewSchema creates a new Schema object
func NewSchema(id string, graph graph.Graph) Schema {
	return &schema{
		id:    id,
		graph: graph,
	}
}

type Schema interface {
	ID() string
	Graph() graph.Graph
	RootNode() graph.Node
}

type schema struct {
	id    string
	graph graph.Graph
}

func (s *schema) ID() string {
	return s.id
}

func (s *schema) Graph() graph.Graph {
	return s.graph
}

func (s *schema) RootNode() graph.Node {
	return s.graph.Root()
}
