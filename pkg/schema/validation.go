package schema

import (
	"fmt"
	"reflect"
	"regexp"
)

const (
	// ValidationRuleMinLength is the validation rule for the minimum length of a string
	ValidationRuleMinLength = "min_length"
	// ValidationRuleMaxLength is the validation rule for the maximum length of a string
	ValidationRuleMaxLength = "max_length"
	// ValidationRulePattern is the validation rule for the pattern of a string
	ValidationRulePattern = "pattern"
	// ValidationRuleMin is the validation rule for the minimum value of a number
	ValidationRuleMin = "min"
	// ValidationRuleMax is the validation rule for the maximum value of a number
	ValidationRuleMax = "max"
	// ValidationRuleRequired is the validation rule for the required value of a field
	ValidationRuleRequired = "required"
	// ValidationRuleEnum is the validation rule for the enum value of a field
	ValidationRuleEnum = "enum"
	// ValidationRuleCustom is the validation rule for the custom value of a field
	ValidationRuleCustom = "custom"
	// ValidationRuleFormat is the validation rule for the format of a field
	ValidationRuleFormat = "format"
	// ValidationRuleUnique is the validation rule for the unique value of a field
	ValidationRuleUnique = "unique"
	// ValidationRuleMinItems is the validation rule for the minimum items of an array
	ValidationRuleMinItems = "min_items"
	// ValidationRuleMaxItems is the validation rule for the maximum items of an array
	ValidationRuleMaxItems = "max_items"
	// ValidationRuleMinProperties is the validation rule for the minimum properties of an object
	ValidationRuleMinProperties = "min_properties"
	// ValidationRuleMaxProperties is the validation rule for the maximum properties of an object
	ValidationRuleMaxProperties = "max_properties"
)

// ValidationRule represents a rule for field validation
type ValidationRule struct {
	Type      string      `json:"type"`
	Value     interface{} `json:"value"`
	Message   string      `json:"message,omitempty"`
	ErrorCode string      `json:"error_code,omitempty"`
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
	case ValidationRuleMinLength:
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

	case ValidationRuleMaxLength:
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

	case ValidationRulePattern:
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

	case ValidationRuleMin:
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

	case ValidationRuleMax:
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

	case ValidationRuleEnum:
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
