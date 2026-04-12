package services_test

// Integration tests for the full workflow versioning lifecycle.
// These tests exercise the service layer end-to-end using in-memory repositories.

import (
	"testing"

	"github.com/open-source-cloud/fuse/internal/mocks"
	"github.com/open-source-cloud/fuse/internal/packages"
	"github.com/open-source-cloud/fuse/internal/repositories"
	"github.com/open-source-cloud/fuse/internal/services"
	"github.com/open-source-cloud/fuse/internal/workflow"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupVersioningService(t *testing.T) services.GraphService {
	t.Helper()
	repo := repositories.NewMemoryGraphRepository()
	pkgRegistry := packages.NewPackageRegistry()
	internalPkgs := packages.NewInternal()
	pkgSvc := services.NewPackageService(repositories.NewMemoryPackageRepository(), pkgRegistry, internalPkgs)
	require.NoError(t, pkgSvc.RegisterInternalPackages())
	return services.NewGraphService(repo, pkgRegistry, nil)
}

// TestVersioning_FullLifecycle exercises create → update → rollback.
func TestVersioning_FullLifecycle(t *testing.T) {
	svc := setupVersioningService(t)
	schema := mocks.SmallTestGraphSchema()
	schemaID := schema.ID

	// Step 1: Create schema → version 1
	_, err := svc.Upsert(schemaID, schema)
	require.NoError(t, err)

	v, err := svc.ListVersions(schemaID)
	require.NoError(t, err)
	require.Len(t, v, 1)
	assert.Equal(t, 1, v[0].Version)
	assert.True(t, v[0].IsActive)

	// Step 2: Update schema → version 2
	schema.Name = "Version 2"
	_, err = svc.Upsert(schemaID, schema)
	require.NoError(t, err)

	v, err = svc.ListVersions(schemaID)
	require.NoError(t, err)
	require.Len(t, v, 2)
	assert.Equal(t, 2, v[1].Version)
	assert.True(t, v[1].IsActive)
	assert.False(t, v[0].IsActive)

	// Pinned: workflow running on v1 can still retrieve it
	gV1, err := svc.FindByIDAndVersion(schemaID, 1)
	require.NoError(t, err)
	assert.Equal(t, "test", gV1.Schema().Name)

	// Active graph is v2
	gActive, err := svc.FindByID(schemaID)
	require.NoError(t, err)
	assert.Equal(t, "Version 2", gActive.Schema().Name)

	// Step 3: Rollback to v1 → creates v3
	sv, err := svc.Rollback(schemaID, 1, "v2 had a bug")
	require.NoError(t, err)
	assert.Equal(t, 3, sv.Version)
	assert.True(t, sv.IsActive)
	assert.Equal(t, "test", sv.Schema.Name)

	// Active graph is now v3 (=v1 content)
	gActive, err = svc.FindByID(schemaID)
	require.NoError(t, err)
	assert.Equal(t, "test", gActive.Schema().Name)

	// History reflects 3 total versions, v3 active
	h, err := svc.GetVersionHistory(schemaID)
	require.NoError(t, err)
	assert.Equal(t, 3, h.TotalVersions)
	assert.Equal(t, 3, h.LatestVersion)
	assert.Equal(t, 3, h.ActiveVersion)

	// Step 4: Activate v2 explicitly
	require.NoError(t, svc.SetActiveVersion(schemaID, 2))

	gActive, err = svc.FindByID(schemaID)
	require.NoError(t, err)
	assert.Equal(t, "Version 2", gActive.Schema().Name)

	h, err = svc.GetVersionHistory(schemaID)
	require.NoError(t, err)
	assert.Equal(t, 2, h.ActiveVersion)
	assert.Equal(t, 3, h.TotalVersions) // total unchanged
}

// TestVersioning_ExistingSchema_MigrationPath simulates an existing schema with no version history.
// A graph saved directly to the repo (bypassing the service) has no version tracking.
// The first service Upsert call triggers version 1 creation from the new schema content.
func TestVersioning_ExistingSchema_MigrationPath(t *testing.T) {
	repo := repositories.NewMemoryGraphRepository()
	pkgRegistry := packages.NewPackageRegistry()
	internalPkgs := packages.NewInternal()
	pkgSvc := services.NewPackageService(repositories.NewMemoryPackageRepository(), pkgRegistry, internalPkgs)
	require.NoError(t, pkgSvc.RegisterInternalPackages())

	// Simulate pre-migration: save a graph directly into the repo without version tracking
	schema := mocks.SmallTestGraphSchema()
	graph, err := workflow.NewGraph(schema)
	require.NoError(t, err)
	require.NoError(t, repo.Save(graph))

	svc := services.NewGraphService(repo, pkgRegistry, nil)

	// GetVersionHistory on a schema with no versions returns zeroed state
	h, err := svc.GetVersionHistory(schema.ID)
	require.NoError(t, err)
	assert.Equal(t, 0, h.TotalVersions)

	// First update via the service creates version 1
	schema.Name = "Post-migration update"
	_, err = svc.Upsert(schema.ID, schema)
	require.NoError(t, err)

	versions, err := svc.ListVersions(schema.ID)
	require.NoError(t, err)
	require.Len(t, versions, 1, "first upsert on legacy schema creates version 1")
	assert.Equal(t, 1, versions[0].Version)
}

// TestVersioning_MultipleSchemas version tracking is per-schema.
func TestVersioning_MultipleSchemas(t *testing.T) {
	svc := setupVersioningService(t)

	schemaA := mocks.SmallTestGraphSchema()
	schemaA.ID = "schema-a"

	schemaB := mocks.SmallTestGraphSchema()
	schemaB.ID = "schema-b"

	_, err := svc.Upsert(schemaA.ID, schemaA)
	require.NoError(t, err)
	_, err = svc.Upsert(schemaB.ID, schemaB)
	require.NoError(t, err)

	// Update A twice
	_, err = svc.Upsert(schemaA.ID, schemaA)
	require.NoError(t, err)
	_, err = svc.Upsert(schemaA.ID, schemaA)
	require.NoError(t, err)

	hA, err := svc.GetVersionHistory(schemaA.ID)
	require.NoError(t, err)
	assert.Equal(t, 3, hA.TotalVersions)

	hB, err := svc.GetVersionHistory(schemaB.ID)
	require.NoError(t, err)
	assert.Equal(t, 1, hB.TotalVersions)
}
