package workflow

type Workflow interface {
	Schema() Schema
}

type DefaultWorkflow struct {
	schema Schema
}

// NewDefaultWorkflow creates and returns a new instance of DefaultWorkflow.
func NewDefaultWorkflow(schema Schema) *DefaultWorkflow {
	return &DefaultWorkflow{
		schema: schema,
	}
}
