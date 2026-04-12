package services_test

import (
	"testing"

	"github.com/open-source-cloud/fuse/internal/mocks"
	"github.com/open-source-cloud/fuse/internal/packages"
	"github.com/open-source-cloud/fuse/internal/repositories"
	"github.com/open-source-cloud/fuse/internal/services"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newVersioningGraphService(t *testing.T) services.GraphService {
	t.Helper()
	memGraphRepo := repositories.NewMemoryGraphRepository()
	pkgRegistry := packages.NewPackageRegistry()
	internalPackages := packages.NewInternal()
	pkgSvc := services.NewPackageService(repositories.NewMemoryPackageRepository(), pkgRegistry, internalPackages)
	require.NoError(t, pkgSvc.RegisterInternalPackages())
	return services.NewGraphService(memGraphRepo, pkgRegistry, nil)
}

// TestGraphService_Upsert_CreatesVersionOne verifies the first Upsert creates version 1.
func TestGraphService_Upsert_CreatesVersionOne(t *testing.T) {
	svc := newVersioningGraphService(t)
	schema := mocks.SmallTestGraphSchema()

	_, err := svc.Upsert(schema.ID, schema)
	require.NoError(t, err)

	versions, err := svc.ListVersions(schema.ID)
	require.NoError(t, err)
	require.Len(t, versions, 1)
	assert.Equal(t, 1, versions[0].Version)
	assert.True(t, versions[0].IsActive)
}

// TestGraphService_Upsert_SecondCallCreatesVersionTwo verifies subsequent Upserts increment version.
func TestGraphService_Upsert_SecondCallCreatesVersionTwo(t *testing.T) {
	svc := newVersioningGraphService(t)
	schema := mocks.SmallTestGraphSchema()

	_, err := svc.Upsert(schema.ID, schema)
	require.NoError(t, err)

	schema.Name = "Updated Name"
	_, err = svc.Upsert(schema.ID, schema)
	require.NoError(t, err)

	versions, err := svc.ListVersions(schema.ID)
	require.NoError(t, err)
	require.Len(t, versions, 2)
	assert.Equal(t, 1, versions[0].Version)
	assert.False(t, versions[0].IsActive)
	assert.Equal(t, 2, versions[1].Version)
	assert.True(t, versions[1].IsActive)
}

// TestGraphService_FindByIDAndVersion retrieves any version's graph.
func TestGraphService_FindByIDAndVersion(t *testing.T) {
	svc := newVersioningGraphService(t)
	schema := mocks.SmallTestGraphSchema()

	_, err := svc.Upsert(schema.ID, schema)
	require.NoError(t, err)

	schema.Name = "V2 Name"
	_, err = svc.Upsert(schema.ID, schema)
	require.NoError(t, err)

	// v1 should still have the original name
	graphV1, err := svc.FindByIDAndVersion(schema.ID, 1)
	require.NoError(t, err)
	assert.Equal(t, "test", graphV1.Schema().Name)

	// v2 should have the updated name
	graphV2, err := svc.FindByIDAndVersion(schema.ID, 2)
	require.NoError(t, err)
	assert.Equal(t, "V2 Name", graphV2.Schema().Name)
}

// TestGraphService_FindByIDAndVersion_NotFound errors for missing version.
func TestGraphService_FindByIDAndVersion_NotFound(t *testing.T) {
	svc := newVersioningGraphService(t)
	schema := mocks.SmallTestGraphSchema()
	_, err := svc.Upsert(schema.ID, schema)
	require.NoError(t, err)

	_, err = svc.FindByIDAndVersion(schema.ID, 99)
	assert.ErrorIs(t, err, repositories.ErrSchemaVersionNotFound)
}

// TestGraphService_GetVersionHistory returns accurate metadata.
func TestGraphService_GetVersionHistory(t *testing.T) {
	svc := newVersioningGraphService(t)
	schema := mocks.SmallTestGraphSchema()

	_, err := svc.Upsert(schema.ID, schema)
	require.NoError(t, err)
	_, err = svc.Upsert(schema.ID, schema)
	require.NoError(t, err)

	history, err := svc.GetVersionHistory(schema.ID)
	require.NoError(t, err)
	assert.Equal(t, schema.ID, history.SchemaID)
	assert.Equal(t, 2, history.LatestVersion)
	assert.Equal(t, 2, history.ActiveVersion)
	assert.Equal(t, 2, history.TotalVersions)
}

// TestGraphService_SetActiveVersion activates an older version.
func TestGraphService_SetActiveVersion(t *testing.T) {
	svc := newVersioningGraphService(t)
	schema := mocks.SmallTestGraphSchema()

	_, err := svc.Upsert(schema.ID, schema)
	require.NoError(t, err)
	_, err = svc.Upsert(schema.ID, schema)
	require.NoError(t, err)

	// Activate v1
	require.NoError(t, svc.SetActiveVersion(schema.ID, 1))

	history, err := svc.GetVersionHistory(schema.ID)
	require.NoError(t, err)
	assert.Equal(t, 1, history.ActiveVersion)
}

// TestGraphService_SetActiveVersion_AlreadyActive is idempotent.
func TestGraphService_SetActiveVersion_AlreadyActive(t *testing.T) {
	svc := newVersioningGraphService(t)
	schema := mocks.SmallTestGraphSchema()

	_, err := svc.Upsert(schema.ID, schema)
	require.NoError(t, err)

	// Activating the already-active version is a no-op
	require.NoError(t, svc.SetActiveVersion(schema.ID, 1))

	history, err := svc.GetVersionHistory(schema.ID)
	require.NoError(t, err)
	assert.Equal(t, 1, history.ActiveVersion)
}

// TestGraphService_SetActiveVersion_NotFound errors for missing version.
func TestGraphService_SetActiveVersion_NotFound(t *testing.T) {
	svc := newVersioningGraphService(t)
	schema := mocks.SmallTestGraphSchema()

	_, err := svc.Upsert(schema.ID, schema)
	require.NoError(t, err)

	err = svc.SetActiveVersion(schema.ID, 99)
	assert.ErrorIs(t, err, repositories.ErrSchemaVersionNotFound)
}

// TestGraphService_Rollback creates a new version with old content and activates it.
func TestGraphService_Rollback(t *testing.T) {
	svc := newVersioningGraphService(t)
	schema := mocks.SmallTestGraphSchema()

	// Create v1
	_, err := svc.Upsert(schema.ID, schema)
	require.NoError(t, err)

	// Create v2 with a name change
	schema.Name = "Broken Version"
	_, err = svc.Upsert(schema.ID, schema)
	require.NoError(t, err)

	// Rollback to v1
	sv, err := svc.Rollback(schema.ID, 1, "rolling back due to bug in v2")
	require.NoError(t, err)
	assert.Equal(t, 3, sv.Version)
	assert.Equal(t, "test", sv.Schema.Name) // content from v1, not "Broken Version"
	assert.Equal(t, "rolling back due to bug in v2", sv.Comment)
	assert.True(t, sv.IsActive)

	// FindByID should return the rolled-back (v1) schema
	activeGraph, err := svc.FindByID(schema.ID)
	require.NoError(t, err)
	assert.Equal(t, "test", activeGraph.Schema().Name)

	// History should show v3 as latest and active
	history, err := svc.GetVersionHistory(schema.ID)
	require.NoError(t, err)
	assert.Equal(t, 3, history.LatestVersion)
	assert.Equal(t, 3, history.ActiveVersion)
	assert.Equal(t, 3, history.TotalVersions)
}

// TestGraphService_Rollback_NotFound errors for a missing source version.
func TestGraphService_Rollback_NotFound(t *testing.T) {
	svc := newVersioningGraphService(t)
	schema := mocks.SmallTestGraphSchema()

	_, err := svc.Upsert(schema.ID, schema)
	require.NoError(t, err)

	_, err = svc.Rollback(schema.ID, 99, "")
	assert.ErrorIs(t, err, repositories.ErrSchemaVersionNotFound)
}

// TestGraphService_Rollback_ToCurrent creates new version even when rolling back to current.
func TestGraphService_Rollback_ToCurrent(t *testing.T) {
	svc := newVersioningGraphService(t)
	schema := mocks.SmallTestGraphSchema()

	_, err := svc.Upsert(schema.ID, schema)
	require.NoError(t, err)

	// Rolling back to the current version creates a redundant new version
	sv, err := svc.Rollback(schema.ID, 1, "rollback to current")
	require.NoError(t, err)
	assert.Equal(t, 2, sv.Version)

	history, err := svc.GetVersionHistory(schema.ID)
	require.NoError(t, err)
	assert.Equal(t, 2, history.TotalVersions)
}

// TestGraphService_FindByID_ReturnsActiveVersion verifies FindByID always returns the active version.
func TestGraphService_FindByID_ReturnsActiveVersion(t *testing.T) {
	svc := newVersioningGraphService(t)
	schema := mocks.SmallTestGraphSchema()

	_, err := svc.Upsert(schema.ID, schema)
	require.NoError(t, err)

	schema.Name = "V2 Name"
	_, err = svc.Upsert(schema.ID, schema)
	require.NoError(t, err)

	// Default: v2 is active
	g, err := svc.FindByID(schema.ID)
	require.NoError(t, err)
	assert.Equal(t, "V2 Name", g.Schema().Name)

	// Activate v1
	require.NoError(t, svc.SetActiveVersion(schema.ID, 1))

	// Now FindByID should return v1's content
	g, err = svc.FindByID(schema.ID)
	require.NoError(t, err)
	assert.Equal(t, "test", g.Schema().Name)
}
