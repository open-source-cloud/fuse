package postgres

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/open-source-cloud/fuse/internal/repositories"
	"github.com/open-source-cloud/fuse/internal/workflow"
	pkgwf "github.com/open-source-cloud/fuse/pkg/workflow"

	"github.com/open-source-cloud/fuse/pkg/objectstore"
)

// TraceRepository implements TraceRepository backed by PostgreSQL + ObjectStore.
type TraceRepository struct {
	pool  *pgxpool.Pool
	store objectstore.ObjectStore
}

// NewTraceRepository creates a new PostgreSQL-backed TraceRepository.
func NewTraceRepository(pool *pgxpool.Pool, store objectstore.ObjectStore) repositories.TraceRepository {
	return &TraceRepository{pool: pool, store: store}
}

func traceObjectKey(workflowID, execID, suffix string) string {
	return fmt.Sprintf("workflows/%s/trace/%s/%s", workflowID, execID, suffix)
}

// Save persists or updates a workflow execution trace.
func (r *TraceRepository) Save(trace *workflow.ExecutionTrace) error {
	ctx := context.Background()

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("postgres/trace: begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Upsert the trace header
	_, err = tx.Exec(ctx, `
		INSERT INTO execution_traces (
			workflow_id, schema_id, status, triggered_at, completed_at,
			duration, error, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, NOW())
		ON CONFLICT (workflow_id) DO UPDATE SET
			status       = EXCLUDED.status,
			completed_at = EXCLUDED.completed_at,
			duration     = EXCLUDED.duration,
			error        = EXCLUDED.error,
			updated_at   = NOW()
	`,
		trace.WorkflowID, trace.SchemaID, trace.Status.String(),
		trace.TriggeredAt, trace.CompletedAt,
		trace.Duration, trace.Error,
	)
	if err != nil {
		return fmt.Errorf("postgres/trace: upsert header: %w", err)
	}

	// Delete existing steps (full replace on each save)
	_, err = tx.Exec(ctx, `DELETE FROM execution_trace_steps WHERE workflow_id = $1`, trace.WorkflowID)
	if err != nil {
		return fmt.Errorf("postgres/trace: delete steps: %w", err)
	}

	// Insert steps with payloads in object store
	for _, step := range trace.Steps {
		var inputRef, outputRef *string

		if step.Input != nil {
			ref, putErr := r.putJSON(ctx, traceObjectKey(trace.WorkflowID, step.ExecID, "input.json"), step.Input)
			if putErr != nil {
				return fmt.Errorf("postgres/trace: put step input: %w", putErr)
			}
			inputRef = &ref
		}

		if step.Output != nil {
			ref, putErr := r.putJSON(ctx, traceObjectKey(trace.WorkflowID, step.ExecID, "output.json"), step.Output)
			if putErr != nil {
				return fmt.Errorf("postgres/trace: put step output: %w", putErr)
			}
			outputRef = &ref
		}

		_, err = tx.Exec(ctx, `
			INSERT INTO execution_trace_steps (
				workflow_id, exec_id, thread_id, function_node_id,
				started_at, completed_at, duration, input_ref, output_ref,
				status, attempt, error
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		`,
			trace.WorkflowID, step.ExecID, safeUint16ToInt16(step.ThreadID),
			step.FunctionNodeID, step.StartedAt, step.CompletedAt,
			step.Duration, inputRef, outputRef,
			step.Status, step.Attempt, step.Error,
		)
		if err != nil {
			return fmt.Errorf("postgres/trace: insert step %s: %w", step.ExecID, err)
		}
	}

	return tx.Commit(ctx)
}

// FindByWorkflowID retrieves the trace for a specific workflow execution.
func (r *TraceRepository) FindByWorkflowID(workflowID string) (*workflow.ExecutionTrace, error) {
	ctx := context.Background()

	// Fetch header
	row := r.pool.QueryRow(ctx, `
		SELECT workflow_id, schema_id, status::TEXT, triggered_at, completed_at,
		       duration, error
		FROM execution_traces
		WHERE workflow_id = $1
	`, workflowID)

	var t workflow.ExecutionTrace
	var statusStr string
	err := row.Scan(
		&t.WorkflowID, &t.SchemaID, &statusStr,
		&t.TriggeredAt, &t.CompletedAt,
		&t.Duration, &t.Error,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, repositories.ErrTraceNotFound
		}
		return nil, fmt.Errorf("postgres/trace: find header: %w", err)
	}
	t.Status = workflow.State(statusStr)

	// Fetch steps
	steps, err := r.loadSteps(ctx, workflowID)
	if err != nil {
		return nil, err
	}
	t.Steps = steps

	return &t, nil
}

