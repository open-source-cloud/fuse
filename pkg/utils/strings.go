package utils

import "strings"

func AfterFirstDot(s string) string {
	if idx := strings.Index(s, "."); idx != -1 && idx+1 < len(s) {
		return s[idx+1:]
	}
	return s
}
