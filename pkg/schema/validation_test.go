package schema

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

// ValidationSuite defines a test suite for schema validation functionality
type ValidationSuite struct {
	suite.Suite
}

func TestValidationSuite(t *testing.T) {
	suite.Run(t, new(ValidationSuite))
}

func (s *ValidationSuite) TestValidateField() {
	// Create a test setup with various validation rules
	s.Run("RequiredField", func() {
		field := Field{
			ID:       "required_field",
			Name:     "Required Field",
			Type:     TypeString,
			Required: true,
		}

		// Test with field present
		data := map[string]interface{}{
			"required_field": "value",
		}
		result := ValidationResult{
			Valid:  true,
			Errors: []ValidationError{},
		}

		validateField(field, data, "", &result)
		s.True(result.Valid, "Validation should pass with required field present")
		s.Empty(result.Errors, "No errors should be reported")

		// Test with field missing
		data = map[string]interface{}{}
		result = ValidationResult{
			Valid:  true,
			Errors: []ValidationError{},
		}

		validateField(field, data, "", &result)
		s.False(result.Valid, "Validation should fail with required field missing")
		s.Len(result.Errors, 1, "One error should be reported")
		s.Equal("required_field", result.Errors[0].FieldID, "Error should reference the correct field")
		s.Contains(result.Errors[0].Message, "required", "Error message should mention field is required")
	})

	s.Run("TypeValidation", func() {
		// Test string type validation
		stringField := Field{
			ID:       "string_field",
			Name:     "String Field",
			Type:     TypeString,
			Required: false,
		}

		// Valid string value
		data := map[string]interface{}{
			"string_field": "valid string",
		}
		result := ValidationResult{
			Valid:  true,
			Errors: []ValidationError{},
		}

		validateField(stringField, data, "", &result)
		s.True(result.Valid, "Validation should pass for valid string")

		// Invalid type (number instead of string)
		data = map[string]interface{}{
			"string_field": 123,
		}
		result = ValidationResult{
			Valid:  true,
			Errors: []ValidationError{},
		}

		validateField(stringField, data, "", &result)
		s.False(result.Valid, "Validation should fail for wrong type")
		s.Len(result.Errors, 1, "One error should be reported")
		s.Contains(result.Errors[0].Message, "invalid type", "Error should mention invalid type")

		// Test integer type validation
		intField := Field{
			ID:       "int_field",
			Name:     "Integer Field",
			Type:     TypeInteger,
			Required: false,
		}

		// Valid integer value (as float64 from JSON)
		data = map[string]interface{}{
			"int_field": float64(42),
		}
		result = ValidationResult{
			Valid:  true,
			Errors: []ValidationError{},
		}

		validateField(intField, data, "", &result)
		s.True(result.Valid, "Validation should pass for valid integer")

		// Invalid integer (float with decimal part)
		data = map[string]interface{}{
			"int_field": float64(42.5),
		}
		result = ValidationResult{
			Valid:  true,
			Errors: []ValidationError{},
		}

		validateField(intField, data, "", &result)
		s.False(result.Valid, "Validation should fail for non-integer number")
	})

	s.Run("ValidationRules", func() {
		// Field with min length validation
		minLengthField := Field{
			ID:       "name",
			Name:     "Name",
			Type:     TypeString,
			Required: true,
			Validation: []ValidationRule{
				{
					Type:      ValidationRuleMinLength,
					Value:     float64(3),
					Message:   "Name must be at least 3 characters",
					ErrorCode: "name_too_short",
				},
			},
		}

		// Valid length
		data := map[string]interface{}{
			"name": "John",
		}
		result := ValidationResult{
			Valid:  true,
			Errors: []ValidationError{},
		}

		validateField(minLengthField, data, "", &result)
		s.True(result.Valid, "Validation should pass for valid length")

		// Invalid length
		data = map[string]interface{}{
			"name": "Jo",
		}
		result = ValidationResult{
			Valid:  true,
			Errors: []ValidationError{},
		}

		validateField(minLengthField, data, "", &result)
		s.False(result.Valid, "Validation should fail for too short string")
		s.Equal("name_too_short", result.Errors[0].ErrorCode, "Error code should match")
		s.Equal("Name must be at least 3 characters", result.Errors[0].Message, "Error message should match")
	})

	s.Run("NestedObjectValidation", func() {
		// Define a schema with nested object
		nestedField := Field{
			ID:       "user",
			Name:     "User",
			Type:     TypeObject,
			Required: true,
			Properties: []Field{
				{
					ID:       "username",
					Name:     "Username",
					Type:     TypeString,
					Required: true,
					Validation: []ValidationRule{
						{
							Type:      ValidationRuleMinLength,
							Value:     float64(3),
							Message:   "Username must be at least 3 characters",
							ErrorCode: "username_too_short",
						},
					},
				},
				{
					ID:       "age",
					Name:     "Age",
					Type:     TypeInteger,
					Required: false,
					Validation: []ValidationRule{
						{
							Type:      ValidationRuleMin,
							Value:     float64(18),
							Message:   "User must be at least 18 years old",
							ErrorCode: "user_too_young",
						},
					},
				},
			},
		}

		// Valid nested object
		data := map[string]interface{}{
			"user": map[string]interface{}{
				"username": "johndoe",
				"age":      float64(25),
			},
		}
		result := ValidationResult{
			Valid:  true,
			Errors: []ValidationError{},
		}

		validateField(nestedField, data, "", &result)
		s.True(result.Valid, "Validation should pass for valid nested object")

		// Invalid nested object - missing required field
		data = map[string]interface{}{
			"user": map[string]interface{}{
				"age": float64(25),
				// missing username
			},
		}
		result = ValidationResult{
			Valid:  true,
			Errors: []ValidationError{},
		}

		validateField(nestedField, data, "", &result)
		s.False(result.Valid, "Validation should fail for missing required nested field")
		s.Contains(result.Errors[0].FieldID, "user.username", "Error should reference the nested field")

		// Invalid nested object - validation rule failure
		data = map[string]interface{}{
			"user": map[string]interface{}{
				"username": "johndoe",
				"age":      float64(16), // too young
			},
		}
		result = ValidationResult{
			Valid:  true,
			Errors: []ValidationError{},
		}

		validateField(nestedField, data, "", &result)
		s.False(result.Valid, "Validation should fail for invalid nested field value")
		s.Contains(result.Errors[0].FieldID, "user.age", "Error should reference the nested field")
		s.Equal("user_too_young", result.Errors[0].ErrorCode, "Error code should match")
	})

	s.Run("ArrayValidation", func() {
		// Create basic array fields for testing
		arrayField := Field{
			ID:       "items",
			Name:     "Items",
			Type:     TypeArray,
			Required: true,
		}

		// Test valid array
		data := map[string]interface{}{
			"items": []interface{}{"item1", "item2"},
		}
		result := ValidationResult{
			Valid:  true,
			Errors: []ValidationError{},
		}

		validateField(arrayField, data, "", &result)
		s.True(result.Valid, "Valid array should pass validation")
		s.Empty(result.Errors, "No errors should be present for valid array")

		// Test wrong type (non-array)
		data = map[string]interface{}{
			"items": "not an array",
		}
		result = ValidationResult{
			Valid:  true,
			Errors: []ValidationError{},
		}

		validateField(arrayField, data, "", &result)
		s.False(result.Valid, "Non-array value should fail validation")
		s.NotEmpty(result.Errors, "Errors should be present for non-array value")
		s.Equal("items", result.Errors[0].FieldID, "Error should reference the array field")
		s.Contains(result.Errors[0].Message, "invalid type", "Error message should mention invalid type")

		// Test missing required array
		data = map[string]interface{}{}
		result = ValidationResult{
			Valid:  true,
			Errors: []ValidationError{},
		}

		validateField(arrayField, data, "", &result)
		s.False(result.Valid, "Missing required array should fail validation")
		s.NotEmpty(result.Errors, "Errors should be present for missing field")
		s.Equal("items", result.Errors[0].FieldID, "Error should reference the array field")
		s.Contains(result.Errors[0].Message, "required", "Error message should mention required")

		// Test empty array
		data = map[string]interface{}{
			"items": []interface{}{},
		}
		result = ValidationResult{
			Valid:  true,
			Errors: []ValidationError{},
		}

		validateField(arrayField, data, "", &result)
		s.True(result.Valid, "Empty array should pass validation")
		s.Empty(result.Errors, "No errors should be present for empty array")

		// Optional array field
		optionalArrayField := Field{
			ID:       "optional_items",
			Name:     "Optional Items",
			Type:     TypeArray,
			Required: false,
		}

		// Test missing optional array
		data = map[string]interface{}{}
		result = ValidationResult{
			Valid:  true,
			Errors: []ValidationError{},
		}

		validateField(optionalArrayField, data, "", &result)
		s.True(result.Valid, "Missing optional array should pass validation")
		s.Empty(result.Errors, "No errors should be present for missing optional array")
	})
}

