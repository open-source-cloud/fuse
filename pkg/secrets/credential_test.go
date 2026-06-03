package secrets_test

import (
	"errors"
	"testing"

	"github.com/open-source-cloud/fuse/pkg/secrets"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCredentialSecretName(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "cred/openai-prod/apiKey", secrets.CredentialSecretName("openai-prod", "apiKey"))
}

func TestCredentialRefs(t *testing.T) {
	t.Parallel()

	assert.True(t, secrets.HasCredentialRef("Bearer {{credential:openai-prod.apiKey}}"))
	assert.False(t, secrets.HasCredentialRef("no refs"))
	assert.False(t, secrets.HasCredentialRef("{{secret:plain}}"))

	// ID may contain dots; field is the trailing segment (split on the last dot).
	refs := secrets.CredentialRefs("{{credential:my.org.openai.apiKey}} {{credential:my.org.openai.apiKey}}")
	require.Len(t, refs, 1)
	assert.Equal(t, [2]string{"my.org.openai", "apiKey"}, refs[0])
}

func TestReplaceCredentialRefs(t *testing.T) {
	t.Parallel()

	out, err := secrets.ReplaceCredentialRefs("Bearer {{credential:openai-prod.apiKey}}!", func(secretName string) (string, error) {
		assert.Equal(t, "cred/openai-prod/apiKey", secretName)
		return "sk-xyz", nil
	})
	require.NoError(t, err)
	assert.Equal(t, "Bearer sk-xyz!", out)

	_, err = secrets.ReplaceCredentialRefs("{{credential:x.y}}", func(string) (string, error) {
		return "", errors.New("boom")
	})
	require.Error(t, err)
}

// TestCredentialNamespaceDoesNotCollideWithSecretRefs guards the load-bearing invariant: the
// "/" separator in a credential's reserved secret name is outside the {{secret:NAME}} charset, so
// a secret reference can never address a credential value.
func TestCredentialNamespaceDoesNotCollideWithSecretRefs(t *testing.T) {
	t.Parallel()
	name := secrets.CredentialSecretName("openai-prod", "apiKey")
	assert.False(t, secrets.HasSecretRef("{{secret:"+name+"}}"))
}
