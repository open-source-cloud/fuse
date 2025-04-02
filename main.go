// Package main provides the entry point for the workflow engine application.
// It demonstrates the usage of the workflow engine with example workflows.
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/gustavobertoi/core-workflow-poc/pkg/logic"
	"github.com/gustavobertoi/core-workflow-poc/pkg/strproc"
	"github.com/gustavobertoi/core-workflow-poc/workflow"
)

func main() {
	// Register node providers
	providers := map[string]workflow.NodeProvider{
		"string": strproc.NewStringProcessorProvider(),
		"logic":  logic.NewLogicProcessorProvider(),
	}

	// Load and execute string workflow
	stringWF, err := workflow.LoadWorkflowFromYAML("examples/string_workflow.yaml")
	if err != nil {
		log.Fatalf("Failed to load string workflow: %v", err)
	}

	wf, err := workflow.ConvertYAMLToWorkflow(stringWF, providers)
	if err != nil {
		log.Fatalf("Failed to convert string workflow: %v", err)
	}

	engine := workflow.NewDefaultEngine()
	result, err := engine.ExecuteWorkflow(context.Background(), wf, nil)
	if err != nil {
		log.Fatalf("Failed to execute string workflow: %v", err)
	}

	fmt.Printf("String workflow result: %v\n", result)

	// Load and execute logical workflow
	logicalWF, err := workflow.LoadWorkflowFromYAML("examples/logical_workflow.yaml")
	if err != nil {
		log.Fatalf("Failed to load logical workflow: %v", err)
	}

	wf, err = workflow.ConvertYAMLToWorkflow(logicalWF, providers)
	if err != nil {
		log.Fatalf("Failed to convert logical workflow: %v", err)
	}

	result, err = engine.ExecuteWorkflow(context.Background(), wf, nil)
	if err != nil {
		log.Fatalf("Failed to execute logical workflow: %v", err)
	}

	fmt.Printf("Logical workflow result: %v\n", result)
}
