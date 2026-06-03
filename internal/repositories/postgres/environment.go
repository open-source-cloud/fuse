package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/open-source-cloud/fuse/internal/repositories"
	"github.com/open-source-cloud/fuse/pkg/workflow"
)

// EnvironmentRepository is a PostgreSQL-backed EnvironmentRepository (ADR-0031). Environments
// are small metadata, so unlike packages/workflows they are stored entirely in the row (no
// object store).
type EnvironmentRepository struct {
	pool *pgxpool.Pool
}

// compile-time assertion.
var _ repositories.EnvironmentRepository = (*EnvironmentRepository)(nil)

// NewEnvironmentRepository creates a new PostgreSQL-backed EnvironmentRepository.
func NewEnvironmentRepository(pool *pgxpool.Pool) repositories.EnvironmentRepository {
	return &EnvironmentRepository{pool: pool}
}

// FindByID retrieves an environment by name.
func (r *EnvironmentRepository) FindByID(name string) (*workflow.Environment, error) {
	ctx := context.Background()

	var description string
	err := r.pool.QueryRow(ctx,
		`SELECT description FROM environments WHERE name = $1`, name,
	).Scan(&description)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, repositories.ErrEnvironmentNotFound
		}
		return nil, fmt.Errorf("postgres/environment: find by id: %w", err)
	}
	return workflow.NewEnvironment(name, description), nil
}

// FindAll retrieves all environments sorted by name.
func (r *EnvironmentRepository) FindAll() ([]*workflow.Environment, error) {
	ctx := context.Background()

	rows, err := r.pool.Query(ctx, `SELECT name, description FROM environments ORDER BY name`)
	if err != nil {
		return nil, fmt.Errorf("postgres/environment: find all: %w", err)
	}
	defer rows.Close()

	envs := make([]*workflow.Environment, 0)
	for rows.Next() {
		var name, description string
		if err := rows.Scan(&name, &description); err != nil {
			return nil, fmt.Errorf("postgres/environment: scan row: %w", err)
		}
		envs = append(envs, workflow.NewEnvironment(name, description))
	}
	return envs, rows.Err()
}

// Save upserts an environment.
func (r *EnvironmentRepository) Save(env *workflow.Environment) error {
	ctx := context.Background()

	_, err := r.pool.Exec(ctx, `
		INSERT INTO environments (name, description, created_at, updated_at)
		VALUES ($1, $2, NOW(), NOW())
		ON CONFLICT (name) DO UPDATE SET
			description = EXCLUDED.description,
			updated_at = NOW()
	`, env.Name, env.Description)
	if err != nil {
		return fmt.Errorf("postgres/environment: upsert %q: %w", env.Name, err)
	}
	return nil
}

// Delete removes an environment by name.
func (r *EnvironmentRepository) Delete(name string) error {
	ctx := context.Background()
	if _, err := r.pool.Exec(ctx, `DELETE FROM environments WHERE name = $1`, name); err != nil {
		return fmt.Errorf("postgres/environment: delete %q: %w", name, err)
	}
	return nil
}
