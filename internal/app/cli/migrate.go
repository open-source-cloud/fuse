package cli

import (
	"fmt"

	"github.com/open-source-cloud/fuse/internal/app/config"
	"github.com/open-source-cloud/fuse/internal/repositories/postgres"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

func newMigrateCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "migrate",
		Short: "Apply PostgreSQL schema migrations",
		Long: "Connects with DB_POSTGRES_DSN and applies all pending embedded SQL migrations (golang-migrate). " +
			"Does not start the HTTP server or actor runtime.",
		Args: cobra.NoArgs,
		RunE: func(*cobra.Command, []string) error {
			cfg := config.Instance()
			if cfg.Database.PostgresDSN == "" {
				return fmt.Errorf("DB_POSTGRES_DSN must be set")
			}
			log.Info().Msg("running database migrations")
			if err := postgres.RunMigrations(cfg.Database.PostgresDSN); err != nil {
				return err
			}
			log.Info().Msg("migrations complete")
			return nil
		},
	}
}