func (s *ValidationSuite) TestValidateType() {
	// Test String type
	s.True(validateType(TypeString, "valid string"), "String type should validate string value")
	s.False(validateType(TypeString, 123), "String type should reject non-string value")
	s.False(validateType(TypeString, true), "String type should reject boolean value")
	s.True(validateType(TypeString, nil), "nil should pass type validation (handled by required check)")

	// Test Integer type
	s.True(validateType(TypeInteger, float64(42)), "Integer type should validate integer value")
	s.False(validateType(TypeInteger, float64(42.5)), "Integer type should reject float with decimal part")
	s.False(validateType(TypeInteger, "42"), "Integer type should reject string value")
	s.True(validateType(TypeInteger, nil), "nil should pass type validation (handled by required check)")

	// Test Float type
	s.True(validateType(TypeFloat, float64(42.5)), "Float type should validate float value")
	s.True(validateType(TypeFloat, float64(42)), "Float type should validate integer value as float")
	s.False(validateType(TypeFloat, "42.5"), "Float type should reject string value")
	s.True(validateType(TypeFloat, nil), "nil should pass type validation (handled by required check)")

	// Test Boolean type
	s.True(validateType(TypeBoolean, true), "Boolean type should validate true value")
	s.True(validateType(TypeBoolean, false), "Boolean type should validate false value")
	s.False(validateType(TypeBoolean, "true"), "Boolean type should reject string value")
	s.False(validateType(TypeBoolean, 1), "Boolean type should reject number value")
	s.True(validateType(TypeBoolean, nil), "nil should pass type validation (handled by required check)")

	// Test Object type
	s.True(validateType(TypeObject, map[string]interface{}{"key": "value"}), "Object type should validate map value")
	s.False(validateType(TypeObject, []interface{}{}), "Object type should reject array value")
	s.False(validateType(TypeObject, "object"), "Object type should reject string value")
	s.True(validateType(TypeObject, nil), "nil should pass type validation (handled by required check)")

	// Test Array type
	s.True(validateType(TypeArray, []interface{}{"item1", "item2"}), "Array type should validate array value")
	s.False(validateType(TypeArray, map[string]interface{}{}), "Array type should reject map value")
	s.False(validateType(TypeArray, "array"), "Array type should reject string value")
	s.True(validateType(TypeArray, nil), "nil should pass type validation (handled by required check)")

	// Test invalid type
	s.False(validateType("invalid_type", "value"), "Invalid type should always return false")
}

