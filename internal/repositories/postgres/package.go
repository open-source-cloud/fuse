package postgres

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/open-source-cloud/fuse/internal/repositories"
	"github.com/open-source-cloud/fuse/pkg/objectstore"
	"github.com/open-source-cloud/fuse/pkg/workflow"
)

// PackageRepository implements PackageRepository backed by PostgreSQL + ObjectStore.
type PackageRepository struct {
	repositories.PackageRepository
	pool  *pgxpool.Pool
	store objectstore.ObjectStore
}

// NewPackageRepository creates a new PostgreSQL-backed PackageRepository.
func NewPackageRepository(pool *pgxpool.Pool, store objectstore.ObjectStore) repositories.PackageRepository {
	return &PackageRepository{pool: pool, store: store}
}

const fusePkgPrefix = "fuse/pkg/"

// packageObjectKey maps business package_id to object-store key under the store root.
// Built-in packages use id fuse/pkg/<name> but are stored as packages/<name>/definition.json.
func packageObjectKey(id string) string {
	if strings.HasPrefix(id, fusePkgPrefix) {
		suffix := strings.TrimPrefix(id, fusePkgPrefix)
		if suffix != "" {
			return fmt.Sprintf("packages/%s/definition.json", suffix)
		}
	}
	return fmt.Sprintf("packages/%s/definition.json", id)
}

// FindByID retrieves a package by its business ID.
func (r *PackageRepository) FindByID(id string) (*workflow.Package, error) {
	ctx := context.Background()

	var defRef string
	err := r.pool.QueryRow(ctx,
		`SELECT definition_ref FROM packages WHERE package_id = $1`, id,
	).Scan(&defRef)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, repositories.ErrPackageNotFound
		}
		return nil, fmt.Errorf("postgres/package: find by id: %w", err)
	}

	data, err := r.store.Get(ctx, defRef)
	if err != nil {
		return nil, fmt.Errorf("postgres/package: get object %q: %w", defRef, err)
	}

	pkg, err := workflow.DecodePackage(data)
	if err != nil {
		return nil, fmt.Errorf("postgres/package: decode package: %w", err)
	}

	return pkg, nil
}

// FindAll retrieves all packages.
func (r *PackageRepository) FindAll() ([]*workflow.Package, error) {
	ctx := context.Background()

	rows, err := r.pool.Query(ctx, `SELECT package_id, definition_ref FROM packages ORDER BY id`)
	if err != nil {
		return nil, fmt.Errorf("postgres/package: find all: %w", err)
	}
	defer rows.Close()

	var pkgs []*workflow.Package
	for rows.Next() {
		var pkgID, defRef string
		if err := rows.Scan(&pkgID, &defRef); err != nil {
			return nil, fmt.Errorf("postgres/package: scan row: %w", err)
		}

		data, err := r.store.Get(ctx, defRef)
		if err != nil {
			return nil, fmt.Errorf("postgres/package: get object %q: %w", defRef, err)
		}

		pkg, err := workflow.DecodePackage(data)
		if err != nil {
			return nil, fmt.Errorf("postgres/package: decode package %q: %w", pkgID, err)
		}
		pkgs = append(pkgs, pkg)
	}
	return pkgs, rows.Err()
}

// Save persists a package definition to the object store and metadata to PostgreSQL.
func (r *PackageRepository) Save(pkg *workflow.Package) error {
	ctx := context.Background()

	objKey := packageObjectKey(pkg.ID)
	data, err := pkg.Encode()
	if err != nil {
		return fmt.Errorf("postgres/package: encode package: %w", err)
	}
	if err := r.store.Put(ctx, objKey, data); err != nil {
		return fmt.Errorf("postgres/package: put package: %w", err)
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("postgres/package: begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	_, err = tx.Exec(ctx, `
		INSERT INTO packages (package_id, definition_ref, created_at, updated_at)
		VALUES ($1, $2, NOW(), NOW())
		ON CONFLICT (package_id) DO UPDATE SET
			definition_ref = EXCLUDED.definition_ref,
			updated_at = NOW()
	`, pkg.ID, objKey)
	if err != nil {
		return fmt.Errorf("postgres/package: upsert: %w", err)
	}

	// Refresh tags
	_, err = tx.Exec(ctx, `DELETE FROM package_tags WHERE package_id = $1`, pkg.ID)
	if err != nil {
		return fmt.Errorf("postgres/package: delete tags: %w", err)
	}
	for k, v := range pkg.Tags {
		_, err = tx.Exec(ctx,
			`INSERT INTO package_tags (package_id, key, value) VALUES ($1, $2, $3)`,
			pkg.ID, k, v)
		if err != nil {
			return fmt.Errorf("postgres/package: insert tag: %w", err)
		}
	}

	// Refresh functions index
	_, err = tx.Exec(ctx, `DELETE FROM package_functions WHERE package_id = $1`, pkg.ID)
	if err != nil {
		return fmt.Errorf("postgres/package: delete functions: %w", err)
	}
	for _, fn := range pkg.Functions {
		_, err = tx.Exec(ctx,
			`INSERT INTO package_functions (package_id, function_id, transport) VALUES ($1, $2, $3)`,
			pkg.ID, fn.ID, fn.Metadata.Transport)
		if err != nil {
			return fmt.Errorf("postgres/package: insert function: %w", err)
		}
	}

	return tx.Commit(ctx)
}

// Delete removes a package by its business ID.
func (r *PackageRepository) Delete(id string) error {
	ctx := context.Background()

	// Cascading deletes handle package_tags and package_functions
	result, err := r.pool.Exec(ctx, `DELETE FROM packages WHERE package_id = $1`, id)
	if err != nil {
		return fmt.Errorf("postgres/package: delete: %w", err)
	}

	if result.RowsAffected() > 0 {
		_ = r.store.Delete(ctx, packageObjectKey(id))
	}
	return nil
}
