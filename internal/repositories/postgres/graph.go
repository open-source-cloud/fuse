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

func graphVersionObjectKey(id string, version int) string {
	return fmt.Sprintf("schemas/%s/v%d/definition.json", id, version)
}

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

// FindByIDAndVersion retrieves the graph for a specific schema version.
func (r *GraphRepository) FindByIDAndVersion(id string, version int) (*workflow.Graph, error) {
	ctx := context.Background()

	var defRef string
	err := r.pool.QueryRow(ctx,
		`SELECT definition_ref FROM graph_schema_versions WHERE schema_id = $1 AND version = $2`,
		id, version,
	).Scan(&defRef)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, repositories.ErrSchemaVersionNotFound
		}
		return nil, fmt.Errorf("postgres/graph: find by id and version: %w", err)
	}

	data, err := r.store.Get(ctx, defRef)
	if err != nil {
		return nil, fmt.Errorf("postgres/graph: get version object %q: %w", defRef, err)
	}

	schema, err := workflow.NewGraphSchemaFromJSON(data)
	if err != nil {
		return nil, fmt.Errorf("postgres/graph: unmarshal version schema: %w", err)
	}

	graph, err := workflow.NewGraph(schema)
	if err != nil {
		return nil, fmt.Errorf("postgres/graph: create version graph: %w", err)
	}

	return graph, nil
}

