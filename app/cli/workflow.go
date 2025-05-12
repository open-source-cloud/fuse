package cli

import (
	"github.com/open-source-cloud/fuse/app/di"
	"github.com/open-source-cloud/fuse/internal/repos"
	"github.com/open-source-cloud/fuse/internal/workflow"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"go.uber.org/fx"
	"os"
	"path"
)

func newWorkflowCommand() *cobra.Command {
	var workflowCmd *cobra.Command
	workflowCmd = &cobra.Command{
		Use:   "workflow",
		Short: "Workflow runner",
		Args:  cobra.NoArgs,
		Run: func(_ *cobra.Command, _ []string) {
			di.Run(fx.Options(
				di.AllModules,
				fx.Invoke(workflowRunner),
			))
		},
	}
	setupWorkflowFlags(workflowCmd)

	return workflowCmd
}

// workflowSpecFile is the path to the workflow config file
var workflowSpecFile string

// init initializes the workflow command flags
func setupWorkflowFlags(workflowCmd *cobra.Command) {
	workflowCmd.Flags().StringVarP(&workflowSpecFile, "config", "c", "", "Path to the workflow config file")
}

func workflowRunner(graphRepo repos.GraphRepo) {
	// We are ok with reading the file here because we are in the CLI
	spec, err := os.ReadFile(workflowSpecFile)
	if err != nil {
		log.Error().Err(err).Msg("Failed to read the workflow spec file")
		return
	}

	var graph *workflow.Graph
	specFileExt := path.Ext(workflowSpecFile)
	switch specFileExt {
	case ".json":
		graph, err = workflow.NewGraphFromJSON(spec)
		if err != nil {
			log.Error().Err(err).Msg("Failed to parse workflow JSON spec file")
			return
		}
	case ".yaml":
		graph, err = workflow.NewGraphFromYAML(spec)
		if err != nil {
			log.Error().Err(err).Msg("Failed to parse a workflow YAML spec file")
			return
		}

	default:
		log.Error().Msg("Unsupported workflow spec file type")
		return
	}

	err = graphRepo.Save(graph)
	if err != nil {
		log.Error().Err(err).Msg("Failed to save workflow graph")
		return
	}
	log.Info().Str("schemaID", graph.ID()).Msg("Workflow graph created")
}
