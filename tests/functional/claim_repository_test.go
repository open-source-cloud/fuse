//go:build functional

package functional_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/open-source-cloud/fuse/internal/repositories"
	internalworkflow "github.com/open-source-cloud/fuse/internal/workflow"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Note: ClaimRepository has no memory contract test because the memory implementation
// is a no-op stub (all methods return nil/empty). The real logic only exists in PostgreSQL.

func contractTestClaimRepository(
	t *testing.T,
	newClaimRepo func() repositories.ClaimRepository,
	wfRepo repositories.WorkflowRepository,
	graphRepo repositories.GraphRepository,
	pool *pgxpool.Pool,
	reset func(),
) {
	t.Helper()

	// seedWorkflows creates n workflows in the given state and returns their IDs.
	seedWorkflows := func(t *testing.T, n int, state internalworkflow.State) []string {
		t.Helper()
		ids := make([]string, n)
		for i := range n {
			wf := newTestWorkflow(t)
			wf.SetState(state)
			require.NoError(t, graphRepo.Save(wf.Graph()))
			require.NoError(t, wfRepo.Save(wf))
			ids[i] = wf.ID().String()
		}
		return ids
	}

	t.Run("Heartbeat upserts node record without error", func(t *testing.T) {
		reset()
		repo := newClaimRepo()

		require.NoError(t, repo.Heartbeat("node-hb-1", "10.0.0.1", 9090))
		// Second call should upsert without error
		require.NoError(t, repo.Heartbeat("node-hb-1", "10.0.0.2", 9091))
	})

	t.Run("ClaimWorkflows claims unclaimed workflows", func(t *testing.T) {
		reset()
		repo := newClaimRepo()
		seedWorkflows(t, 3, internalworkflow.StateUntriggered)

		claimed, err := repo.ClaimWorkflows("node-A", 2)
		require.NoError(t, err)
		assert.Len(t, claimed, 2)

		// Remaining 1 should be claimable
		claimed2, err := repo.ClaimWorkflows("node-A", 5)
		require.NoError(t, err)
		assert.Len(t, claimed2, 1)
	})

	t.Run("ClaimWorkflows skips already-claimed workflows", func(t *testing.T) {
		reset()
		repo := newClaimRepo()
		seedWorkflows(t, 2, internalworkflow.StateRunning)

		claimed, err := repo.ClaimWorkflows("node-A", 10)
		require.NoError(t, err)
		assert.Len(t, claimed, 2)

		// Same node re-claiming should get 0 (already claimed by node-A, lease not expired)
		claimed2, err := repo.ClaimWorkflows("node-A", 10)
		require.NoError(t, err)
		assert.Empty(t, claimed2)
	})

	t.Run("ClaimWorkflows ignores finished workflows", func(t *testing.T) {
		reset()
		repo := newClaimRepo()
		seedWorkflows(t, 2, internalworkflow.StateFinished)

		claimed, err := repo.ClaimWorkflows("node-A", 10)
		require.NoError(t, err)
		assert.Empty(t, claimed)
	})

	t.Run("ReleaseWorkflows releases all claims for a node", func(t *testing.T) {
		reset()
		repo := newClaimRepo()
		seedWorkflows(t, 3, internalworkflow.StateRunning)

		_, err := repo.ClaimWorkflows("node-A", 3)
		require.NoError(t, err)

		require.NoError(t, repo.ReleaseWorkflows("node-A"))

		// Another node should now be able to claim them
		claimed, err := repo.ClaimWorkflows("node-B", 10)
		require.NoError(t, err)
		assert.Len(t, claimed, 3)
	})

	t.Run("FindStaleNodes returns nodes with old heartbeats", func(t *testing.T) {
		reset()
		repo := newClaimRepo()

		// Fresh heartbeat
		require.NoError(t, repo.Heartbeat("node-fresh", "10.0.0.1", 9090))

		// Insert a stale heartbeat directly via SQL
		_, err := pool.Exec(context.Background(),
			`INSERT INTO node_heartbeats (node_id, host, port, started_at, last_seen)
			 VALUES ($1, $2, $3, NOW() - INTERVAL '2 hours', NOW() - INTERVAL '2 hours')`,
			"node-stale", "10.0.0.2", 9091)
		require.NoError(t, err)

		stale, err := repo.FindStaleNodes(30 * time.Second)
		require.NoError(t, err)
		assert.Contains(t, stale, "node-stale")
		assert.NotContains(t, stale, "node-fresh")
	})

	t.Run("ReassignFromStaleNodes releases stale node workflows", func(t *testing.T) {
		reset()
		repo := newClaimRepo()
		seedWorkflows(t, 2, internalworkflow.StateRunning)

		// Claim by stale node
		_, err := repo.ClaimWorkflows("node-stale", 2)
		require.NoError(t, err)

		count, err := repo.ReassignFromStaleNodes([]string{"node-stale"})
		require.NoError(t, err)
		assert.Equal(t, 2, count)

		// Workflows should now be claimable by another node
		claimed, err := repo.ClaimWorkflows("node-healthy", 10)
		require.NoError(t, err)
		assert.Len(t, claimed, 2)
	})

	t.Run("ReassignFromStaleNodes with empty list returns 0", func(t *testing.T) {
		repo := newClaimRepo()
		count, err := repo.ReassignFromStaleNodes([]string{})
		require.NoError(t, err)
		assert.Equal(t, 0, count)
	})

	t.Run("ReassignFromStaleNodes ignores finished workflows", func(t *testing.T) {
		reset()
		repo := newClaimRepo()

		// Create finished workflows and manually set claimed_by
		seedWorkflows(t, 2, internalworkflow.StateFinished)
		_, err := pool.Exec(context.Background(),
			fmt.Sprintf(`UPDATE workflows SET claimed_by = 'node-dead', claimed_at = NOW()`))
		require.NoError(t, err)

		count, err := repo.ReassignFromStaleNodes([]string{"node-dead"})
		require.NoError(t, err)
		assert.Equal(t, 0, count)
	})
}
