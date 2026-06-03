package functional_test

import (
	"testing"

	"github.com/open-source-cloud/fuse/internal/repositories"
	"github.com/open-source-cloud/fuse/pkg/workflow"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func contractTestCredentialRepository(t *testing.T, newRepo func() repositories.CredentialRepository, reset func()) {
	t.Helper()

	t.Run("Save and FindByID returns same credential incl. fields array", func(t *testing.T) {
		reset()
		repo := newRepo()

		require.NoError(t, repo.Save(workflow.NewCredential("openai-prod", "openai", "Prod creds", []string{"apiKey", "baseUrl"})))
		found, err := repo.FindByID("openai-prod")

		require.NoError(t, err)
		assert.Equal(t, "openai", found.Type)
		assert.Equal(t, "Prod creds", found.Description)
		assert.Equal(t, []string{"apiKey", "baseUrl"}, found.Fields)
	})

	t.Run("FindByID returns ErrCredentialNotFound for unknown", func(t *testing.T) {
		reset()
		repo := newRepo()
		_, err := repo.FindByID("nonexistent")
		assert.ErrorIs(t, err, repositories.ErrCredentialNotFound)
	})

	t.Run("FindAll returns saved credentials", func(t *testing.T) {
		reset()
		repo := newRepo()
		require.NoError(t, repo.Save(workflow.NewCredential("a-cred", "custom", "", []string{"token"})))
		require.NoError(t, repo.Save(workflow.NewCredential("b-cred", "custom", "", nil)))

		all, err := repo.FindAll()
		require.NoError(t, err)
		ids := make([]string, len(all))
		for i, c := range all {
			ids[i] = c.ID
		}
		assert.Contains(t, ids, "a-cred")
		assert.Contains(t, ids, "b-cred")
	})

	t.Run("Save overwrites metadata", func(t *testing.T) {
		reset()
		repo := newRepo()
		require.NoError(t, repo.Save(workflow.NewCredential("c1", "openai", "old", []string{"apiKey"})))
		require.NoError(t, repo.Save(workflow.NewCredential("c1", "openai", "new", []string{"apiKey", "baseUrl"})))

		found, err := repo.FindByID("c1")
		require.NoError(t, err)
		assert.Equal(t, "new", found.Description)
		assert.Equal(t, []string{"apiKey", "baseUrl"}, found.Fields)
	})

	t.Run("Delete removes credential", func(t *testing.T) {
		reset()
		repo := newRepo()
		require.NoError(t, repo.Save(workflow.NewCredential("c1", "custom", "", nil)))

		require.NoError(t, repo.Delete("c1"))
		_, err := repo.FindByID("c1")
		assert.ErrorIs(t, err, repositories.ErrCredentialNotFound)
	})
}

func TestMemoryCredentialRepository_Contract(t *testing.T) {
	contractTestCredentialRepository(t, func() repositories.CredentialRepository {
		return repositories.NewMemoryCredentialRepository()
	}, func() {})
}
