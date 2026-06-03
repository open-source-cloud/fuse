package services

import (
	"context"
	"testing"

	"github.com/open-source-cloud/fuse/internal/repositories"
	"github.com/open-source-cloud/fuse/pkg/secrets"
	"github.com/open-source-cloud/fuse/pkg/workflow"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// readOnlyStore implements secrets.SecretStore but NOT secrets.ManagedSecretStore.
type readOnlyStore struct{}

func (readOnlyStore) Resolve(_ context.Context, _ secrets.Scope, _ string) (secrets.SecretValue, error) {
	return secrets.SecretValue{}, secrets.ErrSecretNotFound
}

func newCredentialService() (CredentialService, secrets.SecretStore) {
	store := secrets.NewMemorySecretStore()
	return NewCredentialService(repositories.NewMemoryCredentialRepository(), store), store
}

func TestCredentialService_SaveWritesValuesToSecretStore(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	svc, store := newCredentialService()

	_, err := svc.Save(workflow.NewCredential("openai-prod", "openai", "Prod", nil),
		map[string]string{"apiKey": "sk-staging"}, "staging")
	require.NoError(t, err)

	// Value is stored at the reserved name in the right environment.
	got, err := store.Resolve(ctx, secrets.Scope{Environment: "staging"}, secrets.CredentialSecretName("openai-prod", "apiKey"))
	require.NoError(t, err)
	assert.Equal(t, "sk-staging", got.Reveal())

	// Per-environment isolation: not visible in another environment.
	_, err = store.Resolve(ctx, secrets.Scope{Environment: "default"}, secrets.CredentialSecretName("openai-prod", "apiKey"))
	assert.ErrorIs(t, err, secrets.ErrSecretNotFound)

	// Metadata records the field name; never the value.
	cred, err := svc.FindByID("openai-prod")
	require.NoError(t, err)
	assert.Equal(t, []string{"apiKey"}, cred.Fields)
}

func TestCredentialService_SaveMergesFields(t *testing.T) {
	t.Parallel()
	svc, _ := newCredentialService()

	_, err := svc.Save(workflow.NewCredential("c1", "custom", "", nil), map[string]string{"apiKey": "a"}, "default")
	require.NoError(t, err)
	_, err = svc.Save(workflow.NewCredential("c1", "custom", "", nil), map[string]string{"baseUrl": "b"}, "default")
	require.NoError(t, err)

	cred, err := svc.FindByID("c1")
	require.NoError(t, err)
	assert.Equal(t, []string{"apiKey", "baseUrl"}, cred.Fields)
}

func TestCredentialService_SaveReadOnlyStoreErrors(t *testing.T) {
	t.Parallel()
	svc := NewCredentialService(repositories.NewMemoryCredentialRepository(), readOnlyStore{})

	_, err := svc.Save(workflow.NewCredential("c1", "custom", "", nil), map[string]string{"k": "v"}, "default")
	assert.ErrorIs(t, err, ErrReadOnlySecretStore)
}

func TestCredentialService_DeleteRemovesValues(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	svc, store := newCredentialService()
	_, err := svc.Save(workflow.NewCredential("c1", "custom", "", nil), map[string]string{"apiKey": "a"}, "staging")
	require.NoError(t, err)

	require.NoError(t, svc.Delete("c1", "staging"))

	_, err = svc.FindByID("c1")
	assert.ErrorIs(t, err, repositories.ErrCredentialNotFound)
	_, err = store.Resolve(ctx, secrets.Scope{Environment: "staging"}, secrets.CredentialSecretName("c1", "apiKey"))
	assert.ErrorIs(t, err, secrets.ErrSecretNotFound)
}

func TestCredentialService_Resolve(t *testing.T) {
	t.Parallel()
	svc, _ := newCredentialService()
	_, err := svc.Save(workflow.NewCredential("c1", "custom", "", nil), map[string]string{"apiKey": "a"}, "default")
	require.NoError(t, err)

	v, err := svc.Resolve(context.Background(), "default", "c1", "apiKey")
	require.NoError(t, err)
	assert.Equal(t, "a", v.Reveal())
}
