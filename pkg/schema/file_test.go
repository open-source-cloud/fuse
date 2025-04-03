package schema

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/suite"
)

// FileSuite defines a test suite for schema file operations
type FileSuite struct {
	suite.Suite
	tempDir string
}

func TestFileSuite(t *testing.T) {
	suite.Run(t, new(FileSuite))
}

// SetupSuite creates a temporary directory for file operations
func (s *FileSuite) SetupSuite() {
	tempDir, err := os.MkdirTemp("", "schema-file-test")
	s.Require().NoError(err, "Failed to create temp directory")
	s.tempDir = tempDir
}

// TearDownSuite removes the temporary directory
func (s *FileSuite) TearDownSuite() {
	if s.tempDir != "" {
		if err := os.RemoveAll(s.tempDir); err != nil {
			s.T().Logf("Failed to remove temp directory: %v", err)
		}
	}
}

func (s *FileSuite) TestLoadSchemaFromFile() {
	// Create a valid schema file
	validSchema := &Schema{
		ID:      "test-schema",
		Name:    "Test Schema",
		Version: "1.0",
		Fields: []Field{
			{
				ID:       "field1",
				Name:     "Field 1",
				Type:     TypeString,
				Required: true,
			},
		},
	}

	validSchemaPath := filepath.Join(s.tempDir, "valid-schema.json")
	err := SaveSchemaToFile(validSchema, validSchemaPath)
	s.Require().NoError(err, "Failed to save valid schema")

	// Test loading a valid schema
	loadedSchema, err := LoadSchemaFromFile(validSchemaPath)
	s.NoError(err, "Should load valid schema without error")
	s.Equal("test-schema", loadedSchema.ID, "Loaded schema should have correct ID")
	s.Equal("Test Schema", loadedSchema.Name, "Loaded schema should have correct name")
	s.Equal("1.0", loadedSchema.Version, "Loaded schema should have correct version")
	s.Len(loadedSchema.Fields, 1, "Loaded schema should have one field")
	s.Equal("field1", loadedSchema.Fields[0].ID, "Loaded schema field should have correct ID")

	// Test loading non-existent file
	_, err = LoadSchemaFromFile(filepath.Join(s.tempDir, "non-existent.json"))
	s.Error(err, "Should return error for non-existent file")
	s.Contains(err.Error(), "failed to read schema file", "Error should mention read failure")

	// Test loading invalid JSON
	invalidJSONPath := filepath.Join(s.tempDir, "invalid.json")
	err = os.WriteFile(invalidJSONPath, []byte("{invalid json"), 0600)
	s.Require().NoError(err, "Failed to create invalid JSON file")

	_, err = LoadSchemaFromFile(invalidJSONPath)
	s.Error(err, "Should return error for invalid JSON")
	s.Contains(err.Error(), "failed to parse schema", "Error should mention parse failure")

	// Test loading invalid schema (missing required field)
	invalidSchemaPath := filepath.Join(s.tempDir, "invalid-schema.json")
	err = os.WriteFile(invalidSchemaPath, []byte(`{
		"name": "Invalid Schema",
		"version": "1.0",
		"fields": [
			{
				"id": "field1",
				"name": "Field 1",
				"type": "string",
				"required": true
			}
		]
	}`), 0600)
	s.Require().NoError(err, "Failed to create invalid schema file")

	_, err = LoadSchemaFromFile(invalidSchemaPath)
	s.Error(err, "Should return error for invalid schema")
	s.Contains(err.Error(), "invalid schema definition", "Error should mention schema validation")
}

