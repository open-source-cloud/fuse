package cli

import (
	"github.com/open-source-cloud/fuse/internal/workflow"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"os"
	"path"
)

// workflowSpecFile is the path to the workflow config file
var workflowSpecFile string

var workflowCmd = &cobra.Command{
	Use:   "workflow",
	Short: "Workflow runner",
	Args:  cobra.NoArgs,
	RunE:  workflowRunner,
}

// init initializes the workflow command flags
func init() {
	workflowCmd.Flags().StringVarP(&workflowSpecFile, "config", "c", "", "Path to the workflow config file")
}

// Workflow runner
// This command reads the workflow config file, creates a schema, and starts the engine.
// Then adds the schema to the engine and sends a start message to the engine.
// Then waits for the engine to finish and returns the result.
func workflowRunner(_ *cobra.Command, _ []string) error {
	log.Info().Msg("workflow cli")

	// We are ok with reading the file here because we are in the CLI
	spec, err := os.ReadFile(workflowSpecFile)
	if err != nil {
		log.Error().Err(err).Msg("Failed to read workflow spec file")
		os.Exit(1)
	}

	var graph *workflow.Graph
	specFileExt := path.Ext(workflowSpecFile)
	switch specFileExt {
	case ".json":
		graph, err = workflow.NewGraphFromJSON(spec)
		if err != nil {
			log.Error().Err(err).Msg("Failed to parse workflow JSON spec file")
			os.Exit(1)
		}
	case ".yaml":
		graph, err = workflow.NewGraphFromYAML(spec)
		if err != nil {
			log.Error().Err(err).Msg("Failed to parse workflow YAML spec file")
			os.Exit(1)
		}

	default:
		log.Error().Msg("Unsupported workflow spec file type")
		os.Exit(1)
	}

	err = cli.graphRepo.Save(graph)
	if err != nil {
		log.Error().Err(err).Msg("Failed to save workflow graph")
		os.Exit(1)
	}

	return nil
}