// FindBySchemaID retrieves traces for all executions of a schema with filtering and pagination.
func (r *TraceRepository) FindBySchemaID(schemaID string, opts repositories.TraceQueryOpts) (*repositories.TraceQueryResult, error) {
	ctx := context.Background()

	// Build WHERE clause
	where := `WHERE schema_id = $1`
	args := []any{schemaID}
	idx := 2

	if opts.Status != nil {
		where += fmt.Sprintf(` AND status = $%d::workflow_state`, idx)
		args = append(args, *opts.Status)
		idx++
	}
	if opts.Since != nil {
		where += fmt.Sprintf(` AND triggered_at >= $%d`, idx)
		args = append(args, *opts.Since)
		idx++
	}

	// Count
	var total int
	if err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM execution_traces `+where, args...).Scan(&total); err != nil {
		return nil, fmt.Errorf("postgres/trace: count: %w", err)
	}

	if total == 0 {
		return &repositories.TraceQueryResult{Traces: []*workflow.ExecutionTrace{}, Total: 0}, nil
	}

	// Fetch paginated headers
	limit := opts.Limit
	if limit <= 0 {
		limit = 50
	}
	offset := max(opts.Offset, 0)

	query := fmt.Sprintf(`
		SELECT workflow_id, schema_id, status::TEXT, triggered_at, completed_at,
		       duration, error
		FROM execution_traces %s
		ORDER BY triggered_at DESC
		LIMIT $%d OFFSET $%d
	`, where, idx, idx+1)
	args = append(args, limit, offset)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("postgres/trace: find by schema: %w", err)
	}
	defer rows.Close()

	var traces []*workflow.ExecutionTrace
	for rows.Next() {
		var t workflow.ExecutionTrace
		var statusStr string
		if scanErr := rows.Scan(
			&t.WorkflowID, &t.SchemaID, &statusStr,
			&t.TriggeredAt, &t.CompletedAt,
			&t.Duration, &t.Error,
		); scanErr != nil {
			return nil, fmt.Errorf("postgres/trace: scan: %w", scanErr)
		}
		t.Status = workflow.State(statusStr)
		traces = append(traces, &t)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("postgres/trace: rows: %w", err)
	}

	// Load steps for each trace
	for _, t := range traces {
		steps, loadErr := r.loadSteps(ctx, t.WorkflowID)
		if loadErr != nil {
			return nil, loadErr
		}
		t.Steps = steps
	}

	if traces == nil {
		traces = []*workflow.ExecutionTrace{}
	}

	return &repositories.TraceQueryResult{Traces: traces, Total: total}, nil
}

// Delete removes a trace and its steps.
func (r *TraceRepository) Delete(workflowID string) error {
	ctx := context.Background()
	// Steps are cascade-deleted via FK
	_, err := r.pool.Exec(ctx, `DELETE FROM execution_traces WHERE workflow_id = $1`, workflowID)
	if err != nil {
		return fmt.Errorf("postgres/trace: delete: %w", err)
	}
	return nil
}

func (r *TraceRepository) loadSteps(ctx context.Context, workflowID string) ([]workflow.ExecutionStepTrace, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT exec_id, thread_id, function_node_id, started_at, completed_at,
		       duration, input_ref, output_ref, status, attempt, error
		FROM execution_trace_steps
		WHERE workflow_id = $1
		ORDER BY id
	`, workflowID)
	if err != nil {
		return nil, fmt.Errorf("postgres/trace: load steps: %w", err)
	}
	defer rows.Close()

	var steps []workflow.ExecutionStepTrace
	for rows.Next() {
		var s workflow.ExecutionStepTrace
		var threadID int16
		var inputRef, outputRef *string

		if scanErr := rows.Scan(
			&s.ExecID, &threadID, &s.FunctionNodeID,
			&s.StartedAt, &s.CompletedAt, &s.Duration,
			&inputRef, &outputRef,
			&s.Status, &s.Attempt, &s.Error,
		); scanErr != nil {
			return nil, fmt.Errorf("postgres/trace: scan step: %w", scanErr)
		}
		s.ThreadID = safeInt16ToUint16(threadID)

		// Fetch payloads from object store
		if inputRef != nil {
			var m map[string]any
			if getErr := r.getJSON(ctx, *inputRef, &m); getErr != nil {
				return nil, fmt.Errorf("postgres/trace: fetch step input: %w", getErr)
			}
			s.Input = m
		}

		if outputRef != nil {
			var out pkgwf.FunctionOutput
			if getErr := r.getJSON(ctx, *outputRef, &out); getErr != nil {
				return nil, fmt.Errorf("postgres/trace: fetch step output: %w", getErr)
			}
			s.Output = &out
		}

		steps = append(steps, s)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("postgres/trace: steps rows: %w", err)
	}

	if steps == nil {
		steps = []workflow.ExecutionStepTrace{}
	}
	return steps, nil
}

func (r *TraceRepository) putJSON(ctx context.Context, key string, v any) (string, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return "", fmt.Errorf("marshal: %w", err)
	}
	if err := r.store.Put(ctx, key, data); err != nil {
		return "", fmt.Errorf("put %q: %w", key, err)
	}
	return key, nil
}

func (r *TraceRepository) getJSON(ctx context.Context, key string, dst any) error {
	data, err := r.store.Get(ctx, key)
	if err != nil {
		return fmt.Errorf("get %q: %w", key, err)
	}
	return json.Unmarshal(data, dst)
}
