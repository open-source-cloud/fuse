package actors

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEvaluateFilter_MatchingExpression(t *testing.T) {
	data := map[string]any{
		"schemaId": "my-schema",
		"status":   "finished",
	}

	matches, err := evaluateFilter(`schemaId == "my-schema"`, data)
	assert.NoError(t, err)
	assert.True(t, matches)
}

func TestEvaluateFilter_NonMatchingExpression(t *testing.T) {
	data := map[string]any{
		"schemaId": "other-schema",
		"status":   "finished",
	}

	matches, err := evaluateFilter(`schemaId == "my-schema"`, data)
	assert.NoError(t, err)
	assert.False(t, matches)
}

func TestEvaluateFilter_ComplexExpression(t *testing.T) {
	data := map[string]any{
		"schemaId": "orders",
		"status":   "error",
		"retries":  3,
	}

	matches, err := evaluateFilter(`status == "error" && retries > 2`, data)
	assert.NoError(t, err)
	assert.True(t, matches)
}

func TestEvaluateFilter_InvalidExpression(t *testing.T) {
	data := map[string]any{"key": "value"}

	_, err := evaluateFilter(`!!!invalid`, data)
	assert.Error(t, err)
}

func TestEvaluateFilter_MissingField(t *testing.T) {
	data := map[string]any{"other": "value"}

	// Accessing a missing field should cause an error or return false
	_, err := evaluateFilter(`schemaId == "test"`, data)
	// expr-lang returns error for undefined variables
	assert.Error(t, err)
}
