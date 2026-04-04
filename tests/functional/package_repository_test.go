package functional_test

import (
	"testing"

	"github.com/open-source-cloud/fuse/internal/repositories"
	"github.com/open-source-cloud/fuse/pkg/transport"
	"github.com/open-source-cloud/fuse/pkg/workflow"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func contractTestPackageRepository(t *testing.T, newRepo func() repositories.PackageRepository) {
	t.Helper()

	t.Run("Save and FindByID returns same package", func(t *testing.T) {
		repo := newRepo()
		pkg := workflow.NewPackage("test-pkg",
			workflow.NewFunction("fn-1", workflow.FunctionMetadata{
				Transport: transport.HTTP,
			}, nil),
		)

		require.NoError(t, repo.Save(pkg))
		found, err := repo.FindByID("test-pkg")

		require.NoError(t, err)
		assert.Equal(t, "test-pkg", found.ID)
		require.Len(t, found.Functions, 1)
		assert.Equal(t, "fn-1", found.Functions[0].ID)
	})

	t.Run("FindByID returns error for nonexistent package", func(t *testing.T) {
		repo := newRepo()
		p, err := repo.FindByID("nonexistent-pkg")
		require.Nil(t, p)
		assert.ErrorIs(t, err, repositories.ErrPackageNotFound)
	})

	t.Run("FindAll returns all saved packages", func(t *testing.T) {
		repo := newRepo()
		pkg1 := workflow.NewPackage("pkg-findall-1",
			workflow.NewFunction("fn-a", workflow.FunctionMetadata{Transport: transport.HTTP}, nil),
		)
		pkg2 := workflow.NewPackage("pkg-findall-2",
			workflow.NewFunction("fn-b", workflow.FunctionMetadata{Transport: transport.HTTP}, nil),
		)
		require.NoError(t, repo.Save(pkg1))
		require.NoError(t, repo.Save(pkg2))

		all, err := repo.FindAll()
		require.NoError(t, err)
		// Use >= to be resilient to pre-existing data from other subtests (shared DB)
		assert.GreaterOrEqual(t, len(all), 2)
		ids := make([]string, len(all))
		for i, p := range all {
			ids[i] = p.ID
		}
		assert.Contains(t, ids, "pkg-findall-1")
		assert.Contains(t, ids, "pkg-findall-2")
	})

	t.Run("Delete removes package", func(t *testing.T) {
		repo := newRepo()
		pkg := workflow.NewPackage("pkg-to-delete",
			workflow.NewFunction("fn-x", workflow.FunctionMetadata{Transport: transport.HTTP}, nil),
		)
		require.NoError(t, repo.Save(pkg))

		require.NoError(t, repo.Delete("pkg-to-delete"))

		p, err := repo.FindByID("pkg-to-delete")
		assert.Nil(t, p)
		assert.ErrorIs(t, err, repositories.ErrPackageNotFound)
	})

	t.Run("Save overwrites existing package", func(t *testing.T) {
		repo := newRepo()
		pkg := workflow.NewPackage("pkg-overwrite",
			workflow.NewFunction("fn-old", workflow.FunctionMetadata{Transport: transport.HTTP}, nil),
		)
		require.NoError(t, repo.Save(pkg))

		updated := workflow.NewPackage("pkg-overwrite",
			workflow.NewFunction("fn-new-1", workflow.FunctionMetadata{Transport: transport.HTTP}, nil),
			workflow.NewFunction("fn-new-2", workflow.FunctionMetadata{Transport: transport.HTTP}, nil),
		)
		require.NoError(t, repo.Save(updated))

		found, err := repo.FindByID("pkg-overwrite")
		require.NoError(t, err)
		assert.Len(t, found.Functions, 2)
	})

	t.Run("Save preserves tags", func(t *testing.T) {
		repo := newRepo()
		pkg := workflow.NewPackage("pkg-with-tags",
			workflow.NewFunction("fn-1", workflow.FunctionMetadata{Transport: transport.HTTP}, nil),
		)
		pkg.Tags = map[string]string{"env": "prod", "team": "platform"}

		require.NoError(t, repo.Save(pkg))
		found, err := repo.FindByID("pkg-with-tags")

		require.NoError(t, err)
		assert.Equal(t, "prod", found.Tags["env"])
		assert.Equal(t, "platform", found.Tags["team"])
	})
}

func TestMemoryPackageRepository_Contract(t *testing.T) {
	contractTestPackageRepository(t, func() repositories.PackageRepository {
		return repositories.NewMemoryPackageRepository()
	})
}
