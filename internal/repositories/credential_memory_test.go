package repositories

import (
	"testing"

	"github.com/open-source-cloud/fuse/pkg/workflow"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMemoryCredentialRepository(t *testing.T) {
	t.Parallel()

	t.Run("Save and FindByID round-trip", func(t *testing.T) {
		t.Parallel()
		repo := NewMemoryCredentialRepository()

		require.NoError(t, repo.Save(workflow.NewCredential("openai-prod", "openai", "Prod", []string{"apiKey"})))
		cred, err := repo.FindByID("openai-prod")

		require.NoError(t, err)
		assert.Equal(t, "openai", cred.Type)
		assert.Equal(t, []string{"apiKey"}, cred.Fields)
	})

	t.Run("FindByID returns ErrCredentialNotFound for unknown", func(t *testing.T) {
		t.Parallel()
		repo := NewMemoryCredentialRepository()
		_, err := repo.FindByID("nope")
		assert.ErrorIs(t, err, ErrCredentialNotFound)
	})

	t.Run("FindAll returns sorted credentials", func(t *testing.T) {
		t.Parallel()
		repo := NewMemoryCredentialRepository()
		require.NoError(t, repo.Save(workflow.NewCredential("b-cred", "custom", "", nil)))
		require.NoError(t, repo.Save(workflow.NewCredential("a-cred", "custom", "", nil)))

		creds, err := repo.FindAll()
		require.NoError(t, err)
		require.Len(t, creds, 2)
		assert.Equal(t, "a-cred", creds[0].ID)
		assert.Equal(t, "b-cred", creds[1].ID)
	})

	t.Run("Delete removes a credential", func(t *testing.T) {
		t.Parallel()
		repo := NewMemoryCredentialRepository()
		require.NoError(t, repo.Save(workflow.NewCredential("c1", "custom", "", nil)))

		require.NoError(t, repo.Delete("c1"))
		_, err := repo.FindByID("c1")
		assert.ErrorIs(t, err, ErrCredentialNotFound)
	})
}
