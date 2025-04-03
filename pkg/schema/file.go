package schema

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// LoadSchemaFromFile loads a schema from a JSON file
func LoadSchemaFromFile(filePath string) (*Schema, error) {
	// Read the file
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read schema file: %w", err)
	}

	// Parse the JSON
	var schema Schema
	if err := json.Unmarshal(data, &schema); err != nil {
		return nil, fmt.Errorf("failed to parse schema JSON: %w", err)
	}

	// Validate the schema
	if err := ValidateSchemaDefinition(&schema); err != nil {
		return nil, fmt.Errorf("invalid schema definition: %w", err)
	}

	return &schema, nil
}

// SaveSchemaToFile saves a schema to a JSON file
func SaveSchemaToFile(schema *Schema, filePath string) error {
	// Validate the schema
	if err := ValidateSchemaDefinition(schema); err != nil {
		return fmt.Errorf("invalid schema definition: %w", err)
	}

	// Marshal the schema to JSON
	data, err := json.MarshalIndent(schema, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal schema to JSON: %w", err)
	}

	// Ensure the directory exists
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0750); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Write the file
	if err := os.WriteFile(filePath, data, 0600); err != nil {
		return fmt.Errorf("failed to write schema file: %w", err)
	}

	return nil
}

// ListSchemasInDirectory lists all schema files in a directory
func ListSchemasInDirectory(dirPath string) ([]string, error) {
	// Ensure the directory exists
	if _, err := os.Stat(dirPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("directory does not exist: %w", err)
	}

	// Find all .json files
	var schemaFiles []string
	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && filepath.Ext(path) == ".json" {
			schemaFiles = append(schemaFiles, path)
		}
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to list schema files: %w", err)
	}

	return schemaFiles, nil
}
