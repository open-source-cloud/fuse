package di

import (
	"context"

	"ergo.services/ergo/gen"
	"github.com/open-source-cloud/fuse/internal/services"
	"go.uber.org/fx"
)

func bindSchemaReplicationPublisher(lc fx.Lifecycle, node gen.Node, pub *services.ErgoSchemaUpsertPublisher) {
	lc.Append(fx.Hook{
		OnStart: func(_ context.Context) error {
			pub.BindNode(node)
			return nil
		},
	})
}

// ServicesModule provides the services for the application
var ServicesModule = fx.Module(
	"services",
	fx.Provide(
		services.NewErgoSchemaUpsertPublisher,
		fx.Annotate(
			func(p *services.ErgoSchemaUpsertPublisher) services.SchemaUpsertPublisher { return p },
			fx.As(new(services.SchemaUpsertPublisher)),
		),
		services.NewGraphService,
		services.NewPackageService,
	),
	fx.Invoke(bindSchemaReplicationPublisher),
)
