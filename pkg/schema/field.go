package schema

import (
	"errors"
	"fmt"
)

const (
	// DuplicatedFieldMsg is the error message for a duplicated field ID
	DuplicatedFieldMsg = "duplicate field ID: %s"
	// UnknownFieldTypeMsg is the error message for an unknown field type
	UnknownFieldTypeMsg = "unknown field type: %s"
	// ObjectFieldMustHavePropertiesMsg is the error message for an object field that must have properties
	ObjectFieldMustHavePropertiesMsg = "object field %s must have properties"
	// ArrayFieldMustHaveItemsDefinitionMsg is the error message for an array field that must have items definition
	ArrayFieldMustHaveItemsDefinitionMsg = "array field %s must have items definition"
)

var (
	// ErrFieldIDRequired is the error for a field ID that is required
	ErrFieldIDRequired = errors.New("field ID is required")
	// ErrFieldNameRequired is the error for a field name that is required
	ErrFieldNameRequired = errors.New("field name is required")
	// ErrDuplicateFieldID is the error for a duplicated field ID
	ErrDuplicateFieldID = errors.New("duplicate field ID")
	// ErrUnknownFieldType is the error for an unknown field type
	ErrUnknownFieldType = errors.New("unknown field type")
)

// FieldType represents the type of a field in a schema
type FieldType string

// DefaultValue represents the default value of a field
type DefaultValue interface{}

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

// Field represents a single field in a schema
type Field struct {
	ID          string           `json:"id"`
	Name        string           `json:"name"`
	Description string           `json:"description,omitempty"`
	Type        FieldType        `json:"type"`
	Required    bool             `json:"required"`
	Default     DefaultValue     `json:"default,omitempty"`
	Validation  []ValidationRule `json:"validation,omitempty"`
	Properties  []Field          `json:"properties,omitempty"` // For object type
	Items       *Field           `json:"items,omitempty"`      // For array type
}

// validateFieldDefinition validates that a field is properly defined
func validateFieldDefinition(field Field, fieldIDs map[string]bool) error {
	if field.ID == "" {
		return ErrFieldIDRequired
	}

	if field.Name == "" {
		return ErrFieldNameRequired
	}

	if _, exists := fieldIDs[field.ID]; exists {
		return fmt.Errorf(DuplicatedFieldMsg, field.ID)
	}
	fieldIDs[field.ID] = true

	switch field.Type {
	case TypeString, TypeInteger, TypeFloat, TypeBoolean:
		// Basic types are valid
	case TypeObject:
		if len(field.Properties) == 0 {
			return fmt.Errorf(ObjectFieldMustHavePropertiesMsg, field.ID)
		}

		propertyIDs := make(map[string]bool)
		for _, property := range field.Properties {
			if err := validateFieldDefinition(property, propertyIDs); err != nil {
				return fmt.Errorf("in field %s: %w", field.ID, err)
			}
		}
	case TypeArray:
		if field.Items == nil {
			return fmt.Errorf(ArrayFieldMustHaveItemsDefinitionMsg, field.ID)
		}
		itemIDs := make(map[string]bool)
		if err := validateFieldDefinition(*field.Items, itemIDs); err != nil {
			return fmt.Errorf("in field %s items: %w", field.ID, err)
		}
	default:
		return fmt.Errorf(UnknownFieldTypeMsg, field.Type)
	}

	return nil
}
