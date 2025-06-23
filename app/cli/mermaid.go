package cli

import (
	"fmt"
	"os"
	"path"

	"github.com/open-source-cloud/fuse/app/di"
	"github.com/open-source-cloud/fuse/internal/services"
	"github.com/open-source-cloud/fuse/internal/workflow"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"go.uber.org/fx"
)

func newMermaidCommand() *cobra.Command {
	mermaidCmd := &cobra.Command{
		Use:   "mermaid",
		Short: "Mermaid flowchart generator",
		Args:  cobra.NoArgs,
		Run: func(_ *cobra.Command, _ []string) {
			di.Run(fx.Options(
				di.AllModules,
				fx.Invoke(mermaidHandler),
			))
		},
	}
	setupMermaidFlags(mermaidCmd)

	return mermaidCmd
}

var mermaidSpecFile string

// init initializes the workflow command flags
func setupMermaidFlags(mermaidCmd *cobra.Command) {
	mermaidCmd.Flags().StringVarP(&mermaidSpecFile, "config", "c", "", "Path to the workflow config file")
}

// mermaidHandler is the handler for the mermaid command that prints the mermaid flowchart to the console
func mermaidHandler(graphService services.GraphService) {
	// We are ok with reading the file here because we are in the CLI
	spec, err := os.ReadFile(mermaidSpecFile) //nolint:gosec
	if err != nil {
		log.Error().Err(err).Msg("Failed to read the workflow spec file")
		return
	}

	var graph *workflow.Graph
	specFileExt := path.Ext(mermaidSpecFile)
	switch specFileExt {
	case ".json":
		schema, err := workflow.NewGraphSchemaFromJSON(spec)
		if err != nil {
			log.Error().Err(err).Msg("Failed to parse workflow JSON spec file")
			return
		}
		graph, err = graphService.Upsert(schema.ID, schema)
		if err != nil {
			log.Error().Err(err).Msg("Failed to upsert workflow graph")
			return
		}
	default:
		log.Error().Msg("Unsupported workflow spec file type")
		return
	}

	fmt.Println(graph.MermaidFlowchart())
}