func (s *ValidationSuite) TestApplyValidationRule() {
	// Test MinLength rule
	stringField := Field{
		ID:   "string_field",
		Type: TypeString,
	}
	minLengthRule := ValidationRule{
		Type:  ValidationRuleMinLength,
		Value: float64(3),
	}

	s.True(
		applyValidationRule(minLengthRule, stringField, "long enough"),
		"MinLength rule should pass for string longer than minimum",
	)
	s.False(
		applyValidationRule(minLengthRule, stringField, "ab"),
		"MinLength rule should fail for string shorter than minimum",
	)
	s.False(
		applyValidationRule(minLengthRule, stringField, 123),
		"MinLength rule should fail for non-string values",
	)

	// Test MaxLength rule
	maxLengthRule := ValidationRule{
		Type:  ValidationRuleMaxLength,
		Value: float64(5),
	}

	s.True(
		applyValidationRule(maxLengthRule, stringField, "short"),
		"MaxLength rule should pass for string shorter than maximum",
	)
	s.False(
		applyValidationRule(maxLengthRule, stringField, "too long"),
		"MaxLength rule should fail for string longer than maximum",
	)

	// Test Pattern rule
	patternRule := ValidationRule{
		Type:  ValidationRulePattern,
		Value: "^[a-z]+$",
	}

	s.True(
		applyValidationRule(patternRule, stringField, "lowercase"),
		"Pattern rule should pass for matching string",
	)
	s.False(
		applyValidationRule(patternRule, stringField, "UPPERCASE"),
		"Pattern rule should fail for non-matching string",
	)
	s.False(
		applyValidationRule(patternRule, stringField, "mixed123"),
		"Pattern rule should fail for non-matching string",
	)

	// Test Min rule
	numberField := Field{
		ID:   "number_field",
		Type: TypeInteger,
	}
	minRule := ValidationRule{
		Type:  ValidationRuleMin,
		Value: float64(10),
	}

	s.True(
		applyValidationRule(minRule, numberField, float64(15)),
		"Min rule should pass for number greater than minimum",
	)
	s.True(
		applyValidationRule(minRule, numberField, float64(10)),
		"Min rule should pass for number equal to minimum",
	)
	s.False(
		applyValidationRule(minRule, numberField, float64(5)),
		"Min rule should fail for number less than minimum",
	)
	s.False(
		applyValidationRule(minRule, numberField, "not a number"),
		"Min rule should fail for non-number values",
	)

	// Test Max rule
	maxRule := ValidationRule{
		Type:  ValidationRuleMax,
		Value: float64(100),
	}

	s.True(
		applyValidationRule(maxRule, numberField, float64(50)),
		"Max rule should pass for number less than maximum",
	)
	s.True(
		applyValidationRule(maxRule, numberField, float64(100)),
		"Max rule should pass for number equal to maximum",
	)
	s.False(
		applyValidationRule(maxRule, numberField, float64(150)),
		"Max rule should fail for number greater than maximum",
	)

	// Test Enum rule
	enumRule := ValidationRule{
		Type:  ValidationRuleEnum,
		Value: []interface{}{"red", "green", "blue"},
	}

	s.True(
		applyValidationRule(enumRule, stringField, "red"),
		"Enum rule should pass for value in enum list",
	)
	s.True(
		applyValidationRule(enumRule, stringField, "green"),
		"Enum rule should pass for value in enum list",
	)
	s.False(
		applyValidationRule(enumRule, stringField, "yellow"),
		"Enum rule should fail for value not in enum list",
	)

	// Test with invalid rule types
	invalidRule := ValidationRule{
		Type:  "unknown_rule",
		Value: "test",
	}

	s.True(
		applyValidationRule(invalidRule, stringField, "any value"),
		"Unknown validation rule should always pass",
	)
}
