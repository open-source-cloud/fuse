package workflow

type Schema struct {
	ID string
}

type Instance interface {
	ID() string
	Schema() *Schema
}

type DefaultInstance struct {
	id     string
	schema *Schema
	graph  Graph
}

func NewDefaultInstance(id string, schema *Schema, graph Graph) *DefaultInstance {
	return &DefaultInstance{
		id:     id,
		schema: schema,
		graph:  graph,
	}
}
func (w *DefaultInstance) ID() string {
	return w.id
}
func (w *DefaultInstance) Schema() *Schema {
	return w.schema
}
