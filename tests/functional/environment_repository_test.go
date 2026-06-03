package functional_test

import (
	"testing"

	"github.com/open-source-cloud/fuse/internal/repositories"
	"github.com/open-source-cloud/fuse/pkg/workflow"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func contractTestEnvironmentRepository(t *testing.T, newRepo func() repositories.EnvironmentRepository, reset func()) {
	t.Helper()

	t.Run("default environment is always present", func(t *testing.T) {
		reset()
		repo := newRepo()

		env, err := repo.FindByID(workflow.DefaultEnvironmentName)

		require.NoError(t, err)
		assert.Equal(t, workflow.DefaultEnvironmentName, env.Name)
	})

	t.Run("Save and FindByID returns same environment", func(t *testing.T) {
		reset()
		repo := newRepo()

		require.NoError(t, repo.Save(workflow.NewEnvironment("staging", "Staging env")))
		found, err := repo.FindByID("staging")

		require.NoError(t, err)
		assert.Equal(t, "staging", found.Name)
		assert.Equal(t, "Staging env", found.Description)
	})

	t.Run("FindByID returns error for nonexistent environment", func(t *testing.T) {
		reset()
		repo := newRepo()

		_, err := repo.FindByID("nonexistent-env")

		assert.ErrorIs(t, err, repositories.ErrEnvironmentNotFound)
	})

	t.Run("FindAll returns default plus saved environments", func(t *testing.T) {
		reset()
		repo := newRepo()
		require.NoError(t, repo.Save(workflow.NewEnvironment("staging", "")))
		require.NoError(t, repo.Save(workflow.NewEnvironment("prod", "")))

		all, err := repo.FindAll()

		require.NoError(t, err)
		names := make([]string, len(all))
		for i, e := range all {
			names[i] = e.Name
		}
		assert.Contains(t, names, workflow.DefaultEnvironmentName)
		assert.Contains(t, names, "staging")
		assert.Contains(t, names, "prod")
	})

	t.Run("Save overwrites description", func(t *testing.T) {
		reset()
		repo := newRepo()
		require.NoError(t, repo.Save(workflow.NewEnvironment("staging", "old")))
		require.NoError(t, repo.Save(workflow.NewEnvironment("staging", "new")))

		found, err := repo.FindByID("staging")

		require.NoError(t, err)
		assert.Equal(t, "new", found.Description)
	})

	t.Run("Delete removes environment", func(t *testing.T) {
		reset()
		repo := newRepo()
		require.NoError(t, repo.Save(workflow.NewEnvironment("staging", "")))

		require.NoError(t, repo.Delete("staging"))

		_, err := repo.FindByID("staging")
		assert.ErrorIs(t, err, repositories.ErrEnvironmentNotFound)
	})
}

func TestMemoryEnvironmentRepository_Contract(t *testing.T) {
	contractTestEnvironmentRepository(t, func() repositories.EnvironmentRepository {
		return repositories.NewMemoryEnvironmentRepository()
	}, func() {})
}
