// Package schema provides functionality for defining and validating data schemas.
package schema

import (
	"encoding/json"
	"fmt"
	"reflect"
	"regexp"
)

// FieldType represents the type of a field in a schema
type FieldType string

const (
	// TypeString represents a string field
	TypeString FieldType = "string"
	// TypeInteger represents an integer field
	TypeInteger FieldType = "integer"
	// TypeFloat represents a floating-point number field
	TypeFloat FieldType = "float"
	// TypeBoolean represents a boolean field
	TypeBoolean FieldType = "boolean"
	// TypeObject represents a nested object field
	TypeObject FieldType = "object"
	// TypeArray represents an array field
	TypeArray FieldType = "array"
)

// ValidationRule represents a rule for field validation
type ValidationRule struct {
	Type      string      `json:"type"`
	Value     interface{} `json:"value"`
	Message   string      `json:"message,omitempty"`
	ErrorCode string      `json:"error_code,omitempty"`
}

// Field represents a single field in a schema
type Field struct {
	ID          string           `json:"id"`
	Name        string           `json:"name"`
	Description string           `json:"description,omitempty"`
	Type        FieldType        `json:"type"`
	Required    bool             `json:"required"`
	Default     interface{}      `json:"default,omitempty"`
	Validation  []ValidationRule `json:"validation,omitempty"`
	Properties  []Field          `json:"properties,omitempty"` // For object type
	Items       *Field           `json:"items,omitempty"`      // For array type
}

// Schema represents a complete data schema
type Schema struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Description string  `json:"description,omitempty"`
	Version     string  `json:"version"`
	Fields      []Field `json:"fields"`
}

// ValidationError represents an error during validation
type ValidationError struct {
	FieldID   string `json:"field_id"`
	Message   string `json:"message"`
	ErrorCode string `json:"error_code,omitempty"`
}

// ValidationResult represents the result of a validation
type ValidationResult struct {
	Valid  bool              `json:"valid"`
	Errors []ValidationError `json:"errors,omitempty"`
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
		return fmt.Errorf("schema ID is required")
	}

	if schema.Name == "" {
		return fmt.Errorf("schema name is required")
	}

	if schema.Version == "" {
		return fmt.Errorf("schema version is required")
	}

	if len(schema.Fields) == 0 {
		return fmt.Errorf("schema must have at least one field")
	}

	fieldIDs := make(map[string]bool)
	for _, field := range schema.Fields {
		if err := validateFieldDefinition(field, fieldIDs); err != nil {
			return err
		}
	}

	return nil
}

