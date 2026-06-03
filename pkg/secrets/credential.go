package secrets

import (
	"fmt"
	"regexp"
)

// credentialSecretPrefix namespaces credential field values within the SecretStore. A credential
// field is stored as the secret named "cred/<id>/<field>" (ADR-0031 Option B). The "/" separator
// is deliberately outside the {{secret:NAME}} charset so a secret reference can never address a
// credential value and the two namespaces cannot collide.
const credentialSecretPrefix = "cred"

// CredentialSecretName maps a credential field to its reserved SecretStore name.
func CredentialSecretName(id, field string) string {
	return fmt.Sprintf("%s/%s/%s", credentialSecretPrefix, id, field)
}

// credentialRefPattern matches {{credential:ID.FIELD}} tokens. ID may contain dots; FIELD may not,
// so the value is split on the LAST dot (ID captured greedily, FIELD as the trailing segment).
var credentialRefPattern = regexp.MustCompile(`\{\{credential:([A-Za-z0-9_.\-]+)\.([A-Za-z0-9_\-]+)\}\}`)

// CredentialRefToken renders the {{credential:ID.FIELD}} token form.
func CredentialRefToken(id, field string) string {
	return fmt.Sprintf("{{credential:%s.%s}}", id, field)
}

// HasCredentialRef reports whether s contains any {{credential:ID.FIELD}} token.
func HasCredentialRef(s string) bool {
	return credentialRefPattern.MatchString(s)
}

// CredentialRefs returns the distinct (id, field) pairs referenced in s.
func CredentialRefs(s string) [][2]string {
	matches := credentialRefPattern.FindAllStringSubmatch(s, -1)
	refs := make([][2]string, 0, len(matches))
	seen := make(map[string]struct{}, len(matches))
	for _, m := range matches {
		key := m[1] + "/" + m[2]
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		refs = append(refs, [2]string{m[1], m[2]})
	}
	return refs
}

// ReplaceCredentialRefs replaces each {{credential:ID.FIELD}} in s with resolve(secretName), where
// secretName is the reserved CredentialSecretName(ID, FIELD); it returns the first resolution
// error if any. The resolved plaintext must be wrapped in a SecretValue by the caller.
func ReplaceCredentialRefs(s string, resolve func(secretName string) (string, error)) (string, error) {
	var firstErr error
	out := credentialRefPattern.ReplaceAllStringFunc(s, func(token string) string {
		if firstErr != nil {
			return token
		}
		m := credentialRefPattern.FindStringSubmatch(token)
		val, err := resolve(CredentialSecretName(m[1], m[2]))
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
