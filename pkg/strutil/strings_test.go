package strutil_test

import (
	"testing"

	"github.com/open-source-cloud/fuse/pkg/strutil"
	"github.com/stretchr/testify/suite"
)

func TestStringsTestSuite(t *testing.T) {
	suite.Run(t, new(StringsTestSuite))
}

type StringsTestSuite struct {
	suite.Suite
}

func (s *StringsTestSuite) TestReplaceTokens() {
	tests := []struct {
		name     string
		input    string
		inputMap map[string]any
		expected string
	}{
		{
			name:     "simple token",
			input:    "Hello, {{name}}!",
			inputMap: map[string]any{"name": "world"},
			expected: "Hello, world!",
		},
		{
			name:     "multiple tokens",
			input:    "Hello, {{name}}! You are {{age}} years old.",
			inputMap: map[string]any{"name": "world", "age": 20},
			expected: "Hello, world! You are 20 years old.",
		},
		{
			name:     "missing token",
			input:    "Hello, {{name}}! You are {{age}} years old.",
			inputMap: map[string]any{"name": "world"},
			expected: "Hello, world! You are {{age}} years old.",
		},
		{
			name:     "empty input",
			input:    "Hello, {{name}}! You are {{age}} years old.",
			inputMap: map[string]any{},
			expected: "Hello, {{name}}! You are {{age}} years old.",
		},
	}

	for _, test := range tests {
		s.Run(test.name, func() {
			message := strutil.ReplaceTokens(test.input, test.inputMap)
			s.Equal(test.expected, message)
		})
	}
}
