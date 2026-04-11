package postgres

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/open-source-cloud/fuse/internal/repositories"
)

// ClaimRepository implements ClaimRepository backed by PostgreSQL.
type ClaimRepository struct {
	pool         *pgxpool.Pool
	leaseTimeout time.Duration
}

// NewClaimRepository creates a new PostgreSQL-backed ClaimRepository.
// leaseTimeout controls how long a node may hold a claim before it is considered stale.
func NewClaimRepository(pool *pgxpool.Pool, leaseTimeout time.Duration) repositories.ClaimRepository {
	return &ClaimRepository{pool: pool, leaseTimeout: leaseTimeout}
}

// ClaimWorkflows atomically claims unclaimed or lease-expired workflows for the given node.
func (r *ClaimRepository) ClaimWorkflows(nodeID string, limit int) ([]repositories.ClaimedWorkflow, error) {
	ctx := context.Background()

	rows, err := r.pool.Query(ctx, `
		UPDATE workflows
		SET claimed_by = $1, claimed_at = NOW(), updated_at = NOW()
		WHERE id IN (
			SELECT id FROM workflows
			WHERE (claimed_by IS NULL AND state IN ('untriggered', 'running', 'sleeping'))
			   OR (claimed_by IS NOT NULL AND claimed_by != $1
			       AND claimed_at < NOW() - INTERVAL '1 second' * $3
			       AND state IN ('running', 'sleeping'))
			ORDER BY id
			FOR UPDATE SKIP LOCKED
			LIMIT $2
		)
		RETURNING workflow_id, schema_id, state::TEXT
	`, nodeID, limit, r.leaseTimeout.Seconds())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var claimed []repositories.ClaimedWorkflow
	for rows.Next() {
		var cw repositories.ClaimedWorkflow
		if scanErr := rows.Scan(&cw.WorkflowID, &cw.SchemaID, &cw.State); scanErr != nil {
			return nil, scanErr
		}
		claimed = append(claimed, cw)
	}
	return claimed, rows.Err()
}

// ReleaseWorkflows releases all workflows claimed by the given node.
func (r *ClaimRepository) ReleaseWorkflows(nodeID string) error {
	ctx := context.Background()
	_, err := r.pool.Exec(ctx, `
		UPDATE workflows
		SET claimed_by = NULL, claimed_at = NULL, updated_at = NOW()
		WHERE claimed_by = $1
	`, nodeID)
	return err
}

// Heartbeat upserts the node's heartbeat record.
func (r *ClaimRepository) Heartbeat(nodeID string, host string, port int) error {
	ctx := context.Background()
	_, err := r.pool.Exec(ctx, `
		INSERT INTO node_heartbeats (node_id, host, port, started_at, last_seen)
		VALUES ($1, $2, $3, NOW(), NOW())
		ON CONFLICT (node_id) DO UPDATE SET last_seen = NOW(), host = $2, port = $3
	`, nodeID, host, port)
	return err
}

// FindStaleNodes returns node IDs whose last heartbeat is older than the given timeout.
func (r *ClaimRepository) FindStaleNodes(timeout time.Duration) ([]string, error) {
	ctx := context.Background()
	rows, err := r.pool.Query(ctx, `
		SELECT node_id FROM node_heartbeats
		WHERE last_seen < NOW() - INTERVAL '1 second' * $1
	`, timeout.Seconds())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stale []string
	for rows.Next() {
		var nodeID string
		if scanErr := rows.Scan(&nodeID); scanErr != nil {
			return nil, scanErr
		}
		stale = append(stale, nodeID)
	}
	return stale, rows.Err()
}

// ReassignFromStaleNodes releases workflows claimed by stale nodes.
func (r *ClaimRepository) ReassignFromStaleNodes(staleNodeIDs []string) (int, error) {
	if len(staleNodeIDs) == 0 {
		return 0, nil
	}
	ctx := context.Background()
	tag, err := r.pool.Exec(ctx, `
		UPDATE workflows
		SET claimed_by = NULL, claimed_at = NULL, updated_at = NOW()
		WHERE claimed_by = ANY($1)
		  AND state IN ('running', 'sleeping')
	`, staleNodeIDs)
	if err != nil {
		return 0, err
	}
	return int(tag.RowsAffected()), nil
}
