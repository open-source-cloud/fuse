package cli

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/open-source-cloud/fuse/internal/app/config"
	"github.com/open-source-cloud/fuse/internal/app/di"
	"github.com/open-source-cloud/fuse/internal/logging"
	"github.com/open-source-cloud/fuse/internal/repositories"
	"github.com/open-source-cloud/fuse/internal/repositories/postgres"
	"github.com/open-source-cloud/fuse/internal/services"
	"github.com/open-source-cloud/fuse/pkg/secrets"
	"github.com/open-source-cloud/fuse/pkg/workflow"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"go.uber.org/fx"
)

// credentialsEnvFlag scopes credential value operations to an environment (defaults to FUSE_ENVIRONMENT).
var credentialsEnvFlag string

// credentialsTypeFlag sets the credential type on `credentials set` (defaults to "custom").
var credentialsTypeFlag string

func newCredentialsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "credentials",
		Short: "Manage credentials (typed groups of secret values) in the configured store",
		Long: "Set, list, and delete credentials. A credential's field values are stored in the " +
			"SECRETS_DRIVER backend at cred/<id>/<field>, per environment (ADR-0031). Requires " +
			"driver=memory or driver=postgres for writes.",
	}
	cmd.PersistentFlags().StringVar(&credentialsEnvFlag, "env", "", "Environment scope (defaults to FUSE_ENVIRONMENT)")
	cmd.AddCommand(newCredentialsSetCommand(), newCredentialsListCommand(), newCredentialsDeleteCommand())
	return cmd
}

func newCredentialsSetCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set <id> <field> <value>",
		Short: "Set (or rotate) a credential field value",
		Args:  cobra.ExactArgs(3),
		RunE: func(_ *cobra.Command, args []string) error {
			id, field, value := args[0], args[1], args[2]
			return runCredentialsApp(func(_ context.Context, svc services.CredentialService, env string) error {
				cred := workflow.NewCredential(id, credentialsTypeFlag, "", nil)
				if _, err := svc.Save(cred, map[string]string{field: value}, env); err != nil {
					return err
				}
				log.Info().Str("environment", env).Str("credential", id).Str("field", field).Msg("credential field set")
				return nil
			})
		},
	}
	cmd.Flags().StringVar(&credentialsTypeFlag, "type", "custom", "Credential type (free-form, e.g. openai)")
	return cmd
}

func newCredentialsListCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List credentials (ids, type, and field names; never values)",
		Args:  cobra.NoArgs,
		RunE: func(*cobra.Command, []string) error {
			return runCredentialsApp(func(_ context.Context, svc services.CredentialService, _ string) error {
				creds, err := svc.FindAll()
				if err != nil {
					return err
				}
				if len(creds) == 0 {
					fmt.Println("(no credentials)")
					return nil
				}
				for _, c := range creds {
					fmt.Printf("  - %s (type=%s) fields=%v\n", c.ID, c.Type, c.Fields)
				}
				return nil
			})
		},
	}
}

func newCredentialsDeleteCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete a credential and its field values in the environment",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			id := args[0]
			return runCredentialsApp(func(_ context.Context, svc services.CredentialService, env string) error {
				if err := svc.Delete(id, env); err != nil {
					return err
				}
				log.Info().Str("environment", env).Str("credential", id).Msg("credential deleted")
				return nil
			})
		},
	}
}

// credentialsAppParams are the dependencies the credentials CLI action needs. Pool is optional so
// the command works under the memory driver (no database).
type credentialsAppParams struct {
	fx.In
	LC    fx.Lifecycle
	Cfg   *config.Config
	Store secrets.SecretStore
	Pool  *pgxpool.Pool `optional:"true"`
	SD    fx.Shutdowner
}

// runCredentialsApp boots the minimal DI graph (config + database + secrets), builds a
// CredentialService over the selected secret backend, runs the admin action, and exits.
func runCredentialsApp(action func(context.Context, services.CredentialService, string) error) error {
	var actionErr error
	app := fx.New(
		di.CommonModule,
		di.DatabaseModule,
		di.SecretsModule,
		fx.Invoke(func(p credentialsAppParams) {
			p.LC.Append(fx.Hook{
				OnStart: func(ctx context.Context) error {
					repo := credentialRepoFor(p.Cfg, p.Pool)
					svc := services.NewCredentialService(repo, p.Store)
					env := credentialsEnvFlag
					if env == "" {
						env = p.Cfg.Environment
					}
					actionErr = action(ctx, svc, env)
					go func() { _ = p.SD.Shutdown() }()
					return nil
				},
			})
		}),
		fx.WithLogger(logging.NewFxLogger()),
	)
	app.Run()
	if err := app.Err(); err != nil {
		return err
	}
	return actionErr
}

// credentialRepoFor selects the credential metadata repository matching the database driver.
func credentialRepoFor(cfg *config.Config, pool *pgxpool.Pool) repositories.CredentialRepository {
	if cfg.Database.Driver == config.DBDriverPostgres && pool != nil {
		return postgres.NewCredentialRepository(pool)
	}
	return repositories.NewMemoryCredentialRepository()
}
