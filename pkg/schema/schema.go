// Package schema provides functionality for defining and validating data schemas.
package schema

import (
	"encoding/json"
	"errors"
	"fmt"
)

var (
	// ErrSchemaIDRequired is the error for a schema ID that is required
	ErrSchemaIDRequired = errors.New("schema ID is required")
	// ErrSchemaNameRequired is the error for a schema name that is required
	ErrSchemaNameRequired = errors.New("schema name is required")
	// ErrSchemaVersionRequired is the error for a schema version that is required
	ErrSchemaVersionRequired = errors.New("schema version is required")
	// ErrSchemaFieldsRequired is the error for a schema fields that is required
	ErrSchemaFieldsRequired = errors.New("schema must have at least one field")
)

// Schema represents a complete data schema
type Schema struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Description string  `json:"description,omitempty"`
	Version     string  `json:"version"`
	Fields      []Field `json:"fields"`
}

// LoadSchemaFromJSON loads a schema from a JSON string
func LoadSchemaFromJSON(jsonStr string) (*Schema, error) {
	var schema Schema
	if err := json.Unmarshal([]byte(jsonStr), &schema); err != nil {
		return nil, fmt.Errorf("failed to parse schema JSON: %w", err)
	}

	if err := ValidateSchemaDefinition(&schema); err != nil {
		return nil, err
	}

	return &schema, nil
}

// ValidateSchemaDefinition validates that a schema is properly defined
func ValidateSchemaDefinition(schema *Schema) error {
	if schema.ID == "" {
		return ErrSchemaIDRequired
	}

	if schema.Name == "" {
		return ErrSchemaNameRequired
	}

	if schema.Version == "" {
		return ErrSchemaVersionRequired
	}

	if len(schema.Fields) == 0 {
		return ErrSchemaFieldsRequired
	}

	fieldIDs := make(map[string]bool)
	for _, field := range schema.Fields {
		if err := validateFieldDefinition(field, fieldIDs); err != nil {
			return err
		}
	}

	return nil
}

// Validate validates data against the schema
func (s *Schema) Validate(data interface{}) ValidationResult {
	result := ValidationResult{
		Valid:  true,
		Errors: []ValidationError{},
	}

	// Convert data to map if it's not already
	dataMap, ok := data.(map[string]interface{})
	if !ok {
		result.Valid = false
		result.Errors = append(result.Errors, ValidationError{
			FieldID:   "",
			Message:   "data must be an object",
			ErrorCode: "invalid_data_format",
		})
		return result
	}

	// Validate each field
	for _, field := range s.Fields {
		validateField(field, dataMap, "", &result)
	}

	return result
}
