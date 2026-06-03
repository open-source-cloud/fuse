//go:build functional

package functional_test

import (
	"context"
	"encoding/base64"
	"testing"

	"github.com/open-source-cloud/fuse/internal/repositories/postgres"
	"github.com/open-source-cloud/fuse/pkg/secrets"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPostgresSecretStore_EncryptedRoundTrip(t *testing.T) {
	pool := setupTestPool(t)
	ctx := context.Background()
	_, _ = pool.Exec(ctx, "TRUNCATE TABLE secrets")

	key := base64.StdEncoding.EncodeToString(make([]byte, 32))
	cipher, err := secrets.NewCipherFromBase64Key(key)
	require.NoError(t, err)

	store := postgres.NewSecretStore(pool, cipher)
	scope := secrets.Scope{Environment: "prod"}

	// Not found initially.
	_, err = store.Resolve(ctx, scope, "api-key")
	require.ErrorIs(t, err, secrets.ErrSecretNotFound)

	// Set + resolve (decrypts back to plaintext).
	require.NoError(t, store.Set(ctx, scope, "api-key", "s3cr3t"))
	v, err := store.Resolve(ctx, scope, "api-key")
	require.NoError(t, err)
	assert.Equal(t, "s3cr3t", v.Reveal())

	// The stored column is ciphertext, never plaintext.
	var enc []byte
	require.NoError(t, pool.QueryRow(ctx,
		"SELECT encrypted_value FROM secrets WHERE environment=$1 AND name=$2", "prod", "api-key").Scan(&enc))
	assert.NotContains(t, string(enc), "s3cr3t", "secret must be encrypted at rest")

	// Upsert replaces (rotation).
	require.NoError(t, store.Set(ctx, scope, "api-key", "rotated"))
	v, err = store.Resolve(ctx, scope, "api-key")
	require.NoError(t, err)
	assert.Equal(t, "rotated", v.Reveal())

	// List is scoped to the environment.
	require.NoError(t, store.Set(ctx, secrets.Scope{Environment: "dev"}, "other", "x"))
	names, err := store.List(ctx, "prod")
	require.NoError(t, err)
	assert.Equal(t, []string{"api-key"}, names)

	// Delete.
	require.NoError(t, store.Delete(ctx, scope, "api-key"))
	_, err = store.Resolve(ctx, scope, "api-key")
	require.ErrorIs(t, err, secrets.ErrSecretNotFound)
}
