//go:build functional

package functional_test

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/open-source-cloud/fuse/internal/repositories"
	"github.com/open-source-cloud/fuse/internal/repositories/postgres"
	internalworkflow "github.com/open-source-cloud/fuse/internal/workflow"
	"github.com/open-source-cloud/fuse/pkg/objectstore"
	"github.com/open-source-cloud/fuse/pkg/workflow"
	"github.com/stretchr/testify/require"
)

const functionalSchema = "fuse_functional"

// testDSN reads DB_POSTGRES_DSN from the environment.
func testDSN(t *testing.T) string {
	t.Helper()
	dsn := os.Getenv("DB_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("DB_POSTGRES_DSN not set, skipping postgres functional tests")
	}
	return dsn
}

// withSearchPath appends search_path to the DSN so all operations target the test schema.
func withSearchPath(dsn, schema string) string {
	u, err := url.Parse(dsn)
	if err != nil {
		return dsn
	}
	q := u.Query()
	q.Set("search_path", schema)
	u.RawQuery = q.Encode()
	return u.String()
}

// setupTestPool creates the fuse_functional schema, runs migrations into it,
// truncates all tables, and returns a pool scoped to that schema.
func setupTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	baseDSN := testDSN(t)

	// Connect with the base DSN first to create the schema
	basePool, err := pgxpool.New(context.Background(), baseDSN)
	require.NoError(t, err)
	require.NoError(t, basePool.Ping(context.Background()))

	_, err = basePool.Exec(context.Background(), fmt.Sprintf("CREATE SCHEMA IF NOT EXISTS %s", functionalSchema))
	require.NoError(t, err)
	basePool.Close()

	// Now connect with search_path set to the test schema
	schemaDSN := withSearchPath(baseDSN, functionalSchema)

	pool, err := pgxpool.New(context.Background(), schemaDSN)
	require.NoError(t, err)
	require.NoError(t, pool.Ping(context.Background()))

	// Run migrations into the test schema
	require.NoError(t, postgres.RunMigrations(schemaDSN))

	// Truncate all tables for a clean slate (respects FK order via CASCADE)
	_, err = pool.Exec(context.Background(), `
		TRUNCATE TABLE
			journal_entries,
			sub_workflow_refs,
			awakeables,
			node_heartbeats,
			graph_schema_nodes,
			graph_schema_metadata,
			graph_schema_tags,
			package_functions,
			package_tags,
			workflows,
			graph_schemas,
			packages
		CASCADE
	`)
	require.NoError(t, err)

	t.Cleanup(func() { pool.Close() })
	return pool
}

// testObjectStore returns a memory object store for test payloads.
func testObjectStore() objectstore.ObjectStore {
	return objectstore.NewMemoryObjectStore()
}

// pgEnsureWorkflow returns an ensureWorkflowFn that inserts a stub workflow for FK constraints.
func pgEnsureWorkflow(wfRepo repositories.WorkflowRepository, graphRepo repositories.GraphRepository) ensureWorkflowFn {
	return func(t *testing.T, workflowID string) {
		t.Helper()
		wf := newTestWorkflowWithID(t, workflow.ID(workflowID))
		require.NoError(t, graphRepo.Save(wf.Graph()))
		require.NoError(t, wfRepo.Save(wf))
	}
}

// pgEnsureGraph returns an ensureGraphFn that saves the graph schema to the DB.
func pgEnsureGraph(graphRepo repositories.GraphRepository) ensureGraphFn {
	return func(t *testing.T, wf *internalworkflow.Workflow) {
		t.Helper()
		require.NoError(t, graphRepo.Save(wf.Graph()))
	}
}

// --- Postgres Graph Repository ---

func TestPostgresGraphRepository_Contract(t *testing.T) {
	pool := setupTestPool(t)
	store := testObjectStore()
	contractTestGraphRepository(t, func() repositories.GraphRepository {
		return postgres.NewGraphRepository(pool, store)
	})
}

// --- Postgres Journal Repository ---

func TestPostgresJournalRepository_Contract(t *testing.T) {
	pool := setupTestPool(t)
	store := testObjectStore()
	graphRepo := postgres.NewGraphRepository(pool, store)
	wfRepo := postgres.NewWorkflowRepository(pool, store)

	contractTestJournalRepository(t, func() repositories.JournalRepository {
		return postgres.NewJournalRepository(pool, store)
	}, pgEnsureWorkflow(wfRepo, graphRepo))
}

// --- Postgres Workflow Repository ---

func TestPostgresWorkflowRepository_Contract(t *testing.T) {
	pool := setupTestPool(t)
	store := testObjectStore()
	graphRepo := postgres.NewGraphRepository(pool, store)
	contractTestWorkflowRepository(t, func() repositories.WorkflowRepository {
		return postgres.NewWorkflowRepository(pool, store)
	}, pgEnsureGraph(graphRepo))
}

func TestPostgresWorkflowRepository_SubWorkflow_Contract(t *testing.T) {
	pool := setupTestPool(t)
	store := testObjectStore()
	graphRepo := postgres.NewGraphRepository(pool, store)
	contractTestWorkflowSubWorkflowRefs(t, func() repositories.WorkflowRepository {
		return postgres.NewWorkflowRepository(pool, store)
	}, pgEnsureGraph(graphRepo))
}

// --- Postgres Package Repository ---

func TestPostgresPackageRepository_Contract(t *testing.T) {
	pool := setupTestPool(t)
	store := testObjectStore()
	contractTestPackageRepository(t, func() repositories.PackageRepository {
		return postgres.NewPackageRepository(pool, store)
	})
}

// --- Postgres Awakeable Repository ---

func TestPostgresAwakeableRepository_Contract(t *testing.T) {
	pool := setupTestPool(t)
	store := testObjectStore()
	graphRepo := postgres.NewGraphRepository(pool, store)
	wfRepo := postgres.NewWorkflowRepository(pool, store)

	contractTestAwakeableRepository(t, func() repositories.AwakeableRepository {
		return postgres.NewAwakeableRepository(pool, store)
	}, wfRepo, graphRepo)
}
