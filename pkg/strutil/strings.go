// Package strutil provides string utility functions for common string operations.
// It includes functions for string manipulation, token replacement, and serialization.
package strutil

import (
	"fmt"
	"sort"
	"strings"
)

// AfterFirstDot returns the string after first instance of dot ('.')
func AfterFirstDot(s string) string {
	if idx := strings.Index(s, "."); idx != -1 && idx+1 < len(s) {
		return s[idx+1:]
	}
	return s
}

// SerializeString serializes a string by converting it to lowercase and trimming whitespace
func SerializeString(str string) string {
	if str == "" {
		return ""
	}
	str = strings.ToLower(str)
	str = strings.TrimSpace(str)
	return str
}

// ReplaceTokens replaces tokens in a string with their values from a map
// The keys are sorted by length in descending order to avoid replacing substrings
// Always returns a new string and never modifies the input string
func ReplaceTokens(str string, input map[string]any) string {
	if input == nil {
		return str
	}
	if len(input) == 0 {
		return str
	}

	// Create a new string to avoid modifying the input string
	newStr := str

	// Copy the keys to a new slice and sort them by length in descending order
	// This is to avoid replacing substrings
	keys := make([]string, 0, len(input))
	for k := range input {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		return len(keys[i]) > len(keys[j])
	})

	// Replace the tokens in the new string
	for _, key := range keys {
		val := fmt.Sprintf("%v", input[key])
		newStr = strings.ReplaceAll(newStr, fmt.Sprintf("{{%s}}", key), val)
	}

	return newStr
}
