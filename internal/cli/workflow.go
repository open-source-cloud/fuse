package cli

import (
	"os"

	"github.com/open-source-cloud/fuse/internal/graph/memory"
	"github.com/open-source-cloud/fuse/internal/providers"
	"github.com/open-source-cloud/fuse/internal/workflow"
	"github.com/open-source-cloud/fuse/pkg/uuid"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

// workflowConfigYamlPath is the path to the workflow config file
var workflowConfigYamlPath string

var workflowCmd = &cobra.Command{
	Use:   "workflow",
	Short: "Workflow runner",
	Args:  cobra.NoArgs,
	RunE:  workflowRunner,
}

// init initializes the workflow command flags
func init() {
	workflowCmd.Flags().StringVarP(&workflowConfigYamlPath, "config", "c", "", "Path to the workflow config file")
}

// Workflow runner
// This command reads the workflow config file, creates a schema, and starts the engine
// It then adds the schema to the engine and sends a start message to the engine
// It then waits for the engine to finish and returns the result
func workflowRunner(_ *cobra.Command, _ []string) error {
	engine := workflow.NewEngine()

	providerRegistry := providers.NewRegistry()

	// nolint:gosec
	// We are ok with reading the file here because we are in the CLI
	yamlSpec, err := os.ReadFile(workflowConfigYamlPath)
	if err != nil {
		return err
	}

	schemaDef, graph, err := memory.CreateSchemaFromYaml(yamlSpec, providerRegistry)
	if err != nil {
		return err
	}

	log.Info().Msgf("schema created: %s", schemaDef.Name)

	engine.Start()

	schema := workflow.LoadSchema(uuid.V7(), graph)
	engine.AddSchema(schema)
	engine.SendMessage(workflow.NewEngineMessage(workflow.EngineMessageStartWorkflow, schema.ID()))

	quitOnCtrlC()
	engine.Stop()

	return nil
}
