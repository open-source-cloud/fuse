package workflow

import "fmt"

func NewMemorySchemaRepo() SchemaRepo {
	return &memorySchemaRepo{
		schemas: make(map[string]Schema),
	}
}

// SchemaRepo workflow schema repository
type (
	SchemaRepo interface {
		Get(id string) (Schema, error)
		Save(id string, workflowSchema Schema)
	}

	memorySchemaRepo struct {
		schemas map[string]Schema
	}
)

func (m *memorySchemaRepo) Get(id string) (Schema, error) {
	workflowSchema, ok := m.schemas[id]
	if !ok {
		return nil, fmt.Errorf("schema %s not found", id)
	}

	return workflowSchema, nil
}

func (m *memorySchemaRepo) Save(id string, workflowSchema Schema) {
	m.schemas[id] = workflowSchema
}
