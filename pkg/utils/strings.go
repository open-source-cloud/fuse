// Package utils utility functions
package utils

import "strings"

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
