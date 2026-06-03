package cli

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/open-source-cloud/fuse/internal/app/config"
	"github.com/open-source-cloud/fuse/internal/app/di"
	"github.com/open-source-cloud/fuse/internal/logging"
	"github.com/open-source-cloud/fuse/pkg/secrets"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"go.uber.org/fx"
)

// secretsEnvFlag scopes secret operations to an environment (defaults to FUSE_ENVIRONMENT).
var secretsEnvFlag string

func newSecretsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "secrets",
		Short: "Manage secrets in the configured store",
		Long: "Set, list, and delete secrets in the SECRETS_DRIVER backend. With driver=postgres " +
			"values are AES-256-GCM encrypted at rest. The memory driver is per-process (use " +
			"FUSE_SECRET_<NAME> env vars to seed the server instead).",
	}
	cmd.PersistentFlags().StringVar(&secretsEnvFlag, "env", "", "Environment scope (defaults to FUSE_ENVIRONMENT)")
	cmd.AddCommand(newSecretsSetCommand(), newSecretsListCommand(), newSecretsDeleteCommand())
	return cmd
}

func newSecretsSetCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "set <name> <value>",
		Short: "Set (or rotate) a secret",
		Args:  cobra.ExactArgs(2),
		RunE: func(_ *cobra.Command, args []string) error {
			name, value := args[0], args[1]
			return runSecretsApp(func(ctx context.Context, store secrets.ManagedSecretStore, env string) error {
				if err := store.Set(ctx, secrets.Scope{Environment: env}, name, value); err != nil {
					return err
				}
				log.Info().Str("environment", env).Str("name", name).Msg("secret set")
				return nil
			})
		},
	}
}

func newSecretsListCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List secret names in an environment",
		Args:  cobra.NoArgs,
		RunE: func(*cobra.Command, []string) error {
			return runSecretsApp(func(ctx context.Context, store secrets.ManagedSecretStore, env string) error {
				names, err := store.List(ctx, env)
				if err != nil {
					return err
				}
				// Hide credential-backed secrets (cred/<id>/<field>); they are managed via
				// `fuse credentials`, not exposed in the plain secret surface (ADR-0031).
				visible := make([]string, 0, len(names))
				for _, n := range names {
					if !strings.HasPrefix(n, "cred/") {
						visible = append(visible, n)
					}
				}
				fmt.Printf("secrets in environment %q:\n", env)
				if len(visible) == 0 {
					fmt.Println("  (none)")
				}
				for _, n := range visible {
					fmt.Printf("  - %s\n", n)
				}
				return nil
			})
		},
	}
}

func newSecretsDeleteCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <name>",
		Short: "Delete a secret",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			name := args[0]
			return runSecretsApp(func(ctx context.Context, store secrets.ManagedSecretStore, env string) error {
				if err := store.Delete(ctx, secrets.Scope{Environment: env}, name); err != nil {
					return err
				}
				log.Info().Str("environment", env).Str("name", name).Msg("secret deleted")
				return nil
			})
		},
	}
}

// runSecretsApp boots the minimal DI graph (config + database + secrets), runs the
// admin action against a managed store, and exits. It requires a ManagedSecretStore
// (the memory and postgres backends qualify; a read-only backend does not).
func runSecretsApp(action func(context.Context, secrets.ManagedSecretStore, string) error) error {
	var actionErr error
	app := fx.New(
		di.CommonModule,
		di.DatabaseModule,
		di.SecretsModule,
		fx.Invoke(func(lc fx.Lifecycle, cfg *config.Config, store secrets.SecretStore, sd fx.Shutdowner) {
			lc.Append(fx.Hook{
				OnStart: func(ctx context.Context) error {
					managed, ok := store.(secrets.ManagedSecretStore)
					if !ok {
						actionErr = errors.New("the configured SECRETS_DRIVER is read-only; set/list/delete require driver=memory or driver=postgres")
					} else {
						env := secretsEnvFlag
						if env == "" {
							env = cfg.Environment
						}
						actionErr = action(ctx, managed, env)
					}
					go func() { _ = sd.Shutdown() }()
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
