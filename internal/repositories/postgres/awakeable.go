package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/open-source-cloud/fuse/internal/repositories"
	"github.com/open-source-cloud/fuse/internal/workflow"
	pkgwf "github.com/open-source-cloud/fuse/pkg/workflow"

	"github.com/open-source-cloud/fuse/pkg/objectstore"
)

// AwakeableRepository implements AwakeableRepository backed by PostgreSQL + ObjectStore.
type AwakeableRepository struct {
	pool  *pgxpool.Pool
	store objectstore.ObjectStore
}

// NewAwakeableRepository creates a new PostgreSQL-backed AwakeableRepository.
func NewAwakeableRepository(pool *pgxpool.Pool, store objectstore.ObjectStore) repositories.AwakeableRepository {
	return &AwakeableRepository{pool: pool, store: store}
}

func awakeableResultKey(id string) string {
	return fmt.Sprintf("awakeables/%s/result.json", id)
}

// Save stores an awakeable.
func (r *AwakeableRepository) Save(awakeable *workflow.Awakeable) error {
	ctx := context.Background()

	var resultRef *string
	if awakeable.Result != nil {
		key := awakeableResultKey(awakeable.ID)
		data, err := json.Marshal(awakeable.Result)
		if err != nil {
			return fmt.Errorf("postgres/awakeable: marshal result: %w", err)
		}
		if err := r.store.Put(ctx, key, data); err != nil {
			return fmt.Errorf("postgres/awakeable: put result: %w", err)
		}
		resultRef = &key
	}

	var timeoutNs *int64
	if awakeable.Timeout > 0 {
		ns := int64(awakeable.Timeout / time.Nanosecond)
		timeoutNs = &ns
	}

	var deadlineAt *time.Time
	if !awakeable.DeadlineAt.IsZero() {
		deadlineAt = &awakeable.DeadlineAt
	}

	_, err := r.pool.Exec(ctx, `
		INSERT INTO awakeables (
			awakeable_id, workflow_id, exec_id, thread_id,
			status, timeout_ns, deadline_at, result_ref, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW())
		ON CONFLICT (awakeable_id) DO UPDATE SET
			status = EXCLUDED.status,
			timeout_ns = EXCLUDED.timeout_ns,
			deadline_at = EXCLUDED.deadline_at,
			result_ref = EXCLUDED.result_ref,
			updated_at = NOW()
	`,
		awakeable.ID, awakeable.WorkflowID.String(), awakeable.ExecID.String(),
		safeUint16ToInt16(awakeable.ThreadID), string(awakeable.Status),
		timeoutNs, deadlineAt, resultRef, awakeable.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("postgres/awakeable: save: %w", err)
	}
	return nil
}

// FindByID retrieves an awakeable by its ID.
func (r *AwakeableRepository) FindByID(id string) (*workflow.Awakeable, error) {
	ctx := context.Background()

	a, resultRef, err := r.scanAwakeable(ctx,
		`SELECT awakeable_id, workflow_id, exec_id, thread_id, status,
		        timeout_ns, deadline_at, result_ref, created_at
		 FROM awakeables WHERE awakeable_id = $1`, id)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, repositories.ErrAwakeableNotFound
		}
		return nil, fmt.Errorf("postgres/awakeable: find by id: %w", err)
	}

	if resultRef != nil {
		if err := r.fetchResult(ctx, *resultRef, a); err != nil {
			return nil, err
		}
	}
	return a, nil
}

// FindPending retrieves all pending awakeables for a given workflow.
func (r *AwakeableRepository) FindPending(workflowID string) ([]*workflow.Awakeable, error) {
	ctx := context.Background()

	rows, err := r.pool.Query(ctx, `
		SELECT awakeable_id, workflow_id, exec_id, thread_id, status,
		       timeout_ns, deadline_at, result_ref, created_at
		FROM awakeables
		WHERE workflow_id = $1 AND status = 'pending'
	`, workflowID)
	if err != nil {
		return nil, fmt.Errorf("postgres/awakeable: find pending: %w", err)
	}
	defer rows.Close()

	var result []*workflow.Awakeable
	for rows.Next() {
		a, _, err := r.scanAwakeableRow(rows)
		if err != nil {
			return nil, fmt.Errorf("postgres/awakeable: scan row: %w", err)
		}
		result = append(result, a)
	}
	return result, rows.Err()
}

