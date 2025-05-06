package schema

type (
	// Edge represents the schema for an edge in a graph with an ID, source, destination, and optional metadata.
	Edge struct {
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
		Source   string `json:"source" yaml:"source"`
		Variable string `json:"origin,omitempty" yaml:"origin,omitempty"`
		Value    any    `json:"value,omitempty" yaml:"value,omitempty"`
		MapTo    string `json:"mapTo" yaml:"mapTo"`
	}
)
