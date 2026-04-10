package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"ergo.services/ergo/gen"
	"github.com/open-source-cloud/fuse/internal/app/di"
	"github.com/open-source-cloud/fuse/internal/logging"
	"github.com/open-source-cloud/fuse/internal/services"
	"github.com/open-source-cloud/fuse/internal/workflow"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"go.uber.org/fx"
)

// noopSchemaUpsertPublisher satisfies GraphService for CLI-only upserts without ergo replication.
type noopSchemaUpsertPublisher struct{}

func (noopSchemaUpsertPublisher) PublishLocalUpsert(string, *workflow.GraphSchema) {}

func (noopSchemaUpsertPublisher) BindNode(gen.Node) {}

var (
	seedExamplesDir           string
	seedExamplesCI            bool
	seedExamplesContinueOnErr bool
)

func newSeedCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "seed",
		Short: "Load bundled example data into the configured graph store",
	}
	cmd.AddCommand(newSeedExamplesCommand())
	return cmd
}

func newSeedExamplesCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "examples",
		Short: "Upsert every JSON workflow under a directory as graph schemas",
		Long: "Reads *.json graph specs (same format as PUT /v1/schemas) and upserts them via GraphService. " +
			"Uses DB_DRIVER / DB_POSTGRES_DSN and object store settings like the server. " +
			"Does not start the HTTP API or actor runtime.",
		Args: cobra.NoArgs,
		RunE: func(*cobra.Command, []string) error {
			return runSeedExamplesApp()
		},
	}
	cmd.Flags().StringVar(&seedExamplesDir, "dir", "examples/workflows", "Directory containing example workflow JSON files")
	cmd.Flags().BoolVar(&seedExamplesCI, "ci", false, "Skip examples that need external services or async timer flows (matches scripts that set CI=true)")
	cmd.Flags().BoolVar(&seedExamplesContinueOnErr, "continue-on-error", false, "Log failures and keep seeding; still exits non-zero if any file failed")
	return cmd
}

func runSeedExamplesApp() error {
	absDir, err := filepath.Abs(seedExamplesDir)
	if err != nil {
		return fmt.Errorf("resolve seed dir: %w", err)
	}
	info, err := os.Stat(absDir)
	if err != nil {
		return fmt.Errorf("stat seed dir %q: %w", absDir, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("seed path is not a directory: %s", absDir)
	}

	app := fx.New(
		seedExamplesModules(),
		fx.Invoke(func(pkg services.PackageService, lc fx.Lifecycle, gs services.GraphService, sd fx.Shutdowner) error {
			if err := pkg.RegisterInternalPackages(); err != nil {
				return fmt.Errorf("register internal packages: %w", err)
			}
			lc.Append(fx.Hook{
				OnStart: func(ctx context.Context) error {
					if err := seedExampleWorkflows(ctx, gs, absDir, seedExamplesCI, seedExamplesContinueOnErr); err != nil {
						return err
					}
					go func() { _ = sd.Shutdown() }()
					return nil
				},
			})
			return nil
		}),
		fx.WithLogger(logging.NewFxLogger()),
	)

	app.Run()
	if err := app.Err(); err != nil {
		return err
	}
	return nil
}

func seedExamplesModules() fx.Option {
	return fx.Options(
		di.CommonModule,
		di.PackageModule,
		di.DatabaseModule,
		di.ObjectStoreModule,
		di.RepoModule,
		fx.Provide(
			func() services.SchemaUpsertPublisher {
				return noopSchemaUpsertPublisher{}
			},
			services.NewGraphService,
			services.NewPackageService,
		),
	)
}

func seedExampleWorkflows(ctx context.Context, gs services.GraphService, absDir string, ci, continueOnErr bool) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	entries, err := os.ReadDir(absDir)
	if err != nil {
		return fmt.Errorf("read seed dir: %w", err)
	}

	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.EqualFold(filepath.Ext(name), ".json") {
			continue
		}
		names = append(names, name)
	}
	slices.Sort(names)

	var failed bool
	for _, name := range names {
		path := filepath.Join(absDir, name)
		raw, err := os.ReadFile(path) //nolint:gosec // CLI reads user-selected example paths
		if err != nil {
			log.Error().Err(err).Str("file", path).Msg("failed to read workflow JSON")
			failed = true
			if !continueOnErr {
				return fmt.Errorf("read %s: %w", path, err)
			}
			continue
		}
		if ci && shouldSkipExampleWorkflow(name, raw) {
			log.Info().Str("file", name).Msg("skipping example (--ci)")
			continue
		}
		schema, err := workflow.NewGraphSchemaFromJSON(raw)
		if err != nil {
			log.Error().Err(err).Str("file", path).Msg("invalid workflow JSON")
			failed = true
			if !continueOnErr {
				return fmt.Errorf("parse %s: %w", path, err)
			}
			continue
		}
		graph, err := gs.Upsert(schema.ID, schema)
		if err != nil {
			log.Error().Err(err).Str("file", path).Str("schemaID", schema.ID).Msg("failed to upsert graph schema")
			failed = true
			if !continueOnErr {
				return fmt.Errorf("upsert %s (id=%s): %w", path, schema.ID, err)
			}
			continue
		}
		log.Info().Str("file", name).Str("schemaID", graph.ID()).Msg("seeded graph schema")
	}

	if failed {
		return fmt.Errorf("one or more example workflows failed to seed")
	}
	return nil
}

func shouldSkipExampleWorkflow(fileName string, raw []byte) bool {
	switch strings.ToLower(fileName) {
	case "github-request-example.json":
		return true
	default:
		return strings.Contains(string(raw), "fuse/pkg/logic/timer")
	}
}
