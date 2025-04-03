package schema

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/stretchr/testify/suite"
)

// ProviderSuite defines a test suite for the schema validator provider
type ProviderSuite struct {
	suite.Suite
	tempDir string
}

func TestProviderSuite(t *testing.T) {
	suite.Run(t, new(ProviderSuite))
}

// SetupSuite sets up resources shared by all tests in the suite
func (s *ProviderSuite) SetupSuite() {
	// Create a temporary directory for test files
	tempDir, err := os.MkdirTemp("", "schema-test")
	s.Require().NoError(err, "Failed to create temp directory")
	s.tempDir = tempDir
}

// TearDownSuite cleans up resources after all tests have run
func (s *ProviderSuite) TearDownSuite() {
	if s.tempDir != "" {
		if err := os.RemoveAll(s.tempDir); err != nil {
			log.Printf("Failed to remove temp directory: %v", err)
		}
	}
}

func (s *ProviderSuite) TestSchemaValidatorProvider() {
	provider := NewSchemaValidatorProvider()

	s.Equal("schema_validator", provider.Name(), "Provider should have correct name")

	// Create a temporary schema file for testing
	schemaPath := filepath.Join(s.tempDir, "test-schema.json")
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

	err := os.WriteFile(schemaPath, []byte(schemaJSON), 0600)
	s.Require().NoError(err, "Failed to write schema file")

	// Test creating a node with valid config
	config := map[string]interface{}{
		"schema_path": schemaPath,
	}

	node, err := provider.CreateNode(config)
	s.NoError(err, "Should create node with valid config")
	s.NotNil(node, "Created node should not be nil")

	// Test node ID
	expectedID := "schema_validator_" + schemaPath
	s.Equal(expectedID, node.ID(), "Node should have correct ID")

	// Test validation
	s.NoError(node.Validate(), "Node validation should succeed")

	// Test execution with valid data
	validData := map[string]interface{}{
		"field1": "value1",
	}

	result, err := node.Execute(context.Background(), validData)
	s.NoError(err, "Execution should succeed with valid data")

	// The result should be the input data (pass-through)
	s.True(reflect.DeepEqual(result, validData), "Result should match input data")

	// Test execution with invalid data
	invalidData := map[string]interface{}{
		// Missing required field1
	}

	_, err = node.Execute(context.Background(), invalidData)
	s.Error(err, "Execution should fail with invalid data")

	// Test creating a node with invalid config
	invalidConfig := map[string]interface{}{
		// Missing schema_path and schema_id
	}

	_, err = provider.CreateNode(invalidConfig)
	s.Error(err, "CreateNode should fail with invalid config")
}
