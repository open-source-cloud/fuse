package workflow

const DefaultOutputSchema = "default"

type Node interface {
	ID() string
	InputSchema() *DataSchema
	OutputSchemas(name string) *DataSchema
	Execute(input map[string]any) (interface{}, error)
}
