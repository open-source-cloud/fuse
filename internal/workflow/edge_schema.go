package workflow

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
		ID          string         `json:"id" yaml:"id"`
		From        string         `json:"from" yaml:"from"`
		To          string         `json:"to" yaml:"to"`
		Conditional *EdgeCondition `json:"conditional,omitempty" yaml:"conditional,omitempty"`
		Input       []InputMapping `json:"input,omitempty" yaml:"input,omitempty"`
	}
	// EdgeCondition represents a conditional configuration with a name and its associated value.
	EdgeCondition struct {
		Name  string `json:"name" yaml:"name"`
		Value any    `json:"value" yaml:"value"`
	}
	// InputMapping represents a mapping for node input, including source, origin of data, and target mapping name.
	InputMapping struct {
		Source   InputMappingSource `json:"source" yaml:"source"`
		Variable string             `json:"variable,omitempty" yaml:"variable,omitempty"`
		Value    any                `json:"value,omitempty" yaml:"value,omitempty"`
		MapTo    string             `json:"mapTo" yaml:"mapTo"`
	}
)