// Resolve resolves an awakeable with the given result data.
func (r *AwakeableRepository) Resolve(id string, result map[string]any) error {
	ctx := context.Background()

	// Store result in object store
	key := awakeableResultKey(id)
	data, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("postgres/awakeable: marshal result: %w", err)
	}
	if err := r.store.Put(ctx, key, data); err != nil {
		return fmt.Errorf("postgres/awakeable: put result: %w", err)
	}

	// Atomically update only if still pending
	tag, err := r.pool.Exec(ctx, `
		UPDATE awakeables SET status = 'resolved', result_ref = $1, updated_at = NOW()
		WHERE awakeable_id = $2 AND status = 'pending'
	`, key, id)
	if err != nil {
		return fmt.Errorf("postgres/awakeable: resolve: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return repositories.ErrAwakeableNotFound
	}
	return nil
}

// scanAwakeable executes a single-row query and scans into an Awakeable.
func (r *AwakeableRepository) scanAwakeable(ctx context.Context, query string, args ...any) (*workflow.Awakeable, *string, error) {
	row := r.pool.QueryRow(ctx, query, args...)
	return r.scanAwakeableFromRow(row)
}

func (r *AwakeableRepository) scanAwakeableFromRow(row pgx.Row) (*workflow.Awakeable, *string, error) {
	var (
		a          workflow.Awakeable
		wfID       string
		execID     string
		threadID   int16
		status     string
		timeoutNs  *int64
		deadlineAt *time.Time
		resultRef  *string
	)

	err := row.Scan(
		&a.ID, &wfID, &execID, &threadID, &status,
		&timeoutNs, &deadlineAt, &resultRef, &a.CreatedAt,
	)
	if err != nil {
		return nil, nil, err
	}

	a.WorkflowID = pkgwf.ID(wfID)
	a.ExecID = pkgwf.ExecID(execID)
	a.ThreadID = safeInt16ToUint16(threadID)
	a.Status = workflow.AwakeableStatus(status)
	if timeoutNs != nil {
		a.Timeout = time.Duration(*timeoutNs)
	}
	if deadlineAt != nil {
		a.DeadlineAt = *deadlineAt
	}
	return &a, resultRef, nil
}

// scanAwakeableRow scans a rows.Next() result.
func (r *AwakeableRepository) scanAwakeableRow(rows pgx.Rows) (*workflow.Awakeable, *string, error) {
	var (
		a          workflow.Awakeable
		wfID       string
		execID     string
		threadID   int16
		status     string
		timeoutNs  *int64
		deadlineAt *time.Time
		resultRef  *string
	)

	err := rows.Scan(
		&a.ID, &wfID, &execID, &threadID, &status,
		&timeoutNs, &deadlineAt, &resultRef, &a.CreatedAt,
	)
	if err != nil {
		return nil, nil, err
	}

	a.WorkflowID = pkgwf.ID(wfID)
	a.ExecID = pkgwf.ExecID(execID)
	a.ThreadID = safeInt16ToUint16(threadID)
	a.Status = workflow.AwakeableStatus(status)
	if timeoutNs != nil {
		a.Timeout = time.Duration(*timeoutNs)
	}
	if deadlineAt != nil {
		a.DeadlineAt = *deadlineAt
	}
	return &a, resultRef, nil
}

func (r *AwakeableRepository) fetchResult(ctx context.Context, ref string, a *workflow.Awakeable) error {
	data, err := r.store.Get(ctx, ref)
	if err != nil {
		return fmt.Errorf("postgres/awakeable: get result %q: %w", ref, err)
	}
	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		return fmt.Errorf("postgres/awakeable: unmarshal result: %w", err)
	}
	a.Result = result
	return nil
}