// validateFieldDefinition validates that a field is properly defined
func validateFieldDefinition(field Field, fieldIDs map[string]bool) error {
	if field.ID == "" {
		return fmt.Errorf("field ID is required")
	}

	if field.Name == "" {
		return fmt.Errorf("field name is required")
	}

	if _, exists := fieldIDs[field.ID]; exists {
		return fmt.Errorf("duplicate field ID: %s", field.ID)
	}
	fieldIDs[field.ID] = true

	switch field.Type {
	case TypeString, TypeInteger, TypeFloat, TypeBoolean:
		// Basic types are valid
	case TypeObject:
		if len(field.Properties) == 0 {
			return fmt.Errorf("object field %s must have properties", field.ID)
		}

		propertyIDs := make(map[string]bool)
		for _, property := range field.Properties {
			if err := validateFieldDefinition(property, propertyIDs); err != nil {
				return fmt.Errorf("in field %s: %w", field.ID, err)
			}
		}
	case TypeArray:
		if field.Items == nil {
			return fmt.Errorf("array field %s must have items definition", field.ID)
		}

		itemIDs := make(map[string]bool)
		if err := validateFieldDefinition(*field.Items, itemIDs); err != nil {
			return fmt.Errorf("in field %s items: %w", field.ID, err)
		}
	default:
		return fmt.Errorf("unknown field type: %s", field.Type)
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

// validateField validates a single field in the data
func validateField(field Field, data map[string]interface{}, parentPath string, result *ValidationResult) {
	fieldPath := field.ID
	if parentPath != "" {
		fieldPath = parentPath + "." + field.ID
	}

	value, exists := data[field.ID]

	// Check required fields
	if field.Required && !exists {
		result.Valid = false
		result.Errors = append(result.Errors, ValidationError{
			FieldID:   fieldPath,
			Message:   fmt.Sprintf("Field '%s' is required", field.Name),
			ErrorCode: "required_field",
		})
		return
	}

	// If field doesn't exist and is not required, nothing to validate
	if !exists {
		return
	}

	// Type validation
	valid := validateType(field.Type, value)
	if !valid {
		result.Valid = false
		result.Errors = append(result.Errors, ValidationError{
			FieldID:   fieldPath,
			Message:   fmt.Sprintf("Field '%s' has invalid type, expected %s", field.Name, field.Type),
			ErrorCode: "invalid_type",
		})
		return
	}

	// Apply validation rules
	for _, rule := range field.Validation {
		if !applyValidationRule(rule, field, value) {
			message := rule.Message
			if message == "" {
				message = fmt.Sprintf("Validation failed for field '%s'", field.Name)
			}

			errorCode := rule.ErrorCode
			if errorCode == "" {
				errorCode = "validation_error"
			}

			result.Valid = false
			result.Errors = append(result.Errors, ValidationError{
				FieldID:   fieldPath,
				Message:   message,
				ErrorCode: errorCode,
			})
		}
	}

	// Validate nested fields
	if field.Type == TypeObject && value != nil {
		nestedData, ok := value.(map[string]interface{})
		if ok {
			for _, property := range field.Properties {
				validateField(property, nestedData, fieldPath, result)
			}
		}
	} else if field.Type == TypeArray && value != nil && field.Items != nil {
		arrayValue, ok := value.([]interface{})
		if ok {
			for i, item := range arrayValue {
				if itemMap, ok := item.(map[string]interface{}); ok {
					validateField(*field.Items, itemMap, fmt.Sprintf("%s[%d]", fieldPath, i), result)
				}
			}
		}
	}
}

// validateType checks if a value matches the expected type
func validateType(fieldType FieldType, value interface{}) bool {
	if value == nil {
		return true // Nil values are validated by required check
	}

	switch fieldType {
	case TypeString:
		_, ok := value.(string)
		return ok
	case TypeInteger:
		// In JSON, numbers might be parsed as float64
		num, ok := value.(float64)
		if !ok {
			return false
		}
		// Check if it's an integer
		return num == float64(int(num))
	case TypeFloat:
		_, ok := value.(float64)
		return ok
	case TypeBoolean:
		_, ok := value.(bool)
		return ok
	case TypeObject:
		_, ok := value.(map[string]interface{})
		return ok
	case TypeArray:
		_, ok := value.([]interface{})
		return ok
	default:
		return false
	}
}

// applyValidationRule applies a validation rule to a field value
func applyValidationRule(rule ValidationRule, field Field, value interface{}) bool {
	switch rule.Type {
	case "min_length":
		if field.Type != TypeString {
			return true
		}
		strValue, ok := value.(string)
		if !ok {
			return false
		}
		minLength, ok := rule.Value.(float64)
		if !ok {
			return false
		}
		return len(strValue) >= int(minLength)

	case "max_length":
		if field.Type != TypeString {
			return true
		}
		strValue, ok := value.(string)
		if !ok {
			return false
		}
		maxLength, ok := rule.Value.(float64)
		if !ok {
			return false
		}
		return len(strValue) <= int(maxLength)

	case "pattern":
		if field.Type != TypeString {
			return true
		}
		strValue, ok := value.(string)
		if !ok {
			return false
		}
		pattern, ok := rule.Value.(string)
		if !ok {
			return false
		}
		match, err := regexp.MatchString(pattern, strValue)
		if err != nil {
			return false
		}
		return match

	case "min_value":
		if field.Type != TypeInteger && field.Type != TypeFloat {
			return true
		}
		numValue, ok := value.(float64)
		if !ok {
			return false
		}
		minValue, ok := rule.Value.(float64)
		if !ok {
			return false
		}
		return numValue >= minValue

	case "max_value":
		if field.Type != TypeInteger && field.Type != TypeFloat {
			return true
		}
		numValue, ok := value.(float64)
		if !ok {
			return false
		}
		maxValue, ok := rule.Value.(float64)
		if !ok {
			return false
		}
		return numValue <= maxValue

	case "enum":
		// Validate against a list of allowed values
		enumValues, ok := rule.Value.([]interface{})
		if !ok {
			return false
		}

		for _, enumValue := range enumValues {
			if reflect.DeepEqual(value, enumValue) {
				return true
			}
		}
		return false

	default:
		// Unknown validation rule type
		return true
	}
}
