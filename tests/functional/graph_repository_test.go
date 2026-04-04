package functional_test

import (
	"testing"

	"github.com/open-source-cloud/fuse/internal/mocks"
	"github.com/open-source-cloud/fuse/internal/repositories"
	internalworkflow "github.com/open-source-cloud/fuse/internal/workflow"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func contractTestGraphRepository(t *testing.T, newRepo func() repositories.GraphRepository) {
	t.Helper()

	t.Run("Save and FindByID returns same graph", func(t *testing.T) {
		repo := newRepo()
		schema := mocks.SmallTestGraphSchema()
		graph, err := internalworkflow.NewGraph(schema)
		require.NoError(t, err)

		err = repo.Save(graph)
		require.NoError(t, err)

		found, err := repo.FindByID(graph.ID())
		require.NoError(t, err)
		assert.Equal(t, graph.ID(), found.ID())
	})

	t.Run("FindByID returns error for nonexistent graph", func(t *testing.T) {
		repo := newRepo()
		g, err := repo.FindByID("nonexistent-graph-id")
		require.Nil(t, g)
		assert.ErrorIs(t, err, repositories.ErrGraphNotFound)
	})

	t.Run("Save overwrites existing graph", func(t *testing.T) {
		repo := newRepo()
		schema := mocks.SmallTestGraphSchema()
		graph, err := internalworkflow.NewGraph(schema)
		require.NoError(t, err)
		require.NoError(t, repo.Save(graph))

		err = repo.Save(graph)
		require.NoError(t, err)

		found, err := repo.FindByID(graph.ID())
		require.NoError(t, err)
		assert.Equal(t, graph.ID(), found.ID())
	})

	t.Run("List returns saved schemas sorted by ID", func(t *testing.T) {
		repo := newRepo()
		s1 := mocks.SmallTestGraphSchema()
		s1.ID = "zebra-schema"
		s1.Name = "Z"
		g1, err := internalworkflow.NewGraph(s1)
		require.NoError(t, err)
		require.NoError(t, repo.Save(g1))

		s2 := mocks.SmallTestGraphSchema()
		s2.ID = "alpha-schema"
		s2.Name = "A"
		g2, err := internalworkflow.NewGraph(s2)
		require.NoError(t, err)
		require.NoError(t, repo.Save(g2))

		list, err := repo.List()
		require.NoError(t, err)
		require.Len(t, list, 2)
		assert.Equal(t, "alpha-schema", list[0].SchemaID)
		assert.Equal(t, "A", list[0].Name)
		assert.Equal(t, "zebra-schema", list[1].SchemaID)
		assert.Equal(t, "Z", list[1].Name)
	})
}

func TestMemoryGraphRepository_Contract(t *testing.T) {
	contractTestGraphRepository(t, repositories.NewMemoryGraphRepository)
}
