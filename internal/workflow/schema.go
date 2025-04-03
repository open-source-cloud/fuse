package workflow

// YAMLWorkflow represents a workflow definition in YAML
type YAMLWorkflow struct {
	ID          string     `yaml:"id"`
	Name        string     `yaml:"name"`
	Description string     `yaml:"description"`
	Nodes       []YAMLNode `yaml:"nodes"`
	Edges       []YAMLEdge `yaml:"edges"`
}

// YAMLNode represents a node definition in YAML
type YAMLNode struct {
	ID       string                 `yaml:"id"`
	Type     string                 `yaml:"type"`     // e.g., "string_processor", "if", "switch"
	Provider string                 `yaml:"provider"` // e.g., "strproc", "logic"
	Config   map[string]interface{} `yaml:"config"`
}

// YAMLEdge represents an edge definition in YAML
type YAMLEdge struct {
	From      string                 `yaml:"from"`
	To        string                 `yaml:"to"`
	Condition map[string]interface{} `yaml:"condition,omitempty"`
}