func (s *FileSuite) TestSaveSchemaToFile() {
	// Create a schema to save
	schema := &Schema{
		ID:      "save-test-schema",
		Name:    "Save Test Schema",
		Version: "1.0",
		Fields: []Field{
			{
				ID:       "field1",
				Name:     "Field 1",
				Type:     TypeString,
				Required: true,
			},
		},
	}

	// Test saving to a new file
	schemaPath := filepath.Join(s.tempDir, "saved-schema.json")
	err := SaveSchemaToFile(schema, schemaPath)
	s.NoError(err, "Should save schema without error")
	s.FileExists(schemaPath, "Schema file should exist after saving")

	// Verify saved content by loading it back
	loadedSchema, err := LoadSchemaFromFile(schemaPath)
	s.NoError(err, "Should load saved schema without error")
	s.Equal(schema.ID, loadedSchema.ID, "Loaded schema should match original")

	// Test saving to a nested directory that doesn't exist
	nestedPath := filepath.Join(s.tempDir, "nested", "dirs", "schema.json")
	err = SaveSchemaToFile(schema, nestedPath)
	s.NoError(err, "Should create directories and save schema")
	s.FileExists(nestedPath, "Schema file should exist in nested directory")

	// Test saving an invalid schema
	invalidSchema := &Schema{
		// Missing ID
		Name:    "Invalid Schema",
		Version: "1.0",
		Fields: []Field{
			{
				ID:       "field1",
				Name:     "Field 1",
				Type:     TypeString,
				Required: true,
			},
		},
	}

	err = SaveSchemaToFile(invalidSchema, filepath.Join(s.tempDir, "invalid-schema.json"))
	s.Error(err, "Should return error for invalid schema")
	s.Contains(err.Error(), "invalid schema definition", "Error should mention schema validation")
}

func (s *FileSuite) TestListSchemasInDirectory() {
	// Create a subdirectory for this test
	testDir := filepath.Join(s.tempDir, "list-test")
	err := os.MkdirAll(testDir, 0750)
	s.Require().NoError(err, "Failed to create test directory")

	// Create some schema files
	schema1Path := filepath.Join(testDir, "schema1.json")
	schema2Path := filepath.Join(testDir, "schema2.json")
	nonSchemaPath := filepath.Join(testDir, "not-a-schema.txt")

	schema := &Schema{
		ID:      "list-test-schema",
		Name:    "List Test Schema",
		Version: "1.0",
		Fields: []Field{
			{
				ID:       "field1",
				Name:     "Field 1",
				Type:     TypeString,
				Required: true,
			},
		},
	}

	err = SaveSchemaToFile(schema, schema1Path)
	s.Require().NoError(err, "Failed to save schema1")

	err = SaveSchemaToFile(schema, schema2Path)
	s.Require().NoError(err, "Failed to save schema2")

	err = os.WriteFile(nonSchemaPath, []byte("not a schema"), 0600)
	s.Require().NoError(err, "Failed to create non-schema file")

	// Create a nested directory with another schema
	nestedDir := filepath.Join(testDir, "nested")
	err = os.MkdirAll(nestedDir, 0750)
	s.Require().NoError(err, "Failed to create nested directory")

	nestedSchemaPath := filepath.Join(nestedDir, "nested-schema.json")
	err = SaveSchemaToFile(schema, nestedSchemaPath)
	s.Require().NoError(err, "Failed to save nested schema")

	// Test listing schemas
	schemas, err := ListSchemasInDirectory(testDir)
	s.NoError(err, "Should list schemas without error")
	s.Len(schemas, 3, "Should find 3 schema files")

	// Verify all schema files are found
	foundSchema1 := false
	foundSchema2 := false
	foundNestedSchema := false

	for _, path := range schemas {
		switch path {
		case schema1Path:
			foundSchema1 = true
		case schema2Path:
			foundSchema2 = true
		case nestedSchemaPath:
			foundNestedSchema = true
		}
	}

	s.True(foundSchema1, "Should find schema1.json")
	s.True(foundSchema2, "Should find schema2.json")
	s.True(foundNestedSchema, "Should find nested-schema.json")

	// Test listing from non-existent directory
	_, err = ListSchemasInDirectory(filepath.Join(s.tempDir, "non-existent"))
	s.Error(err, "Should return error for non-existent directory")
	s.Contains(err.Error(), "directory does not exist", "Error should mention directory not found")
}
