package schema

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/open-source-cloud/fuse/internal/workflow"
)

// ValidatorConfig represents the configuration for a schema validator node
type ValidatorConfig struct {
	SchemaID   string `json:"schema_id"`
	SchemaPath string `json:"schema_path"`
}

// ValidatorNode implements the workflow.Node interface for schema validation
type ValidatorNode struct {
	id     string
	config ValidatorConfig
	schema *Schema
}

// NewSchemaValidatorProvider creates a new provider for schema validation nodes
func NewSchemaValidatorProvider() workflow.NodeProvider {
	return &schemaValidatorProvider{}
}

type schemaValidatorProvider struct{}

func (p *schemaValidatorProvider) Name() string {
	return "schema_validator"
}

func (p *schemaValidatorProvider) CreateNode(config interface{}) (workflow.Node, error) {
	// Convert config to map if it's not already
	configMap, ok := config.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid config type for schema validator")
	}

	// Extract schema ID or path
	schemaID, hasID := configMap["schema_id"].(string)
	schemaPath, hasPath := configMap["schema_path"].(string)

	if !hasID && !hasPath {
		return nil, fmt.Errorf("either schema_id or schema_path is required for schema validator")
	}

	nodeConfig := ValidatorConfig{
		SchemaID:   schemaID,
		SchemaPath: schemaPath,
	}

	// Create a unique ID for the node
	var nodeID string
	if hasID {
		nodeID = fmt.Sprintf("schema_validator_%s", schemaID)
	} else {
		nodeID = fmt.Sprintf("schema_validator_%s", schemaPath)
	}

	// Load the schema if path is provided
	var schema *Schema
	var err error
	if hasPath {
		schema, err = LoadSchemaFromFile(schemaPath)
		if err != nil {
			return nil, fmt.Errorf("failed to load schema from file: %w", err)
		}
	}

	return &ValidatorNode{
		id:     nodeID,
		config: nodeConfig,
		schema: schema,
	}, nil
}

func (p *schemaValidatorProvider) ValidateConfig(config interface{}) error {
	cfg, ok := config.(ValidatorConfig)
	if !ok {
		return fmt.Errorf("invalid config type for schema validator")
	}

	if cfg.SchemaID == "" && cfg.SchemaPath == "" {
		return fmt.Errorf("either schema_id or schema_path is required")
	}

	return nil
}

// ID returns the unique identifier of the schema validator node
func (n *ValidatorNode) ID() string {
	return n.id
}

// Execute validates the input data against the schema
func (n *ValidatorNode) Execute(_ context.Context, input interface{}) (interface{}, error) {
	// If schema is not loaded yet, load it based on ID
	if n.schema == nil && n.config.SchemaID != "" {
		// Implement schema loading by ID (e.g., from a registry)
		return nil, fmt.Errorf("schema loading by ID is not implemented yet")
	}

	if n.schema == nil {
		return nil, fmt.Errorf("no schema available for validation")
	}

	// Validate the input against the schema
	result := n.schema.Validate(input)

	// If validation failed, return an error
	if !result.Valid {
		errorsJSON, _ := json.Marshal(result.Errors)
		return nil, fmt.Errorf("schema validation failed: %s", errorsJSON)
	}

	// If validation succeeded, return the input data (pass-through)
	return input, nil
}

// Validate checks if the schema validator node is properly configured
func (n *ValidatorNode) Validate() error {
	if n.config.SchemaID == "" && n.config.SchemaPath == "" {
		return fmt.Errorf("either schema_id or schema_path is required")
	}

	// If path is provided, check if the schema can be loaded
	if n.config.SchemaPath != "" && n.schema == nil {
		schema, err := LoadSchemaFromFile(n.config.SchemaPath)
		if err != nil {
			return fmt.Errorf("failed to load schema from file: %w", err)
		}
		n.schema = schema
	}

	return nil
}
