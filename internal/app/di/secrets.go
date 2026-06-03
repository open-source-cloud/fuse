package di

import (
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/open-source-cloud/fuse/internal/app/config"
	"github.com/open-source-cloud/fuse/internal/repositories/postgres"
	"github.com/open-source-cloud/fuse/pkg/secrets"
	"github.com/rs/zerolog/log"
	"go.uber.org/fx"
)

// SecretsModule provides the SecretStore (selected by SECRETS_DRIVER) and the
// narrow Resolver the workflow engine uses to resolve secret references.
var SecretsModule = fx.Module(
	"secrets",
	fx.Provide(
		provideSecretStore,
		provideSecretResolver,
	),
)

type secretStoreParams struct {
	fx.In
	Config *config.Config
	Pool   *pgxpool.Pool `optional:"true"`
}

// provideSecretStore selects the secret backend by SECRETS_DRIVER, mirroring the
// object-store driver pattern. Postgres encrypts at rest (AES-256-GCM); it falls
// back to memory (with a warning) if no DB pool is available.
func provideSecretStore(p secretStoreParams) (secrets.SecretStore, error) {
	switch p.Config.Secrets.Driver {
	case "postgres":
		if p.Pool == nil {
			log.Warn().Msg("SECRETS_DRIVER=postgres but no database pool; falling back to the memory secret store")
			return secrets.NewMemorySecretStoreFromEnv(p.Config.Environment), nil
		}
		cipher, err := secrets.NewCipherFromBase64Key(p.Config.Secrets.EncryptionKey)
		if err != nil {
			return nil, fmt.Errorf("%w (set SECRETS_ENCRYPTION_KEY to a base64-encoded 32-byte key)", err)
		}
		log.Info().Str("environment", p.Config.Environment).Msg("using postgres (encrypted) secret store")
		return postgres.NewSecretStore(p.Pool, cipher), nil
	default:
		log.Debug().Str("environment", p.Config.Environment).Msg("using memory secret store")
		return secrets.NewMemorySecretStoreFromEnv(p.Config.Environment), nil
	}
}

// provideSecretResolver binds the store + configured environment into a Resolver.
func provideSecretResolver(store secrets.SecretStore, cfg *config.Config) secrets.Resolver {
	return secrets.NewResolver(store, cfg.Environment)
}
