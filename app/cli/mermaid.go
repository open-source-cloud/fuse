package cli

import (
	"fmt"
	"github.com/open-source-cloud/fuse/internal/workflow"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"os"
	"path"
)

func newMermaidCommand() *cobra.Command {
	var mermaidCmd *cobra.Command
	mermaidCmd = &cobra.Command{
		Use:   "mermaid",
		Short: "Mermaid flowchart generator",
		Args:  cobra.NoArgs,
		Run: func(_ *cobra.Command, _ []string) {
			mermaidHandler()
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

func mermaidHandler() {
	// We are ok with reading the file here because we are in the CLI
	spec, err := os.ReadFile(mermaidSpecFile)
	if err != nil {
		log.Error().Err(err).Msg("Failed to read the workflow spec file")
		return
	}

	var graph *workflow.Graph
	specFileExt := path.Ext(mermaidSpecFile)
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

	fmt.Println(graph.MermaidFlowchart())
}
