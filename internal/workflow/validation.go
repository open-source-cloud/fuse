package workflow

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/open-source-cloud/fuse/pkg/workflow"
)

// ValidateInputMapping validates a value against a ParameterSchema.
// Returns nil if valid, or an error describing the validation failure.
func ValidateInputMapping(schema *workflow.ParameterSchema, value any) error {
	if schema == nil {
		return nil
	}

	if schema.Required && value == nil {
		return fmt.Errorf("required parameter %q is nil", schema.Name)
	}

	if value == nil {
		return nil
	}

	if schema.Type != "" {
		if err := validateType(schema.Type, value); err != nil {
			return fmt.Errorf("parameter %q: %w", schema.Name, err)
		}
	}

	return nil
}

func validateType(expected string, value any) error {
	if strings.HasPrefix(expected, "[]") {
		rv := reflect.ValueOf(value)
		if rv.Kind() != reflect.Slice && rv.Kind() != reflect.Array {
			return fmt.Errorf("expected %s, got %T", expected, value)
		}
		return nil
	}

	switch expected {
	case "string":
		if _, ok := value.(string); !ok {
			return fmt.Errorf("expected string, got %T", value)
		}
	case "int":
		if _, ok := toInt(value); !ok {
			return fmt.Errorf("expected int, got %T", value)
		}
	case "float64":
		if _, ok := toFloat64(value); !ok {
			return fmt.Errorf("expected float64, got %T", value)
		}
	case "bool":
		if _, ok := value.(bool); !ok {
			return fmt.Errorf("expected bool, got %T", value)
		}
	case "map":
		rv := reflect.ValueOf(value)
		if rv.Kind() != reflect.Map {
			return fmt.Errorf("expected map, got %T", value)
		}
		if rv.Type().Key().Kind() != reflect.String {
			return fmt.Errorf("expected map with string keys, got %T", value)
		}
	case "any":
		// any type always passes
	default:
		// unknown type — skip validation
	}
	return nil
}

// toInt attempts to coerce a value to int, handling JSON numbers (float64) and other int types.
func toInt(value any) (int, bool) {
	switch v := value.(type) {
	case int:
		return v, true
	case int8:
		return int(v), true
	case int16:
		return int(v), true
	case int32:
		return int(v), true
	case int64:
		return int(v), true
	case float64:
		if v == float64(int(v)) {
			return int(v), true
		}
		return 0, false
	case float32:
		if v == float32(int(v)) {
			return int(v), true
		}
		return 0, false
	default:
		return 0, false
	}
}

// toFloat64 attempts to coerce a value to float64, handling int types.
func toFloat64(value any) (float64, bool) {
	switch v := value.(type) {
	case float64:
		return v, true
	case float32:
		return float64(v), true
	case int:
		return float64(v), true
	case int8:
		return float64(v), true
	case int16:
		return float64(v), true
	case int32:
		return float64(v), true
	case int64:
		return float64(v), true
	default:
		return 0, false
	}
}
