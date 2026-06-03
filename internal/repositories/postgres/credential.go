package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/open-source-cloud/fuse/internal/repositories"
	"github.com/open-source-cloud/fuse/pkg/workflow"
)

// CredentialRepository is a PostgreSQL-backed CredentialRepository (ADR-0031 Option B). It stores
// only credential metadata; field values live in the secrets table at cred/<id>/<field>.
type CredentialRepository struct {
	pool *pgxpool.Pool
}

// compile-time assertion.
var _ repositories.CredentialRepository = (*CredentialRepository)(nil)

// NewCredentialRepository creates a new PostgreSQL-backed CredentialRepository.
func NewCredentialRepository(pool *pgxpool.Pool) repositories.CredentialRepository {
	return &CredentialRepository{pool: pool}
}

// FindByID retrieves a credential by id.
func (r *CredentialRepository) FindByID(id string) (*workflow.Credential, error) {
	ctx := context.Background()

	var credType, description string
	var fields []string
	err := r.pool.QueryRow(ctx,
		`SELECT type, description, fields FROM credentials WHERE id = $1`, id,
	).Scan(&credType, &description, &fields)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, repositories.ErrCredentialNotFound
		}
		return nil, fmt.Errorf("postgres/credential: find by id: %w", err)
	}
	return workflow.NewCredential(id, credType, description, fields), nil
}

// FindAll retrieves all credentials sorted by id.
func (r *CredentialRepository) FindAll() ([]*workflow.Credential, error) {
	ctx := context.Background()

	rows, err := r.pool.Query(ctx, `SELECT id, type, description, fields FROM credentials ORDER BY id`)
	if err != nil {
		return nil, fmt.Errorf("postgres/credential: find all: %w", err)
	}
	defer rows.Close()

	creds := make([]*workflow.Credential, 0)
	for rows.Next() {
		var id, credType, description string
		var fields []string
		if err := rows.Scan(&id, &credType, &description, &fields); err != nil {
			return nil, fmt.Errorf("postgres/credential: scan row: %w", err)
		}
		creds = append(creds, workflow.NewCredential(id, credType, description, fields))
	}
	return creds, rows.Err()
}

// Save upserts a credential's metadata.
func (r *CredentialRepository) Save(cred *workflow.Credential) error {
	ctx := context.Background()

	_, err := r.pool.Exec(ctx, `
		INSERT INTO credentials (id, type, description, fields, created_at, updated_at)
		VALUES ($1, $2, $3, $4, NOW(), NOW())
		ON CONFLICT (id) DO UPDATE SET
			type = EXCLUDED.type,
			description = EXCLUDED.description,
			fields = EXCLUDED.fields,
			updated_at = NOW()
	`, cred.ID, cred.Type, cred.Description, cred.Fields)
	if err != nil {
		return fmt.Errorf("postgres/credential: upsert %q: %w", cred.ID, err)
	}
	return nil
}

// Delete removes a credential's metadata by id.
func (r *CredentialRepository) Delete(id string) error {
	ctx := context.Background()
	if _, err := r.pool.Exec(ctx, `DELETE FROM credentials WHERE id = $1`, id); err != nil {
		return fmt.Errorf("postgres/credential: delete %q: %w", id, err)
	}
	return nil
}
