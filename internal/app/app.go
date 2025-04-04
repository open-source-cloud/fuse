package app

import (
	"context"
	"github.com/mattn/go-colorable"
	"github.com/open-source-cloud/fuse/internal/workflow"
	"github.com/open-source-cloud/fuse/pkg/logic"
	"github.com/open-source-cloud/fuse/pkg/schema"
	"github.com/open-source-cloud/fuse/pkg/strproc"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"time"
)

func initLog() {
	zerolog.TimeFieldFormat = time.TimeOnly
	zerolog.SetGlobalLevel(zerolog.DebugLevel)
	log.Logger = log.Output(zerolog.ConsoleWriter{
		Out:        colorable.NewColorableStdout(),
		TimeFormat: time.TimeOnly,
	}).With().Caller().Logger()
}

func runApplication() {
	initLog()

	// Register node providers
	providers := map[string]workflow.NodeProvider{
		"string": strproc.NewStringProcessorProvider(),
		"logic":  logic.NewLogicProcessorProvider(),
		"schema": schema.NewSchemaValidatorProvider(),
	}

	// Load and execute string workflow
	stringWF, err := workflow.LoadWorkflowFromYAML("examples/workflow/string_workflow.yaml")
	if err != nil {
		log.Info().Msgf("Failed to load string workflow: %v", err)
	}

	wf, err := workflow.ConvertYAMLToWorkflow(stringWF, providers)
	if err != nil {
		log.Info().Msgf("Failed to convert string workflow: %v", err)
	}

	engine := workflow.NewDefaultEngine()
	result, err := engine.ExecuteWorkflow(context.Background(), wf, nil)
	if err != nil {
		log.Info().Msgf("Failed to execute string workflow: %v", err)
	}

	log.Info().Msgf("String workflow result: %v", result)

	// Load and execute logical workflow
	logicalWF, err := workflow.LoadWorkflowFromYAML("examples/workflow/logical_workflow.yaml")
	if err != nil {
		log.Info().Msgf("Failed to load logical workflow: %v", err)
	}

	wf, err = workflow.ConvertYAMLToWorkflow(logicalWF, providers)
	if err != nil {
		log.Info().Msgf("Failed to convert logical workflow: %v", err)
	}

	result, err = engine.ExecuteWorkflow(context.Background(), wf, nil)
	if err != nil {
		log.Info().Msgf("Failed to execute logical workflow: %v", err)
	}

	log.Info().Msgf("Logical workflow result: %v", result)

	// Load and execute schema validation workflow
	schemaWF, err := workflow.LoadWorkflowFromYAML("examples/workflow/schema_workflow.yaml")
	if err != nil {
		log.Info().Msgf("Failed to load schema workflow: %v", err)
	}

	wf, err = workflow.ConvertYAMLToWorkflow(schemaWF, providers)
	if err != nil {
		log.Info().Msgf("Failed to convert schema workflow: %v", err)
	}

	// Create sample valid user data
	userData := map[string]interface{}{
		"username": "johndoe",
		"email":    "john.doe@example.com",
		"age":      float64(25),
		"address": map[string]interface{}{
			"street": "123 Main St",
			"city":   "Anytown",
			"state":  "CA",
			"zip":    "12345",
		},
		"preferences": map[string]interface{}{
			"theme":         "light",
			"notifications": true,
		},
	}

	result, err = engine.ExecuteWorkflow(context.Background(), wf, userData)
	if err != nil {
		log.Info().Msgf("Failed to execute schema workflow: %v", err)
	}

	log.Info().Msgf("Schema workflow result: %v", result)
}
