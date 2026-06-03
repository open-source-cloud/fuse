package repositories

import (
	"testing"

	"github.com/open-source-cloud/fuse/pkg/workflow"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMemoryEnvironmentRepository(t *testing.T) {
	t.Parallel()

	t.Run("seeds the default environment", func(t *testing.T) {
		t.Parallel()
		repo := NewMemoryEnvironmentRepository()

		env, err := repo.FindByID(workflow.DefaultEnvironmentName)

		require.NoError(t, err)
		assert.Equal(t, workflow.DefaultEnvironmentName, env.Name)
	})

	t.Run("Save and FindByID round-trip", func(t *testing.T) {
		t.Parallel()
		repo := NewMemoryEnvironmentRepository()

		require.NoError(t, repo.Save(workflow.NewEnvironment("staging", "Staging env")))
		env, err := repo.FindByID("staging")

		require.NoError(t, err)
		assert.Equal(t, "staging", env.Name)
		assert.Equal(t, "Staging env", env.Description)
	})

	t.Run("FindByID returns ErrEnvironmentNotFound for unknown", func(t *testing.T) {
		t.Parallel()
		repo := NewMemoryEnvironmentRepository()

		_, err := repo.FindByID("nope")

		assert.ErrorIs(t, err, ErrEnvironmentNotFound)
	})

	t.Run("FindAll returns sorted environments including default", func(t *testing.T) {
		t.Parallel()
		repo := NewMemoryEnvironmentRepository()
		require.NoError(t, repo.Save(workflow.NewEnvironment("staging", "")))
		require.NoError(t, repo.Save(workflow.NewEnvironment("prod", "")))

		envs, err := repo.FindAll()

		require.NoError(t, err)
		names := make([]string, len(envs))
		for i, e := range envs {
			names[i] = e.Name
		}
		assert.Equal(t, []string{"default", "prod", "staging"}, names)
	})

	t.Run("Delete removes an environment", func(t *testing.T) {
		t.Parallel()
		repo := NewMemoryEnvironmentRepository()
		require.NoError(t, repo.Save(workflow.NewEnvironment("staging", "")))

		require.NoError(t, repo.Delete("staging"))
		_, err := repo.FindByID("staging")

		assert.ErrorIs(t, err, ErrEnvironmentNotFound)
	})
}
