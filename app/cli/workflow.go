package cli

import (
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
// This command reads the workflow config file, creates a schema, and starts the engine.
// Then adds the schema to the engine and sends a start message to the engine.
// Then waits for the engine to finish and returns the result.
func workflowRunner(_ *cobra.Command, _ []string) error {
	log.Info().Msg("workflow cli")
	//cfg, err := config.NewConfig()
	//if err != nil {
	//	return err
	//}
	//
	//if err = cfg.Validate(); err != nil {
	//	return err
	//}
	//
	//cfg.Server.Run = true

	//appSupervisor := app.NewSupervisor(cfg)
	//appSupervisor.Start()
	//
	//providerRegistry := packages.NewRegistry()
	//// nolint:gosec
	//// We are ok with reading the file here because we are in the CLI
	//yamlSpec, err := os.ReadFile(workflowConfigYamlPath)
	//if err != nil {
	//	return err
	//}
	//
	//schemaDef, g, err := graph.CreateSchemaFromYaml(yamlSpec, providerRegistry)
	//if err != nil {
	//	return err
	//}
	//
	//log.Info().Msgf("schema created: %s", schemaDef.Name)
	//
	//schema := workflow.LoadSchema(uuid.V7(), g)
	//appSupervisor.AddSchema(schema)
	//appSupervisor.SendMessageTo(
	//	actors.WorkflowEngine,
	//	context.Background(),
	//	actors.NewMessage(enginemsg.StartWorkflow, map[string]any{"schema_id": schema.ID()}),
	//)

	//appSupervisor.Stop()

	return nil
}