// SaveVersion persists a new SchemaVersion. If sv.IsActive is true the active-version
// pointer on graph_schemas is updated.
func (r *GraphRepository) SaveVersion(sv *workflow.SchemaVersion) error {
	ctx := context.Background()

	data, err := json.Marshal(&sv.Schema)
	if err != nil {
		return fmt.Errorf("postgres/graph: marshal version schema: %w", err)
	}

	objKey := graphVersionObjectKey(sv.SchemaID, sv.Version)
	if err := r.store.Put(ctx, objKey, data); err != nil {
		return fmt.Errorf("postgres/graph: put version object: %w", err)
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("postgres/graph: begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	_, err = tx.Exec(ctx, `
		INSERT INTO graph_schema_versions (schema_id, version, definition_ref, created_by, comment, is_active, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (schema_id, version) DO UPDATE SET
			definition_ref = EXCLUDED.definition_ref,
			created_by     = EXCLUDED.created_by,
			comment        = EXCLUDED.comment,
			is_active      = EXCLUDED.is_active
	`, sv.SchemaID, sv.Version, objKey, sv.CreatedBy, sv.Comment, sv.IsActive, sv.CreatedAt)
	if err != nil {
		return fmt.Errorf("postgres/graph: insert version: %w", err)
	}

	if sv.IsActive {
		_, err = tx.Exec(ctx, `
			UPDATE graph_schema_versions SET is_active = FALSE
			WHERE schema_id = $1 AND version != $2
		`, sv.SchemaID, sv.Version)
		if err != nil {
			return fmt.Errorf("postgres/graph: deactivate old versions: %w", err)
		}

		_, err = tx.Exec(ctx, `
			UPDATE graph_schemas SET active_version = $1 WHERE schema_id = $2
		`, sv.Version, sv.SchemaID)
		if err != nil {
			return fmt.Errorf("postgres/graph: update active version: %w", err)
		}
	}

	return tx.Commit(ctx)
}

// ListVersions returns all recorded versions for a schema ordered by version ascending.
func (r *GraphRepository) ListVersions(schemaID string) ([]workflow.SchemaVersion, error) {
	ctx := context.Background()

	// Verify schema exists
	var exists bool
	err := r.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM graph_schemas WHERE schema_id = $1)`, schemaID).Scan(&exists)
	if err != nil {
		return nil, fmt.Errorf("postgres/graph: check schema exists: %w", err)
	}
	if !exists {
		return nil, repositories.ErrGraphNotFound
	}

	rows, err := r.pool.Query(ctx, `
		SELECT version, definition_ref, created_by, comment, is_active, created_at
		FROM graph_schema_versions
		WHERE schema_id = $1
		ORDER BY version ASC
	`, schemaID)
	if err != nil {
		return nil, fmt.Errorf("postgres/graph: list versions: %w", err)
	}
	defer rows.Close()

	var out []workflow.SchemaVersion
	for rows.Next() {
		var (
			version   int
			defRef    string
			createdBy *string
			comment   *string
			isActive  bool
			createdAt time.Time
		)
		if err := rows.Scan(&version, &defRef, &createdBy, &comment, &isActive, &createdAt); err != nil {
			return nil, fmt.Errorf("postgres/graph: scan version row: %w", err)
		}

		data, err := r.store.Get(ctx, defRef)
		if err != nil {
			return nil, fmt.Errorf("postgres/graph: get version object %q: %w", defRef, err)
		}
		schema, err := workflow.NewGraphSchemaFromJSON(data)
		if err != nil {
			return nil, fmt.Errorf("postgres/graph: unmarshal version schema: %w", err)
		}

		sv := workflow.SchemaVersion{
			SchemaID:  schemaID,
			Version:   version,
			Schema:    *schema,
			IsActive:  isActive,
			CreatedAt: createdAt,
		}
		if createdBy != nil {
			sv.CreatedBy = *createdBy
		}
		if comment != nil {
			sv.Comment = *comment
		}
		out = append(out, sv)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("postgres/graph: list versions rows: %w", err)
	}
	if out == nil {
		out = []workflow.SchemaVersion{}
	}
	return out, nil
}

// SetActiveVersion updates the active version pointer and updates definition_ref on graph_schemas
// so FindByID returns the correct schema.
func (r *GraphRepository) SetActiveVersion(schemaID string, version int) error {
	ctx := context.Background()

	var defRef string
	err := r.pool.QueryRow(ctx,
		`SELECT definition_ref FROM graph_schema_versions WHERE schema_id = $1 AND version = $2`,
		schemaID, version,
	).Scan(&defRef)
	if err != nil {
		if err == pgx.ErrNoRows {
			return repositories.ErrSchemaVersionNotFound
		}
		return fmt.Errorf("postgres/graph: set active version lookup: %w", err)
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("postgres/graph: begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	_, err = tx.Exec(ctx, `
		UPDATE graph_schema_versions SET is_active = (version = $2) WHERE schema_id = $1
	`, schemaID, version)
	if err != nil {
		return fmt.Errorf("postgres/graph: update version is_active: %w", err)
	}

	_, err = tx.Exec(ctx, `
		UPDATE graph_schemas SET active_version = $1, definition_ref = $2 WHERE schema_id = $3
	`, version, defRef, schemaID)
	if err != nil {
		return fmt.Errorf("postgres/graph: update active version on schema: %w", err)
	}

	return tx.Commit(ctx)
}

// GetVersionHistory returns aggregate version metadata for a schema.
func (r *GraphRepository) GetVersionHistory(schemaID string) (*workflow.SchemaVersionHistory, error) {
	ctx := context.Background()

	var activeVersion int
	var total int
	var latestVersion int

	err := r.pool.QueryRow(ctx, `
		SELECT
			gs.active_version,
			COUNT(gsv.version)::INT,
			COALESCE(MAX(gsv.version), 0)::INT
		FROM graph_schemas gs
		LEFT JOIN graph_schema_versions gsv ON gsv.schema_id = gs.schema_id
		WHERE gs.schema_id = $1
		GROUP BY gs.active_version
	`, schemaID).Scan(&activeVersion, &total, &latestVersion)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, repositories.ErrGraphNotFound
		}
		return nil, fmt.Errorf("postgres/graph: get version history: %w", err)
	}

	return &workflow.SchemaVersionHistory{
		SchemaID:      schemaID,
		ActiveVersion: activeVersion,
		LatestVersion: latestVersion,
		TotalVersions: total,
	}, nil
}
