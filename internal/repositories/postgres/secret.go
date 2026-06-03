package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/open-source-cloud/fuse/pkg/secrets"
)

// SecretStore is an encrypted-at-rest secrets.ManagedSecretStore backed by
// PostgreSQL. Values are AES-256-GCM encrypted on write and decrypted on read;
// the database column only ever holds ciphertext.
type SecretStore struct {
	pool   *pgxpool.Pool
	cipher *secrets.Cipher
}

// compile-time assertion.
var _ secrets.ManagedSecretStore = (*SecretStore)(nil)

// NewSecretStore creates a PostgreSQL-backed encrypted secret store.
func NewSecretStore(pool *pgxpool.Pool, cipher *secrets.Cipher) *SecretStore {
	return &SecretStore{pool: pool, cipher: cipher}
}

// Resolve decrypts and returns the secret for (environment, name).
func (r *SecretStore) Resolve(ctx context.Context, scope secrets.Scope, name string) (secrets.SecretValue, error) {
	var enc []byte
	err := r.pool.QueryRow(ctx,
		`SELECT encrypted_value FROM secrets WHERE environment = $1 AND name = $2`,
		scope.Environment, name,
	).Scan(&enc)
	if err != nil {
		if err == pgx.ErrNoRows {
			return secrets.SecretValue{}, fmt.Errorf("%w: %q (environment %q)", secrets.ErrSecretNotFound, name, scope.Environment)
		}
		return secrets.SecretValue{}, fmt.Errorf("postgres/secrets: query: %w", err)
	}
	plaintext, err := r.cipher.Decrypt(enc)
	if err != nil {
		return secrets.SecretValue{}, fmt.Errorf("postgres/secrets: decrypt %q: %w", name, err)
	}
	return secrets.NewSecretValue(string(plaintext)), nil
}

// Set encrypts and upserts a secret value.
func (r *SecretStore) Set(ctx context.Context, scope secrets.Scope, name, value string) error {
	enc, err := r.cipher.Encrypt([]byte(value))
	if err != nil {
		return fmt.Errorf("postgres/secrets: encrypt %q: %w", name, err)
	}
	_, err = r.pool.Exec(ctx,
		`INSERT INTO secrets (environment, name, encrypted_value, source, created_at, updated_at)
		 VALUES ($1, $2, $3, 'manual', NOW(), NOW())
		 ON CONFLICT (environment, name)
		 DO UPDATE SET encrypted_value = EXCLUDED.encrypted_value, updated_at = NOW()`,
		scope.Environment, name, enc,
	)
	if err != nil {
		return fmt.Errorf("postgres/secrets: upsert %q: %w", name, err)
	}
	return nil
}

// List returns the secret names in an environment, sorted.
func (r *SecretStore) List(ctx context.Context, environment string) ([]string, error) {
	rows, err := r.pool.Query(ctx, `SELECT name FROM secrets WHERE environment = $1 ORDER BY name`, environment)
	if err != nil {
		return nil, fmt.Errorf("postgres/secrets: list: %w", err)
	}
	defer rows.Close()

	names := make([]string, 0)
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("postgres/secrets: scan: %w", err)
		}
		names = append(names, name)
	}
	return names, rows.Err()
}

// Delete removes a secret.
func (r *SecretStore) Delete(ctx context.Context, scope secrets.Scope, name string) error {
	if _, err := r.pool.Exec(ctx, `DELETE FROM secrets WHERE environment = $1 AND name = $2`, scope.Environment, name); err != nil {
		return fmt.Errorf("postgres/secrets: delete %q: %w", name, err)
	}
	return nil
}
