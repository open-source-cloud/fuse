package workflow

// EdgeConditionType classifies how the condition is evaluated
type EdgeConditionType string

const (
	// ConditionExact matches edge value exactly (current behavior)
	ConditionExact EdgeConditionType = "exact"
	// ConditionExpression evaluates an expr-lang expression
	ConditionExpression EdgeConditionType = "expression"
	// ConditionDefault matches when no other condition on the same node matches
	ConditionDefault EdgeConditionType = "default"
)

const (
	// SourceSchema source data from the workflow graph schema
	SourceSchema InputMappingSource = "schema"
	// SourceFlow source data from workflow
	SourceFlow InputMappingSource = "flow"
)

type (
	// InputMappingSource defines an enum for supported input mapping sources
	InputMappingSource string
	// EdgeSchema represents the schema for an edge in a graph with an ID, source, destination, and optional metadata.
	EdgeSchema struct {
		ID          string         `json:"id" yaml:"id" validate:"required"`
		From        string         `json:"from" yaml:"from" validate:"required"`
		To          string         `json:"to" yaml:"to" validate:"required"`
		Conditional *EdgeCondition `json:"conditional,omitempty" yaml:"conditional,omitempty"`
		Input       []InputMapping `json:"input,omitempty" yaml:"input,omitempty"`
		OnError     bool           `json:"onError,omitempty" yaml:"onError,omitempty"`
	}
	// EdgeCondition represents a conditional configuration with a name and its associated value.
	EdgeCondition struct {
		Name       string            `json:"name" yaml:"name" validate:"required"`
		Type       EdgeConditionType `json:"type,omitempty" yaml:"type,omitempty"`
		Value      any               `json:"value,omitempty" yaml:"value,omitempty"`
		Expression string            `json:"expression,omitempty" yaml:"expression,omitempty"`
	}
	// InputMapping represents a mapping for node input, including source, origin of data, and target mapping name.
	InputMapping struct {
		Source   InputMappingSource `json:"source" yaml:"source" validate:"required"`
		Variable string             `json:"variable,omitempty" yaml:"variable,omitempty"`
		Value    any                `json:"value,omitempty" yaml:"value,omitempty"`
		MapTo    string             `json:"mapTo" yaml:"mapTo" validate:"required"`
	}
)

// Clone creates a deep copy of the EdgeSchema
func (e EdgeSchema) Clone() *EdgeSchema {
	inputs := make([]InputMapping, len(e.Input))
	for i, input := range e.Input {
		inputs[i] = input.Clone()
	}
	var conditional *EdgeCondition
	if e.Conditional != nil {
		conditional = e.Conditional.Clone()
	}
	return &EdgeSchema{
		ID:          e.ID,
		From:        e.From,
		To:          e.To,
		Conditional: conditional,
		Input:       inputs,
		OnError:     e.OnError,
	}
}

// Clone creates a deep copy of the EdgeCondition
func (e EdgeCondition) Clone() *EdgeCondition {
	return &EdgeCondition{
		Name:       e.Name,
		Type:       e.Type,
		Value:      e.Value,
		Expression: e.Expression,
	}
}

// Clone creates a deep copy of the InputMapping
func (e InputMapping) Clone() InputMapping {
	return InputMapping{
		Source:   e.Source,
		Variable: e.Variable,
		Value:    e.Value,
		MapTo:    e.MapTo,
	}
}
