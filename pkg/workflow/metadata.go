package workflow

import "github.com/open-source-cloud/fuse/pkg/transport"

// FunctionMetadata defines the metadata structure for a Function
type FunctionMetadata struct {
	Transport transport.Type `json:"transport"`
	Input     InputMetadata  `json:"input"`
	Output    OutputMetadata `json:"output,omitempty"`
}

// InputMetadata represents one Input or Result Metadata descriptor
type InputMetadata struct {
	// CustomParameters serves to indicate that the some parameters are schemaless and can be mapped directly from the input (raw data).
	// This is useful for functions like (logic/if), (logic/switch), (logic/for).
	// For using the raw values as conditions of the function expressions.
	CustomParameters bool              `json:"customParameters"`
	Parameters       []ParameterSchema `json:"parameters"`
	Edges            InputEdgeMetadata `json:"edges,omitempty"`
}

// OutputMetadata represents the output metadata for a node
type OutputMetadata struct {
	Parameters             []ParameterSchema    `json:"parameters"`
	ConditionalOutput      bool                 `json:"conditionalOutput"`
	ConditionalOutputField string               `json:"conditionalOutputField"`
	Edges                  []OutputEdgeMetadata `json:"edges,omitempty"`
}

// InputEdgeMetadata represents edge configuration for a node
type InputEdgeMetadata struct {
	Count      int               `json:"count"`
	Parameters []ParameterSchema `json:"parameters"`
}

// ConditionalEdgeMetadata represents additional metadata for a conditional edge
type ConditionalEdgeMetadata struct {
	Value any `json:"value"`
}

// OutputEdgeMetadata represents an output edge metadata configuration
type OutputEdgeMetadata struct {
	Name            string                  `json:"name"`
	ConditionalEdge ConditionalEdgeMetadata `json:"conditionalEdge"`
	Count           int                     `json:"count"`
	Parameters      []ParameterSchema       `json:"parameters"`
}

// ParameterSchema represents a schema definition for a single Data field.
// Each field in the schema can have specific properties like type, validation rules, and metadata.
//
// Validation array format:
// The Validations slice contains strings that specify rules for validation.
// Examples:
//
// - "min=18": Ensures the field has a minimum value of 18 (applicable to numeric types like int or float).
// - "Max=99": Ensures the field has a maximum value of 99 (applicable to numeric types like int or float).
// - "regex=^[a-zA-Z0-9]+$": Ensures the field matches a specific regular expression pattern (applicable to string types).
// - "len=10": Ensures the field is exactly 10 characters long (for strings or arrays).
// - "in=male,female,other": Ensures the field value is one of the specified allowed values.
// - "Required": Ensures the field is mandatory, though typically this is also expressed with the Required bool field.
// - "email": Ensures the field contains a valid email address (applicable to string types).
// - "uuid": Ensures the field contains a valid UUID value.
//
// Example usage:
// FieldName: "Username", Type: "string", Required: true, Validations: []string{"len=8", "regex=^[a-zA-Z]+$"}
// FieldName: "Age", Type: "int", Required: true, Validations: []string{"min=18", "max=65"}
type ParameterSchema struct {
	Name        string   `json:"name"`        // The variable name
	Type        string   `json:"type"`        // The variable type (e.g., string, int, bool, etc.)
	Required    bool     `json:"required"`    // Whether the field is mandatory
	Validations []string `json:"validations"` // A list of validations to apply (e.g., min, max, regex, etc.)
	Description string   `json:"description"` // Optional description of the field
	Default     any      `json:"default"`     // Default value if any
}
