package workflow

import "github.com/open-source-cloud/fuse/internal/graph"

type Schema interface {
	ID() string
	Name() string
	Description() string
	Graph() graph.Graph
}

func LoadSchema(id string, graph graph.Graph) (Schema, error) {
	return &SchemaImpl{
		id:    id,
		graph: graph,
	}, nil
}

type SchemaImpl struct {
	id    string
	graph graph.Graph
}

func (s *SchemaImpl) ID() string {
	return s.id
}

func (s *SchemaImpl) Name() string {
	//TODO implement me
	panic("implement me")
}

func (s *SchemaImpl) Description() string {
	//TODO implement me
	panic("implement me")
}

func (s *SchemaImpl) Graph() graph.Graph {
	return s.graph
}
