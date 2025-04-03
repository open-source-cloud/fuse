package schema

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

// FieldSuite defines a test suite for field-related functionality
type FieldSuite struct {
	suite.Suite
}

func TestFieldSuite(t *testing.T) {
	suite.Run(t, new(FieldSuite))
}

func (s *FieldSuite) TestValidateFieldDefinition() {
	// Test cases for valid field definitions
	s.Run("BasicValidField", func() {
		field := Field{
			ID:       "test_field",
			Name:     "Test Field",
			Type:     TypeString,
			Required: true,
		}

		fieldIDs := make(map[string]bool)
		err := validateFieldDefinition(field, fieldIDs)
		s.NoError(err, "Valid field should pass validation")
		s.True(fieldIDs["test_field"], "Field ID should be added to map")
	})

	s.Run("ValidObjectField", func() {
		field := Field{
			ID:       "object_field",
			Name:     "Object Field",
			Type:     TypeObject,
			Required: true,
			Properties: []Field{
				{
					ID:       "nested_field",
					Name:     "Nested Field",
					Type:     TypeString,
					Required: true,
				},
			},
		}

		fieldIDs := make(map[string]bool)
		err := validateFieldDefinition(field, fieldIDs)
		s.NoError(err, "Valid object field should pass validation")
	})

	s.Run("ValidArrayField", func() {
		nestedField := Field{
			ID:       "nested_field",
			Name:     "Nested Field",
			Type:     TypeString,
			Required: true,
		}

		field := Field{
			ID:       "array_field",
			Name:     "Array Field",
			Type:     TypeArray,
			Required: true,
			Items:    &nestedField,
		}

		fieldIDs := make(map[string]bool)
		err := validateFieldDefinition(field, fieldIDs)
		s.NoError(err, "Valid array field should pass validation")
	})

	// Test cases for invalid field definitions
	s.Run("MissingID", func() {
		field := Field{
			Name:     "Missing ID Field",
			Type:     TypeString,
			Required: true,
		}

		fieldIDs := make(map[string]bool)
		err := validateFieldDefinition(field, fieldIDs)
		s.Error(err, "Field with missing ID should fail validation")
		s.Equal(ErrFieldIDRequired, err, "Should return correct error type")
	})

	s.Run("MissingName", func() {
		field := Field{
			ID:       "missing_name",
			Type:     TypeString,
			Required: true,
		}

		fieldIDs := make(map[string]bool)
		err := validateFieldDefinition(field, fieldIDs)
		s.Error(err, "Field with missing name should fail validation")
		s.Equal(ErrFieldNameRequired, err, "Should return correct error type")
	})

	s.Run("DuplicateID", func() {
		fieldIDs := map[string]bool{"duplicate_id": true}
		field := Field{
			ID:       "duplicate_id",
			Name:     "Duplicate ID",
			Type:     TypeString,
			Required: true,
		}

		err := validateFieldDefinition(field, fieldIDs)
		s.Error(err, "Field with duplicate ID should fail validation")
		s.Contains(err.Error(), "duplicate field ID", "Error message should mention duplicate ID")
	})

	s.Run("InvalidType", func() {
		field := Field{
			ID:       "invalid_type",
			Name:     "Invalid Type",
			Type:     "invalid_type",
			Required: true,
		}

		fieldIDs := make(map[string]bool)
		err := validateFieldDefinition(field, fieldIDs)
		s.Error(err, "Field with invalid type should fail validation")
		s.Contains(err.Error(), "unknown field type", "Error message should mention unknown type")
	})

	s.Run("ObjectWithoutProperties", func() {
		field := Field{
			ID:         "empty_object",
			Name:       "Empty Object",
			Type:       TypeObject,
			Required:   true,
			Properties: []Field{}, // Empty properties array
		}

		fieldIDs := make(map[string]bool)
		err := validateFieldDefinition(field, fieldIDs)
		s.Error(err, "Object field without properties should fail validation")
		s.Contains(err.Error(), "must have properties", "Error message should mention missing properties")
	})

	s.Run("ArrayWithoutItems", func() {
		field := Field{
			ID:       "empty_array",
			Name:     "Empty Array",
			Type:     TypeArray,
			Required: true,
			Items:    nil, // Missing items definition
		}

		fieldIDs := make(map[string]bool)
		err := validateFieldDefinition(field, fieldIDs)
		s.Error(err, "Array field without items should fail validation")
		s.Contains(err.Error(), "must have items definition", "Error message should mention missing items definition")
	})

	s.Run("InvalidNestedObject", func() {
		field := Field{
			ID:       "invalid_nested",
			Name:     "Invalid Nested",
			Type:     TypeObject,
			Required: true,
			Properties: []Field{
				{
					ID:   "", // Missing ID in nested field
					Name: "Bad Nested Field",
					Type: TypeString,
				},
			},
		}

		fieldIDs := make(map[string]bool)
		err := validateFieldDefinition(field, fieldIDs)
		s.Error(err, "Object with invalid nested field should fail validation")
		s.Contains(err.Error(), "field ID is required", "Error message should identify the issue")
	})

	s.Run("InvalidArrayItems", func() {
		nestedField := Field{
			ID:   "", // Missing ID in items definition
			Name: "Bad Items Field",
			Type: TypeString,
		}

		field := Field{
			ID:       "invalid_items",
			Name:     "Invalid Items",
			Type:     TypeArray,
			Required: true,
			Items:    &nestedField,
		}

		fieldIDs := make(map[string]bool)
		err := validateFieldDefinition(field, fieldIDs)
		s.Error(err, "Array with invalid items definition should fail validation")
		s.Contains(err.Error(), "field ID is required", "Error message should identify the issue")
	})
}
