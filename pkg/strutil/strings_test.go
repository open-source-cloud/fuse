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

func (s *StringsTestSuite) TestAfterFirstDot() {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{name: "simple string without dot", input: "hello world", expected: "hello world"},
		{name: "string with dot tow consecutive dots", input: "hello..world.test", expected: ".world.test"},
		{name: "string with dot and space", input: "hello world test.", expected: ""},
		{name: "string with dot and multiple dots", input: "hello.world.test.test.test", expected: "world.test.test.test"},
	}

	for _, test := range tests {
		s.Run(test.name, func() {
			s.Equal(test.expected, strutil.AfterFirstDot(test.input))
		})
	}
}
