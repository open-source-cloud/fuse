package secrets_test

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"testing"

	"github.com/open-source-cloud/fuse/pkg/secrets"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSecretValue_Redaction(t *testing.T) {
	t.Parallel()
	sv := secrets.NewSecretValue("s3cr3t")

	assert.Equal(t, "s3cr3t", sv.Reveal())
	assert.Equal(t, secrets.RedactedMarker, sv.String())

	b, err := json.Marshal(sv)
	require.NoError(t, err)
	assert.JSONEq(t, `"***"`, string(b))

	// Inside a map[string]any (how node I/O flows to journal/snapshot/trace).
	mb, err := json.Marshal(map[string]any{"apiKey": sv, "plain": "x"})
	require.NoError(t, err)
	assert.JSONEq(t, `{"apiKey":"***","plain":"x"}`, string(mb))
}

func TestSecretRefs(t *testing.T) {
	t.Parallel()
	assert.True(t, secrets.HasSecretRef("Bearer {{secret:api-token}}"))
	assert.False(t, secrets.HasSecretRef("no refs here"))
	assert.Equal(t, []string{"a", "b"}, secrets.SecretRefNames("{{secret:a}} and {{secret:b}} and {{secret:a}}"))

	out, err := secrets.ReplaceSecretRefs("Bearer {{secret:tok}}!", func(name string) (string, error) {
		assert.Equal(t, "tok", name)
		return "XYZ", nil
	})
	require.NoError(t, err)
	assert.Equal(t, "Bearer XYZ!", out)
}

func TestCipher_RoundTrip(t *testing.T) {
	t.Parallel()
	key := base64.StdEncoding.EncodeToString(make([]byte, 32))
	c, err := secrets.NewCipherFromBase64Key(key)
	require.NoError(t, err)

	ct, err := c.Encrypt([]byte("hunter2"))
	require.NoError(t, err)
	assert.NotEqual(t, "hunter2", string(ct), "ciphertext must not equal plaintext")

	pt, err := c.Decrypt(ct)
	require.NoError(t, err)
	assert.Equal(t, "hunter2", string(pt))

	// Tampering is detected (GCM auth).
	ct[len(ct)-1] ^= 0xFF
	_, err = c.Decrypt(ct)
	assert.Error(t, err)
}

func TestCipher_BadKey(t *testing.T) {
	t.Parallel()
	_, err := secrets.NewCipherFromBase64Key("not-base64!!")
	assert.Error(t, err)
	_, err = secrets.NewCipherFromBase64Key(base64.StdEncoding.EncodeToString(make([]byte, 16)))
	assert.Error(t, err, "key must be 32 bytes")
}

func TestMemorySecretStore(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	s := secrets.NewMemorySecretStore()
	scope := secrets.Scope{Environment: "prod"}

	_, err := s.Resolve(ctx, scope, "missing")
	require.ErrorIs(t, err, secrets.ErrSecretNotFound)

	require.NoError(t, s.Set(ctx, scope, "api-key", "abc"))
	v, err := s.Resolve(ctx, scope, "api-key")
	require.NoError(t, err)
	assert.Equal(t, "abc", v.Reveal())

	// Scoped by environment.
	_, err = s.Resolve(ctx, secrets.Scope{Environment: "dev"}, "api-key")
	require.ErrorIs(t, err, secrets.ErrSecretNotFound)

	names, err := s.List(ctx, "prod")
	require.NoError(t, err)
	assert.Equal(t, []string{"api-key"}, names)

	require.NoError(t, s.Delete(ctx, scope, "api-key"))
	_, err = s.Resolve(ctx, scope, "api-key")
	require.ErrorIs(t, err, secrets.ErrSecretNotFound)
}

func TestMemorySecretStoreFromEnv(t *testing.T) {
	t.Setenv("FUSE_SECRET_GREETING", "hello")
	s := secrets.NewMemorySecretStoreFromEnv("default")
	v, err := s.Resolve(context.Background(), secrets.Scope{Environment: "default"}, "GREETING")
	require.NoError(t, err)
	assert.Equal(t, "hello", v.Reveal())
}

func TestResolver_BindsEnvironment(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	store := secrets.NewMemorySecretStore()
	require.NoError(t, store.Set(ctx, secrets.Scope{Environment: "staging"}, "tok", "T"))

	r := secrets.NewResolver(store, "staging")
	v, err := r.Resolve(ctx, "wf-1", "tok")
	require.NoError(t, err)
	assert.Equal(t, "T", v.Reveal())
}
