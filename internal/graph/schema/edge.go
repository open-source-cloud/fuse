package schema

type (
	// Edge represents the schema for an edge in a graph with an ID, source, destination, and optional metadata.
	Edge struct {
		ID          string         `json:"id" yaml:"id"`
		From        string         `json:"from" yaml:"from"`
		To          string         `json:"to" yaml:"to"`
		Conditional *EdgeCondition `json:"conditional,omitempty" yaml:"conditional,omitempty"`
	}
	// EdgeCondition represents a conditional configuration with a name and its associated value.
	EdgeCondition struct {
		Name  string `json:"name" yaml:"name"`
		Value any    `json:"value" yaml:"value"`
	}
)
