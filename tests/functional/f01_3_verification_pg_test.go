//go:build functional

package functional_test

// Postgres invocation of the F-01.3 parity contract. The shared body, and what it does and does not
// prove, is documented in f01_3_verification_test.go. Skips silently without DB_POSTGRES_DSN
// (tests/functional/postgres_test.go:25), so a green run here proves nothing unless the DSN was set.

import (
	"context"
	"testing"

	"github.com/open-source-cloud/fuse/internal/repositories"
	"github.com/open-source-cloud/fuse/internal/repositories/postgres"
	"github.com/stretchr/testify/require"
)

func TestPostgresGraphRepository_LiveSchemaEditIsolation_Contract(t *testing.T) {
	pool := setupTestPool(t)
	store := testObjectStore()
	contractTestGraphLiveSchemaEditIsolation(t, func() repositories.GraphRepository {
		return postgres.NewGraphRepository(pool, store)
	}, func() {
		_, err := pool.Exec(context.Background(),
			"TRUNCATE TABLE graph_schema_versions, graph_schema_nodes, graph_schema_metadata, graph_schema_tags, graph_schemas CASCADE")
		require.NoError(t, err)
	})
}
