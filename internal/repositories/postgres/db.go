// Package postgres implements the repository interfaces backed by PostgreSQL
// and a pluggable ObjectStore for data payloads.
package postgres

import (
	"embed"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5" // pgx v5 driver for migrate
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/rs/zerolog/log"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// RunMigrations applies all pending database migrations using the embedded SQL files.
func RunMigrations(dsn string) error {
	source, err := iofs.New(migrationsFS, "migrations")
	if err != nil {
		return fmt.Errorf("postgres: open embedded migrations: %w", err)
	}

	m, err := migrate.NewWithSourceInstance("iofs", source, convertDSNForMigrate(dsn))
	if err != nil {
		return fmt.Errorf("postgres: create migrator: %w", err)
	}
	defer func() {
		srcErr, dbErr := m.Close()
		if srcErr != nil {
			log.Warn().Err(srcErr).Msg("close migration source")
		}
		if dbErr != nil {
			log.Warn().Err(dbErr).Msg("close migration database")
		}
	}()

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("postgres: run migrations: %w", err)
	}

	version, dirty, _ := m.Version()
	log.Info().Uint("version", version).Bool("dirty", dirty).Msg("database migrations applied")
	return nil
}

// convertDSNForMigrate converts a standard postgres:// DSN to the pgx5:// scheme
// required by golang-migrate's pgx v5 driver.
func convertDSNForMigrate(dsn string) string {
	if len(dsn) > 11 && dsn[:11] == "postgres://" {
		return "pgx5://" + dsn[11:]
	}
	if len(dsn) > 14 && dsn[:14] == "postgresql://" {
		return "pgx5://" + dsn[14:]
	}
	return dsn
}
