package ai

import (
	"testing"

	"github.com/open-source-cloud/fuse/pkg/workflow"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMangleDemangleToolName_RoundTrip(t *testing.T) {
	t.Parallel()

	cases := []struct {
		id      string
		mangled string
	}{
		{"system/sleep", "system__sleep"},
		{"fuse/pkg/logic/sum", "fuse__pkg__logic__sum"},
		{"fuse/pkg/ai/agent", "fuse__pkg__ai__agent"},
		{"noslash", "noslash"},
	}
	for _, tc := range cases {
		assert.Equal(t, tc.mangled, MangleToolName(tc.id), "mangle %q", tc.id)
		assert.Equal(t, tc.id, DemangleToolName(tc.mangled), "demangle %q", tc.mangled)
	}
}

func TestJSONSchemaType_Mapping(t *testing.T) {
	t.Parallel()

	cases := map[string]string{
		"string":    "string",
		"int":       "integer",
		"int64":     "integer",
		"float":     "number",
		"float64":   "number",
		"bool":      "boolean",
		"map":       "object",
		"object":    "object",
		"array":     "array",
		"slice":     "array",
		"[]float64": "array",
		"[]string":  "array",
		"weirdtype": "string", // unknown -> string
	}
	for in, want := range cases {
		assert.Equalf(t, want, jsonSchemaType(in), "jsonSchemaType(%q)", in)
	}
}

func TestParameterSchemaToJSONSchema_BuildsObject(t *testing.T) {
	t.Parallel()

	params := []workflow.ParameterSchema{
		{Name: "values", Type: "[]float64", Required: true, Description: "Values to sum", Default: []int{}},
		{Name: "label", Type: "string", Required: false},
		{Name: "count", Type: "int", Required: true},
	}

	schema := ParameterSchemaToJSONSchema(params)

	assert.Equal(t, "object", schema["type"])

	props, ok := schema["properties"].(map[string]any)
	require.True(t, ok)
	require.Len(t, props, 3)

	values, ok := props["values"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "array", values["type"])
	assert.Equal(t, "Values to sum", values["description"])
	assert.NotNil(t, values["default"])

	count, ok := props["count"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "integer", count["type"])

	// required is sorted and contains only the required params.
	required, ok := schema["required"].([]string)
	require.True(t, ok)
	assert.Equal(t, []string{"count", "values"}, required)
}

func TestParameterSchemaToJSONSchema_EmptyParams(t *testing.T) {
	t.Parallel()

	schema := ParameterSchemaToJSONSchema(nil)
	assert.Equal(t, "object", schema["type"])
	props, ok := schema["properties"].(map[string]any)
	require.True(t, ok)
	assert.Empty(t, props)
	_, hasRequired := schema["required"]
	assert.False(t, hasRequired, "no required key when there are no required params")
}
