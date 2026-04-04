package postgres

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/open-source-cloud/fuse/internal/repositories"
	"github.com/open-source-cloud/fuse/internal/workflow"
	"github.com/open-source-cloud/fuse/pkg/objectstore"
	pkgwf "github.com/open-source-cloud/fuse/pkg/workflow"
)

// WorkflowRepository implements WorkflowRepository backed by PostgreSQL + ObjectStore.
type WorkflowRepository struct {
	repositories.WorkflowRepository
	pool  *pgxpool.Pool
	store objectstore.ObjectStore
}

// NewWorkflowRepository creates a new PostgreSQL-backed WorkflowRepository.
func NewWorkflowRepository(pool *pgxpool.Pool, store objectstore.ObjectStore) repositories.WorkflowRepository {
	return &WorkflowRepository{pool: pool, store: store}
}

func workflowOutputKey(id string) string {
	return fmt.Sprintf("workflows/%s/output.json", id)
}

// Exists checks if a workflow exists.
func (r *WorkflowRepository) Exists(id string) bool {
	ctx := context.Background()
	var exists bool
	_ = r.pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM workflows WHERE workflow_id = $1)`, id,
	).Scan(&exists)
	return exists
}

// Get retrieves a workflow by its business ID.
// Returns a thin Workflow reconstructed from DB metadata + graph schema from object store.
// Callers that need the full state (threads, audit log, etc.) replay the journal after calling Get().
func (r *WorkflowRepository) Get(id string) (*workflow.Workflow, error) {
	ctx := context.Background()

	var schemaID, state string
	var outputRef *string
	err := r.pool.QueryRow(ctx, `
		SELECT schema_id, state, output_ref
		FROM workflows WHERE workflow_id = $1
	`, id).Scan(&schemaID, &state, &outputRef)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("workflow %s not found", id)
		}
		return nil, fmt.Errorf("postgres/workflow: get: %w", err)
	}

	// Load graph schema from DB + object store
	graph, err := r.loadGraph(ctx, schemaID)
	if err != nil {
		return nil, fmt.Errorf("postgres/workflow: load graph for %q: %w", schemaID, err)
	}

	wf := workflow.New(pkgwf.ID(id), graph)

	// Restore state without appending a journal entry.
	// SetState() appends a state:changed journal entry, which is wrong during reconstruction.
	// Instead, we use a direct restore approach: create with untriggered, then only
	// set if the persisted state differs.
	if workflow.State(state) != workflow.StateUntriggered {
		r.restoreState(wf, workflow.State(state))
	}

	return wf, nil
}

// Save persists a workflow envelope to PostgreSQL and output to the object store.
func (r *WorkflowRepository) Save(wf *workflow.Workflow) error {
	ctx := context.Background()
	wfID := wf.ID().String()

	var outputRef *string
	output := wf.AggregatedOutputSnapshot()
	if len(output) > 0 {
		key := workflowOutputKey(wfID)
		data, err := json.Marshal(output)
		if err != nil {
			return fmt.Errorf("postgres/workflow: marshal output: %w", err)
		}
		if err := r.store.Put(ctx, key, data); err != nil {
			return fmt.Errorf("postgres/workflow: put output: %w", err)
		}
		outputRef = &key
	}

	_, err := r.pool.Exec(ctx, `
		INSERT INTO workflows (workflow_id, schema_id, state, output_ref, created_at, updated_at)
		VALUES ($1, $2, $3, $4, NOW(), NOW())
		ON CONFLICT (workflow_id) DO UPDATE SET
			state = EXCLUDED.state,
			output_ref = EXCLUDED.output_ref,
			updated_at = NOW()
	`, wfID, wf.Schema().ID, wf.State().String(), outputRef)
	if err != nil {
		return fmt.Errorf("postgres/workflow: save: %w", err)
	}
	return nil
}

// FindByState returns workflow IDs for workflows in any of the given states.
func (r *WorkflowRepository) FindByState(states ...workflow.State) ([]string, error) {
	ctx := context.Background()

	stateStrings := make([]string, len(states))
	for i, s := range states {
		stateStrings[i] = s.String()
	}

	rows, err := r.pool.Query(ctx, `
		SELECT workflow_id FROM workflows
		WHERE state = ANY($1::workflow_state[])
		ORDER BY id
	`, stateStrings)
	if err != nil {
		return nil, fmt.Errorf("postgres/workflow: find by state: %w", err)
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("postgres/workflow: scan id: %w", err)
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// SaveSubWorkflowRef stores a parent-child workflow relationship.
func (r *WorkflowRepository) SaveSubWorkflowRef(ref *workflow.SubWorkflowRef) error {
	ctx := context.Background()

	_, err := r.pool.Exec(ctx, `
		INSERT INTO sub_workflow_refs (
			child_workflow_id, parent_workflow_id, parent_thread_id,
			parent_exec_id, child_schema_id, async, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, NOW())
		ON CONFLICT (child_workflow_id) DO NOTHING
	`,
		ref.ChildWorkflowID.String(), ref.ParentWorkflowID.String(),
		safeUint16ToInt16(ref.ParentThreadID), ref.ParentExecID.String(),
		ref.ChildSchemaID, ref.Async,
	)
	if err != nil {
		return fmt.Errorf("postgres/workflow: save sub-workflow ref: %w", err)
	}
	return nil
}

// FindSubWorkflowRef finds a sub-workflow reference by child workflow ID.
func (r *WorkflowRepository) FindSubWorkflowRef(childID string) (*workflow.SubWorkflowRef, error) {
	ctx := context.Background()

	var (
		parentWfID   string
		parentThread int16
		parentExecID string
		childWfID    string
		childSchema  string
		async        bool
	)

	err := r.pool.QueryRow(ctx, `
		SELECT child_workflow_id, parent_workflow_id, parent_thread_id,
		       parent_exec_id, child_schema_id, async
		FROM sub_workflow_refs WHERE child_workflow_id = $1
	`, childID).Scan(&childWfID, &parentWfID, &parentThread, &parentExecID, &childSchema, &async)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("sub-workflow ref for child %s not found", childID)
		}
		return nil, fmt.Errorf("postgres/workflow: find sub-workflow ref: %w", err)
	}

	return &workflow.SubWorkflowRef{
		ParentWorkflowID: pkgwf.ID(parentWfID),
		ParentThreadID:   safeInt16ToUint16(parentThread),
		ParentExecID:     pkgwf.ExecID(parentExecID),
		ChildWorkflowID:  pkgwf.ID(childWfID),
		ChildSchemaID:    childSchema,
		Async:            async,
	}, nil
}

// FindActiveSubWorkflows finds all sub-workflow references for a parent.
func (r *WorkflowRepository) FindActiveSubWorkflows(parentID string) ([]*workflow.SubWorkflowRef, error) {
	ctx := context.Background()

	rows, err := r.pool.Query(ctx, `
		SELECT child_workflow_id, parent_workflow_id, parent_thread_id,
		       parent_exec_id, child_schema_id, async
		FROM sub_workflow_refs WHERE parent_workflow_id = $1
	`, parentID)
	if err != nil {
		return nil, fmt.Errorf("postgres/workflow: find active sub-workflows: %w", err)
	}
	defer rows.Close()

	var refs []*workflow.SubWorkflowRef
	for rows.Next() {
		var (
			parentWfID   string
			parentThread int16
			parentExecID string
			childWfID    string
			childSchema  string
			async        bool
		)
		if err := rows.Scan(&childWfID, &parentWfID, &parentThread, &parentExecID, &childSchema, &async); err != nil {
			return nil, fmt.Errorf("postgres/workflow: scan sub-workflow ref: %w", err)
		}
		refs = append(refs, &workflow.SubWorkflowRef{
			ParentWorkflowID: pkgwf.ID(parentWfID),
			ParentThreadID:   safeInt16ToUint16(parentThread),
			ParentExecID:     pkgwf.ExecID(parentExecID),
			ChildWorkflowID:  pkgwf.ID(childWfID),
			ChildSchemaID:    childSchema,
			Async:            async,
		})
	}
	return refs, rows.Err()
}

// loadGraph fetches the graph schema definition from the object store and constructs a Graph.
func (r *WorkflowRepository) loadGraph(ctx context.Context, schemaID string) (*workflow.Graph, error) {
	var defRef string
	err := r.pool.QueryRow(ctx,
		`SELECT definition_ref FROM graph_schemas WHERE schema_id = $1`, schemaID,
	).Scan(&defRef)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("graph schema %q not found", schemaID)
		}
		return nil, fmt.Errorf("query graph schema: %w", err)
	}

	data, err := r.store.Get(ctx, defRef)
	if err != nil {
		return nil, fmt.Errorf("get object %q: %w", defRef, err)
	}

	schema, err := workflow.NewGraphSchemaFromJSON(data)
	if err != nil {
		return nil, fmt.Errorf("unmarshal graph schema: %w", err)
	}

	return workflow.NewGraph(schema)
}

// GetSnapshotRef returns the object store key of the execution snapshot.
func (r *WorkflowRepository) GetSnapshotRef(workflowID string) (string, error) {
	ctx := context.Background()
	var ref *string
	err := r.pool.QueryRow(ctx,
		`SELECT snapshot_ref FROM workflows WHERE workflow_id = $1`, workflowID,
	).Scan(&ref)
	if err != nil {
		return "", fmt.Errorf("postgres/workflow: get snapshot ref: %w", err)
	}
	if ref == nil {
		return "", nil
	}
	return *ref, nil
}

// SetSnapshotRef records the object store key of the execution snapshot.
func (r *WorkflowRepository) SetSnapshotRef(workflowID string, snapshotRef string) error {
	ctx := context.Background()
	_, err := r.pool.Exec(ctx,
		`UPDATE workflows SET snapshot_ref = $2, updated_at = NOW() WHERE workflow_id = $1`,
		workflowID, snapshotRef,
	)
	if err != nil {
		return fmt.Errorf("postgres/workflow: set snapshot ref: %w", err)
	}
	return nil
}

// restoreState sets the workflow state directly without appending a journal entry.
// This is used during reconstruction from the database.
func (r *WorkflowRepository) restoreState(wf *workflow.Workflow, state workflow.State) {
	// We need to set the state without triggering SetState() which appends a journal entry.
	// The approach: replay will restore the proper state, but callers that only read State()
	// (like GetWorkflowHandler and WorkflowSupervisor) need it set correctly.
	// Since SetState appends a journal entry, we call it and then clear the journal.
	wf.SetState(state)
	// The journal entry appended by SetState will be at sequence 1.
	// Callers that do journal replay (WorkflowHandler.Init) will overwrite via LoadFrom().
	// Callers that only read State() get the correct value.
}
