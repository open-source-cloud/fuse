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
	"github.com/open-source-cloud/fuse/pkg/objectstore"
)

// GraphRepository implements repositories.GraphRepository backed by PostgreSQL + ObjectStore.
type GraphRepository struct {
	repositories.GraphRepository
	pool  *pgxpool.Pool
	store objectstore.ObjectStore
}

// NewGraphRepository creates a new PostgreSQL-backed GraphRepository.
func NewGraphRepository(pool *pgxpool.Pool, store objectstore.ObjectStore) repositories.GraphRepository {
	return &GraphRepository{pool: pool, store: store}
}

func graphObjectKey(id string) string {
	return fmt.Sprintf("schemas/%s/definition.json", id)
}

// FindByID retrieves a graph by its schema business ID.
func (r *GraphRepository) FindByID(id string) (*workflow.Graph, error) {
	ctx := context.Background()

	var defRef string
	err := r.pool.QueryRow(ctx,
		`SELECT definition_ref FROM graph_schemas WHERE schema_id = $1`, id,
	).Scan(&defRef)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, repositories.ErrGraphNotFound
		}
		return nil, fmt.Errorf("postgres/graph: find by id: %w", err)
	}

	data, err := r.store.Get(ctx, defRef)
	if err != nil {
		return nil, fmt.Errorf("postgres/graph: get object %q: %w", defRef, err)
	}

	schema, err := workflow.NewGraphSchemaFromJSON(data)
	if err != nil {
		return nil, fmt.Errorf("postgres/graph: unmarshal schema: %w", err)
	}

	graph, err := workflow.NewGraph(schema)
	if err != nil {
		return nil, fmt.Errorf("postgres/graph: create graph: %w", err)
	}

	return graph, nil
}

// List returns schema_id and name for all rows in graph_schemas, ordered by schema_id.
func (r *GraphRepository) List() ([]repositories.GraphSchemaListItem, error) {
	ctx := context.Background()
	rows, err := r.pool.Query(ctx, `SELECT schema_id, name FROM graph_schemas ORDER BY schema_id`)
	if err != nil {
		return nil, fmt.Errorf("postgres/graph: list: %w", err)
	}
	defer rows.Close()

	out := make([]repositories.GraphSchemaListItem, 0)
	for rows.Next() {
		var id, name string
		if err := rows.Scan(&id, &name); err != nil {
			return nil, fmt.Errorf("postgres/graph: list scan: %w", err)
		}
		out = append(out, repositories.GraphSchemaListItem{SchemaID: id, Name: name})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("postgres/graph: list rows: %w", err)
	}
	return out, nil
}

// Save persists a graph's schema definition to the object store and metadata to PostgreSQL.
func (r *GraphRepository) Save(graph *workflow.Graph) error {
	ctx := context.Background()
	schema := graph.Schema()

	data, err := json.Marshal(&schema)
	if err != nil {
		return fmt.Errorf("postgres/graph: marshal schema: %w", err)
	}

	objKey := graphObjectKey(schema.ID)
	if err := r.store.Put(ctx, objKey, data); err != nil {
		return fmt.Errorf("postgres/graph: put object: %w", err)
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("postgres/graph: begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Extract timeout in nanoseconds
	var timeoutNs *int64
	if schema.Timeout != nil && schema.Timeout.Total.Duration() > 0 {
		ns := int64(schema.Timeout.Total.Duration() / time.Nanosecond)
		timeoutNs = &ns
	}

	// Upsert graph_schemas
	_, err = tx.Exec(ctx, `
		INSERT INTO graph_schemas (schema_id, name, timeout_total_ns, definition_ref, created_at, updated_at)
		VALUES ($1, $2, $3, $4, NOW(), NOW())
		ON CONFLICT (schema_id) DO UPDATE SET
			name = EXCLUDED.name,
			timeout_total_ns = EXCLUDED.timeout_total_ns,
			definition_ref = EXCLUDED.definition_ref,
			updated_at = NOW()
	`, schema.ID, schema.Name, timeoutNs, objKey)
	if err != nil {
		return fmt.Errorf("postgres/graph: upsert schema: %w", err)
	}

	// Refresh tags
	_, err = tx.Exec(ctx, `DELETE FROM graph_schema_tags WHERE schema_id = $1`, schema.ID)
	if err != nil {
		return fmt.Errorf("postgres/graph: delete tags: %w", err)
	}
	for k, v := range schema.Tags {
		_, err = tx.Exec(ctx,
			`INSERT INTO graph_schema_tags (schema_id, key, value) VALUES ($1, $2, $3)`,
			schema.ID, k, v)
		if err != nil {
			return fmt.Errorf("postgres/graph: insert tag: %w", err)
		}
	}

	// Refresh metadata
	_, err = tx.Exec(ctx, `DELETE FROM graph_schema_metadata WHERE schema_id = $1`, schema.ID)
	if err != nil {
		return fmt.Errorf("postgres/graph: delete metadata: %w", err)
	}
	for k, v := range schema.Metadata {
		_, err = tx.Exec(ctx,
			`INSERT INTO graph_schema_metadata (schema_id, key, value) VALUES ($1, $2, $3)`,
			schema.ID, k, v)
		if err != nil {
			return fmt.Errorf("postgres/graph: insert metadata: %w", err)
		}
	}

	// Refresh node index
	_, err = tx.Exec(ctx, `DELETE FROM graph_schema_nodes WHERE schema_id = $1`, schema.ID)
	if err != nil {
		return fmt.Errorf("postgres/graph: delete nodes: %w", err)
	}
	for _, node := range schema.Nodes {
		_, err = tx.Exec(ctx,
			`INSERT INTO graph_schema_nodes (schema_id, node_id, function_ref) VALUES ($1, $2, $3)`,
			schema.ID, node.ID, node.Function)
		if err != nil {
			return fmt.Errorf("postgres/graph: insert node: %w", err)
		}
	}

	return tx.Commit(ctx)
}
