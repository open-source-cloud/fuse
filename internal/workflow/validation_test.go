package workflow

import (
	"testing"

	"github.com/open-source-cloud/fuse/pkg/workflow"
	"github.com/stretchr/testify/assert"
)

func TestValidateInputMapping(t *testing.T) {
	tests := []struct {
		name    string
		schema  *workflow.ParameterSchema
		value   any
		wantErr bool
	}{
		// nil schema
		{
			name:    "nil schema always passes",
			schema:  nil,
			value:   "anything",
			wantErr: false,
		},
		// required field checks
		{
			name:    "required field with nil value fails",
			schema:  &workflow.ParameterSchema{Name: "name", Type: "string", Required: true},
			value:   nil,
			wantErr: true,
		},
		{
			name:    "optional field with nil value passes",
			schema:  &workflow.ParameterSchema{Name: "name", Type: "string", Required: false},
			value:   nil,
			wantErr: false,
		},
		// string type
		{
			name:    "string type with string value passes",
			schema:  &workflow.ParameterSchema{Name: "s", Type: "string"},
			value:   "hello",
			wantErr: false,
		},
		{
			name:    "string type with int value fails",
			schema:  &workflow.ParameterSchema{Name: "s", Type: "string"},
			value:   42,
			wantErr: true,
		},
		// int type
		{
			name:    "int type with int value passes",
			schema:  &workflow.ParameterSchema{Name: "n", Type: "int"},
			value:   42,
			wantErr: false,
		},
		{
			name:    "int type with float64 whole number passes (JSON number coercion)",
			schema:  &workflow.ParameterSchema{Name: "n", Type: "int"},
			value:   float64(42),
			wantErr: false,
		},
		{
			name:    "int type with float64 fractional fails",
			schema:  &workflow.ParameterSchema{Name: "n", Type: "int"},
			value:   42.5,
			wantErr: true,
		},
		{
			name:    "int type with string value fails",
			schema:  &workflow.ParameterSchema{Name: "n", Type: "int"},
			value:   "not a number",
			wantErr: true,
		},
		// float64 type
		{
			name:    "float64 type with float64 value passes",
			schema:  &workflow.ParameterSchema{Name: "f", Type: "float64"},
			value:   3.14,
			wantErr: false,
		},
		{
			name:    "float64 type with int value passes (coercion)",
			schema:  &workflow.ParameterSchema{Name: "f", Type: "float64"},
			value:   42,
			wantErr: false,
		},
		{
			name:    "float64 type with string value fails",
			schema:  &workflow.ParameterSchema{Name: "f", Type: "float64"},
			value:   "not a float",
			wantErr: true,
		},
		// bool type
		{
			name:    "bool type with bool value passes",
			schema:  &workflow.ParameterSchema{Name: "b", Type: "bool"},
			value:   true,
			wantErr: false,
		},
		{
			name:    "bool type with string value fails",
			schema:  &workflow.ParameterSchema{Name: "b", Type: "bool"},
			value:   "true",
			wantErr: true,
		},
		// map type
		{
			name:    "map type with map value passes",
			schema:  &workflow.ParameterSchema{Name: "m", Type: "map"},
			value:   map[string]any{"key": "value"},
			wantErr: false,
		},
		{
			name:    "map type with map[string]string passes (headers-like)",
			schema:  &workflow.ParameterSchema{Name: "headers", Type: "map"},
			value:   map[string]string{"Authorization": "Bearer x"},
			wantErr: false,
		},
		{
			name:    "map type with non-string keys fails",
			schema:  &workflow.ParameterSchema{Name: "m", Type: "map"},
			value:   map[int]string{1: "a"},
			wantErr: true,
		},
		{
			name:    "map type with string value fails",
			schema:  &workflow.ParameterSchema{Name: "m", Type: "map"},
			value:   "not a map",
			wantErr: true,
		},
		// any type
		{
			name:    "any type with string passes",
			schema:  &workflow.ParameterSchema{Name: "a", Type: "any"},
			value:   "hello",
			wantErr: false,
		},
		{
			name:    "any type with int passes",
			schema:  &workflow.ParameterSchema{Name: "a", Type: "any"},
			value:   42,
			wantErr: false,
		},
		// array types
		{
			name:    "array type with slice value passes",
			schema:  &workflow.ParameterSchema{Name: "arr", Type: "[]int"},
			value:   []int{1, 2, 3},
			wantErr: false,
		},
		{
			name:    "array type with non-slice value fails",
			schema:  &workflow.ParameterSchema{Name: "arr", Type: "[]int"},
			value:   42,
			wantErr: true,
		},
		{
			name:    "array type with any slice passes",
			schema:  &workflow.ParameterSchema{Name: "arr", Type: "[]any"},
			value:   []any{1, "two", 3.0},
			wantErr: false,
		},
		// unknown type
		{
			name:    "unknown type skips validation",
			schema:  &workflow.ParameterSchema{Name: "x", Type: "customType"},
			value:   "anything",
			wantErr: false,
		},
		// empty type
		{
			name:    "empty type skips validation",
			schema:  &workflow.ParameterSchema{Name: "x", Type: ""},
			value:   42,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Act
			err := ValidateInputMapping(tt.schema, tt.value)

			// Assert
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
