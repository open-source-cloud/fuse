package repositories_test

import (
	"testing"
	"time"

	"github.com/open-source-cloud/fuse/internal/mocks"
	"github.com/open-source-cloud/fuse/internal/repositories"
	"github.com/open-source-cloud/fuse/internal/workflow"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newPopulatedMemoryRepo(t *testing.T) (repositories.GraphRepository, *workflow.Graph) {
	t.Helper()
	repo := repositories.NewMemoryGraphRepository()
	schema := mocks.SmallTestGraphSchema()
	graph, err := workflow.NewGraph(schema)
	require.NoError(t, err)
	require.NoError(t, repo.Save(graph))
	return repo, graph
}

// TestMemoryGraphRepository_SaveAndGetVersion tests basic version save and retrieval.
func TestMemoryGraphRepository_SaveAndGetVersion(t *testing.T) {
	repo, graph := newPopulatedMemoryRepo(t)

	schema := graph.Schema()
	sv := &workflow.SchemaVersion{
		SchemaID:  schema.ID,
		Version:   1,
		Schema:    schema,
		CreatedAt: time.Now().UTC(),
		IsActive:  true,
	}
	require.NoError(t, repo.SaveVersion(sv))

	got, err := repo.FindByIDAndVersion(schema.ID, 1)
	require.NoError(t, err)
	assert.Equal(t, schema.ID, got.ID())
}

// TestMemoryGraphRepository_FindByIDAndVersion_NotFound returns an error for missing versions.
func TestMemoryGraphRepository_FindByIDAndVersion_NotFound(t *testing.T) {
	repo := repositories.NewMemoryGraphRepository()
	_, err := repo.FindByIDAndVersion("ghost", 1)
	assert.ErrorIs(t, err, repositories.ErrSchemaVersionNotFound)
}

// TestMemoryGraphRepository_ListVersions returns versions sorted ascending.
func TestMemoryGraphRepository_ListVersions(t *testing.T) {
	repo, graph := newPopulatedMemoryRepo(t)
	schema := graph.Schema()

	for _, v := range []int{2, 1, 3} {
		sv := &workflow.SchemaVersion{
			SchemaID:  schema.ID,
			Version:   v,
			Schema:    schema,
			CreatedAt: time.Now().UTC(),
		}
		require.NoError(t, repo.SaveVersion(sv))
	}

	versions, err := repo.ListVersions(schema.ID)
	require.NoError(t, err)
	require.Len(t, versions, 3)
	assert.Equal(t, 1, versions[0].Version)
	assert.Equal(t, 2, versions[1].Version)
	assert.Equal(t, 3, versions[2].Version)
}

// TestMemoryGraphRepository_ListVersions_SchemaNotFound returns ErrGraphNotFound for missing schemas.
func TestMemoryGraphRepository_ListVersions_SchemaNotFound(t *testing.T) {
	repo := repositories.NewMemoryGraphRepository()
	_, err := repo.ListVersions("ghost")
	assert.ErrorIs(t, err, repositories.ErrGraphNotFound)
}

// TestMemoryGraphRepository_SetActiveVersion activates a version and updates the cached graph.
func TestMemoryGraphRepository_SetActiveVersion(t *testing.T) {
	repo, graph := newPopulatedMemoryRepo(t)
	schema := graph.Schema()

	// Save two versions
	for _, v := range []int{1, 2} {
		sv := &workflow.SchemaVersion{
			SchemaID:  schema.ID,
			Version:   v,
			Schema:    schema,
			CreatedAt: time.Now().UTC(),
			IsActive:  v == 2,
		}
		require.NoError(t, repo.SaveVersion(sv))
	}

	// Activate v1
	require.NoError(t, repo.SetActiveVersion(schema.ID, 1))

	history, err := repo.GetVersionHistory(schema.ID)
	require.NoError(t, err)
	assert.Equal(t, 1, history.ActiveVersion)

	versions, err := repo.ListVersions(schema.ID)
	require.NoError(t, err)
	activeCount := 0
	for _, sv := range versions {
		if sv.IsActive {
			activeCount++
			assert.Equal(t, 1, sv.Version)
		}
	}
	assert.Equal(t, 1, activeCount, "exactly one version should be active")
}

// TestMemoryGraphRepository_SetActiveVersion_NotFound errors for unknown version.
func TestMemoryGraphRepository_SetActiveVersion_NotFound(t *testing.T) {
	repo, graph := newPopulatedMemoryRepo(t)
	schema := graph.Schema()

	sv := &workflow.SchemaVersion{
		SchemaID: schema.ID, Version: 1, Schema: schema, CreatedAt: time.Now().UTC(),
	}
	require.NoError(t, repo.SaveVersion(sv))

	err := repo.SetActiveVersion(schema.ID, 99)
	assert.ErrorIs(t, err, repositories.ErrSchemaVersionNotFound)
}

// TestMemoryGraphRepository_GetVersionHistory returns correct aggregate metadata.
func TestMemoryGraphRepository_GetVersionHistory(t *testing.T) {
	repo, graph := newPopulatedMemoryRepo(t)
	schema := graph.Schema()

	for _, v := range []int{1, 2, 3} {
		sv := &workflow.SchemaVersion{
			SchemaID:  schema.ID,
			Version:   v,
			Schema:    schema,
			CreatedAt: time.Now().UTC(),
			IsActive:  v == 3,
		}
		require.NoError(t, repo.SaveVersion(sv))
	}

	history, err := repo.GetVersionHistory(schema.ID)
	require.NoError(t, err)
	assert.Equal(t, schema.ID, history.SchemaID)
	assert.Equal(t, 3, history.ActiveVersion)
	assert.Equal(t, 3, history.LatestVersion)
	assert.Equal(t, 3, history.TotalVersions)
}

// TestMemoryGraphRepository_GetVersionHistory_NoVersions returns zeroed history when no versions are tracked.
func TestMemoryGraphRepository_GetVersionHistory_NoVersions(t *testing.T) {
	repo, graph := newPopulatedMemoryRepo(t)
	schema := graph.Schema()

	history, err := repo.GetVersionHistory(schema.ID)
	require.NoError(t, err)
	assert.Equal(t, 0, history.TotalVersions)
	assert.Equal(t, 0, history.LatestVersion)
}

// TestMemoryGraphRepository_GetVersionHistory_SchemaNotFound errors for missing schema.
func TestMemoryGraphRepository_GetVersionHistory_SchemaNotFound(t *testing.T) {
	repo := repositories.NewMemoryGraphRepository()
	_, err := repo.GetVersionHistory("ghost")
	assert.ErrorIs(t, err, repositories.ErrGraphNotFound)
}

// TestMemoryGraphRepository_SaveVersion_IsActive_deactivates_others ensures only one version is active.
func TestMemoryGraphRepository_SaveVersion_IsActive_deactivates_others(t *testing.T) {
	repo, graph := newPopulatedMemoryRepo(t)
	schema := graph.Schema()

	sv1 := &workflow.SchemaVersion{
		SchemaID: schema.ID, Version: 1, Schema: schema,
		CreatedAt: time.Now().UTC(), IsActive: true,
	}
	require.NoError(t, repo.SaveVersion(sv1))

	sv2 := &workflow.SchemaVersion{
		SchemaID: schema.ID, Version: 2, Schema: schema,
		CreatedAt: time.Now().UTC(), IsActive: true,
	}
	require.NoError(t, repo.SaveVersion(sv2))

	versions, err := repo.ListVersions(schema.ID)
	require.NoError(t, err)

	activeCount := 0
	for _, sv := range versions {
		if sv.IsActive {
			activeCount++
		}
	}
	assert.Equal(t, 1, activeCount, "saving v2 as active should deactivate v1")

	history, err := repo.GetVersionHistory(schema.ID)
	require.NoError(t, err)
	assert.Equal(t, 2, history.ActiveVersion)
}
