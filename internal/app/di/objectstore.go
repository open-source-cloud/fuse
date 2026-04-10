package di

import (
	"context"

	"github.com/open-source-cloud/fuse/internal/app/config"
	"github.com/open-source-cloud/fuse/pkg/objectstore"
	"github.com/rs/zerolog/log"
	"go.uber.org/fx"
)

// ObjectStoreModule provides the pluggable object store based on OBJECT_STORE_DRIVER.
var ObjectStoreModule = fx.Module(
	"objectstore",
	fx.Provide(provideObjectStore),
)

func provideObjectStore(cfg *config.Config) (objectstore.ObjectStore, error) {
	switch cfg.ObjectStore.Driver {
	case "filesystem":
		log.Info().Str("path", cfg.ObjectStore.FSBasePath).Msg("using filesystem object store")
		return objectstore.NewFilesystemObjectStore(cfg.ObjectStore.FSBasePath)

	case "s3":
		log.Info().
			Str("bucket", cfg.ObjectStore.S3Bucket).
			Str("endpoint", cfg.ObjectStore.S3Endpoint).
			Msg("using S3 object store")
		return objectstore.NewS3ObjectStore(context.Background(), objectstore.S3Config{
			Endpoint:  cfg.ObjectStore.S3Endpoint,
			Bucket:    cfg.ObjectStore.S3Bucket,
			Region:    cfg.ObjectStore.S3Region,
			AccessKey: cfg.ObjectStore.S3AccessKey,
			SecretKey: cfg.ObjectStore.S3SecretKey,
			UseSSL:    cfg.ObjectStore.S3UseSSL,
		})

	default: // "memory"
		log.Debug().Msg("using memory object store")
		return objectstore.NewMemoryObjectStore(), nil
	}
}
