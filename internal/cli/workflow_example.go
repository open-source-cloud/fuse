package cli

import (
	"github.com/open-source-cloud/fuse/internal/graph"
	"github.com/open-source-cloud/fuse/internal/providers/debug"
	"github.com/open-source-cloud/fuse/internal/workflow"
	"github.com/open-source-cloud/fuse/pkg/uuid"
	"github.com/spf13/cobra"
)

// Workflow example command
var workflowCmd = &cobra.Command{
	Use:   "workflow",
	Short: "Workflow example",
	RunE:  workflowExampleRunner,
}

// Workflow example runner
func workflowExampleRunner(_ *cobra.Command, _ []string) error {
	provider := debug.NewNodeProvider()
	rootNode := graph.NewNode(uuid.V7(), provider.Nodes()[0])
	nextNode := graph.NewNode(uuid.V7(), provider.Nodes()[0])
	newGraph := graph.NewGraph(rootNode)
	newGraph.AddNode(rootNode.ID(), nextNode)
	schema := workflow.LoadSchema(uuid.V7(), newGraph)

	engine := workflow.NewEngine()
	engine.Start()

	engine.AddSchema(schema)
	engine.SendMessage(workflow.NewEngineMessage(workflow.EngineMessageStartWorkflow, schema.ID()))

	quitOnCtrlC()
	engine.Stop()

	return nil
}
