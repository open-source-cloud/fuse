package schema

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestSchemaValidatorProvider(t *testing.T) {
	provider := NewSchemaValidatorProvider()

	if provider.Name() != "schema_validator" {
		t.Errorf("Expected provider name 'schema_validator', got '%s'", provider.Name())
	}

	// Create a temporary schema file for testing
	tempDir, err := os.MkdirTemp("", "schema-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer func() {
		if err := os.RemoveAll(tempDir); err != nil {
			log.Printf("Failed to remove temp directory: %v", err)
		}
	}()

	schemaPath := filepath.Join(tempDir, "test-schema.json")
	schemaJSON := `{
		"id": "test-schema",
		"name": "Test Schema",
		"version": "1.0",
		"fields": [
			{
				"id": "field1",
				"name": "Field 1",
				"type": "string",
				"required": true
			}
		]
	}`

	if err := os.WriteFile(schemaPath, []byte(schemaJSON), 0600); err != nil {
		t.Fatalf("Failed to write schema file: %v", err)
	}

	// Test creating a node with valid config
	config := map[string]interface{}{
		"schema_path": schemaPath,
	}

	node, err := provider.CreateNode(config)
	if err != nil {
		t.Errorf("Failed to create node: %v", err)
	}

	if node == nil {
		t.Fatal("Expected node to be created, but got nil")
	}

	// Test node ID
	expectedID := "schema_validator_" + schemaPath
	if node.ID() != expectedID {
		t.Errorf("Expected node ID '%s', got '%s'", expectedID, node.ID())
	}

	// Test validation
	if err := node.Validate(); err != nil {
		t.Errorf("Node validation failed: %v", err)
	}

	// Test execution with valid data
	validData := map[string]interface{}{
		"field1": "value1",
	}

	result, err := node.Execute(context.Background(), validData)
	if err != nil {
		t.Errorf("Execution failed with valid data: %v", err)
	}

	// The result should be the input data (pass-through)
	if !reflect.DeepEqual(result, validData) {
		t.Errorf("Expected result to be the input data, but got %v", result)
	}

	// Test execution with invalid data
	invalidData := map[string]interface{}{
		// Missing required field1
	}

	_, err = node.Execute(context.Background(), invalidData)
	if err == nil {
		t.Error("Expected execution to fail with invalid data, but it succeeded")
	}

	// Test creating a node with invalid config
	invalidConfig := map[string]interface{}{
		// Missing schema_path and schema_id
	}

	_, err = provider.CreateNode(invalidConfig)
	if err == nil {
		t.Error("Expected CreateNode to fail with invalid config, but it succeeded")
	}
}
