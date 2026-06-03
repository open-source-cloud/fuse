// Package secrets provides the secret-value wrapper, the SecretStore seam, the
// {{secret:NAME}} reference syntax, and the in-memory + AES-256-GCM building
// blocks shared by the workflow engine and the secret backends.
//
// A resolved secret is carried as a SecretValue so it renders as a redaction
// marker everywhere the engine serializes or logs node input/output, while the
// consuming function can Reveal() the plaintext where it is actually used.
package secrets

import (
	"encoding/json"
	"regexp"
)

// RedactedMarker is what a SecretValue renders as in any non-Reveal context.
const RedactedMarker = "***"

// SecretValue wraps a resolved secret's plaintext. It marshals and stringifies
// to RedactedMarker so it never leaks into journals, snapshots, traces, logs, or
// aggregated output; the consuming function calls Reveal() to obtain the
// plaintext only where the value is actually used.
type SecretValue struct {
	plaintext string
}

// NewSecretValue wraps a plaintext secret.
func NewSecretValue(plaintext string) SecretValue {
	return SecretValue{plaintext: plaintext}
}

// Reveal returns the plaintext. Call this ONLY where the value is consumed (e.g.
// an HTTP Authorization header), never where it would be logged or persisted.
func (s SecretValue) Reveal() string { return s.plaintext }

// String renders the redaction marker so fmt/%v never prints the plaintext.
func (s SecretValue) String() string { return RedactedMarker }

// MarshalJSON renders the redaction marker so every JSON sink redacts the value.
func (s SecretValue) MarshalJSON() ([]byte, error) {
	return json.Marshal(RedactedMarker)
}

// secretRefPattern matches {{secret:NAME}} tokens; NAME is [A-Za-z0-9_.-]+.
var secretRefPattern = regexp.MustCompile(`\{\{secret:([A-Za-z0-9_.\-]+)\}\}`)

// HasSecretRef reports whether s contains any {{secret:NAME}} token.
func HasSecretRef(s string) bool {
	return secretRefPattern.MatchString(s)
}

// SecretRefNames returns the distinct secret names referenced in s.
func SecretRefNames(s string) []string {
	matches := secretRefPattern.FindAllStringSubmatch(s, -1)
	names := make([]string, 0, len(matches))
	seen := make(map[string]struct{}, len(matches))
	for _, m := range matches {
		if _, ok := seen[m[1]]; ok {
			continue
		}
		seen[m[1]] = struct{}{}
		names = append(names, m[1])
	}
	return names
}

// ReplaceSecretRefs replaces each {{secret:NAME}} in s with resolve(NAME),
// returning the first resolution error if any.
func ReplaceSecretRefs(s string, resolve func(name string) (string, error)) (string, error) {
	var firstErr error
	out := secretRefPattern.ReplaceAllStringFunc(s, func(token string) string {
		if firstErr != nil {
			return token
		}
		m := secretRefPattern.FindStringSubmatch(token)
		val, err := resolve(m[1])
		if err != nil {
			firstErr = err
			return token
		}
		return val
	})
	if firstErr != nil {
		return "", firstErr
	}
	return out, nil
}
