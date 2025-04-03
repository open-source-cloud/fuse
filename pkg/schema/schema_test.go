package schema

import (
	"encoding/json"
	"testing"
)

func TestLoadSchemaFromJSON(t *testing.T) {
	// Test valid schema
	validJSON := `{
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

	schema, err := LoadSchemaFromJSON(validJSON)
	if err != nil {
		t.Errorf("Failed to load valid schema: %v", err)
	}
	if schema.ID != "test-schema" {
		t.Errorf("Expected schema ID 'test-schema', got '%s'", schema.ID)
	}

	// Test invalid schema (missing required field)
	invalidJSON := `{
		"name": "Invalid Schema",
		"version": "1.0",
		"fields": []
	}`

	_, err = LoadSchemaFromJSON(invalidJSON)
	if err == nil {
		t.Error("Expected error for invalid schema, got nil")
	}

	// Test invalid JSON
	_, err = LoadSchemaFromJSON("{invalid json")
	if err == nil {
		t.Error("Expected error for invalid JSON, got nil")
	}
}

func TestValidateSchemaDefinition(t *testing.T) {
	// Test valid schema
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

	err := ValidateSchemaDefinition(validSchema)
	if err != nil {
		t.Errorf("Failed to validate valid schema: %v", err)
	}

	// Test missing ID
	invalidSchema := &Schema{
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

	err = ValidateSchemaDefinition(invalidSchema)
	if err == nil {
		t.Error("Expected error for schema with missing ID, got nil")
	}

	// Test missing fields
	invalidSchema = &Schema{
		ID:      "invalid-schema",
		Name:    "Invalid Schema",
		Version: "1.0",
		Fields:  []Field{},
	}

	err = ValidateSchemaDefinition(invalidSchema)
	if err == nil {
		t.Error("Expected error for schema with no fields, got nil")
	}

	// Test duplicate field IDs
	invalidSchema = &Schema{
		ID:      "invalid-schema",
		Name:    "Invalid Schema",
		Version: "1.0",
		Fields: []Field{
			{
				ID:       "field1",
				Name:     "Field 1",
				Type:     TypeString,
				Required: true,
			},
			{
				ID:       "field1", // Duplicate ID
				Name:     "Field 2",
				Type:     TypeString,
				Required: true,
			},
		},
	}

	err = ValidateSchemaDefinition(invalidSchema)
	if err == nil {
		t.Error("Expected error for schema with duplicate field IDs, got nil")
	}
}

func TestValidate(t *testing.T) {
	// Create a schema for testing
	schema := Schema{
		ID:      "test-schema",
		Name:    "Test Schema",
		Version: "1.0",
		Fields: []Field{
			{
				ID:       "name",
				Name:     "Name",
				Type:     TypeString,
				Required: true,
				Validation: []ValidationRule{
					{
						Type:      "min_length",
						Value:     float64(3),
						Message:   "Name must be at least 3 characters long",
						ErrorCode: "name_too_short",
					},
				},
			},
			{
				ID:       "age",
				Name:     "Age",
				Type:     TypeInteger,
				Required: true,
				Validation: []ValidationRule{
					{
						Type:      "min_value",
						Value:     float64(18),
						Message:   "Must be at least 18 years old",
						ErrorCode: "age_too_young",
					},
				},
			},
			{
				ID:       "email",
				Name:     "Email",
				Type:     TypeString,
				Required: false,
				Validation: []ValidationRule{
					{
						Type:      "pattern",
						Value:     "^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\\.[a-zA-Z]{2,}$",
						Message:   "Invalid email format",
						ErrorCode: "email_invalid_format",
					},
				},
			},
			{
				ID:       "preferences",
				Name:     "Preferences",
				Type:     TypeObject,
				Required: false,
				Properties: []Field{
					{
						ID:       "theme",
						Name:     "Theme",
						Type:     TypeString,
						Required: true,
						Validation: []ValidationRule{
							{
								Type:      "enum",
								Value:     []interface{}{"light", "dark"},
								Message:   "Theme must be either 'light' or 'dark'",
								ErrorCode: "theme_invalid",
							},
						},
					},
				},
			},
		},
	}

	// Test valid data
	validData := map[string]interface{}{
		"name":  "John Doe",
		"age":   float64(25),
		"email": "john@example.com",
		"preferences": map[string]interface{}{
			"theme": "dark",
		},
	}

	result := schema.Validate(validData)
	if !result.Valid {
		errorsJSON, _ := json.Marshal(result.Errors)
		t.Errorf("Expected valid data to pass validation, but got errors: %s", errorsJSON)
	}

	// Test missing required field
	invalidData := map[string]interface{}{
		"name": "John Doe",
		// Missing 'age' field
	}

	result = schema.Validate(invalidData)
	if result.Valid {
		t.Error("Expected invalid data to fail validation, but it passed")
	}

	// Test invalid field value (min_length)
	invalidData = map[string]interface{}{
		"name": "Jo", // Too short
		"age":  float64(25),
	}

	result = schema.Validate(invalidData)
	if result.Valid {
		t.Error("Expected invalid name length to fail validation, but it passed")
	}

	// Test invalid field value (min_value)
	invalidData = map[string]interface{}{
		"name": "John Doe",
		"age":  float64(16), // Too young
	}

	result = schema.Validate(invalidData)
	if result.Valid {
		t.Error("Expected invalid age to fail validation, but it passed")
	}

	// Test invalid pattern
	invalidData = map[string]interface{}{
		"name":  "John Doe",
		"age":   float64(25),
		"email": "not-an-email", // Invalid email format
	}

	result = schema.Validate(invalidData)
	if result.Valid {
		t.Error("Expected invalid email to fail validation, but it passed")
	}

	// Test invalid nested object
	invalidData = map[string]interface{}{
		"name": "John Doe",
		"age":  float64(25),
		"preferences": map[string]interface{}{
			"theme": "blue", // Invalid theme
		},
	}

	result = schema.Validate(invalidData)
	if result.Valid {
		t.Error("Expected invalid theme to fail validation, but it passed")
	}
}
