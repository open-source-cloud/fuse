package cli

import (
	"bytes"
	"encoding/json"
	"github.com/open-source-cloud/fuse/app/di"
	"github.com/open-source-cloud/fuse/internal/repos"
	"github.com/open-source-cloud/fuse/internal/workflow"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"go.uber.org/fx"
	"io"
	"net/http"
	"os"
	"path"
)

func newWorkflowCommand() *cobra.Command {
	workflowCmd := &cobra.Command{
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

func workflowRunner(graphFactory *workflow.GraphFactory, graphRepo repos.GraphRepo) {
	// We are ok with reading the file here because we are in the CLI
	spec, err := os.ReadFile(workflowSpecFile) //nolint:gosec
	if err != nil {
		log.Error().Err(err).Msg("Failed to read the workflow spec file")
		return
	}

	var graph *workflow.Graph
	specFileExt := path.Ext(workflowSpecFile)
	switch specFileExt {
	case ".json":
		graph, err = graphFactory.NewGraphFromJSON(spec)
		if err != nil {
			log.Error().Err(err).Msg("Failed to parse workflow JSON spec file")
			return
		}
	case ".yaml":
		graph, err = graphFactory.NewGraphFromYAML(spec)
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

	// make http request to run the supplied workflow once
	payload := map[string]string{"schemaId": graph.ID()}
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		log.Error().Err(err).Msg("Failed to trigger workflow: failed marshaling payload")
		return
	}

	resp, err := http.Post("http://localhost:9090/v1/workflows/trigger", "application/json", bytes.NewBuffer(jsonPayload))
	if err != nil {
		log.Error().Err(err).Msg("Failed to trigger workflow: failed making http request")
		return
	}
	defer func(Body io.ReadCloser) {
		err = Body.Close()
		if err != nil {
			log.Error().Err(err).Msg("Failed to close response body")
		}
	}(resp.Body)
}
