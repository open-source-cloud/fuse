package cli

import (
	"github.com/open-source-cloud/fuse/internal/graph"
	"github.com/open-source-cloud/fuse/internal/providers/logic"
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
	engine := workflow.NewEngine()

	logicProvider := logic.NewNodeProvider()

	rootNodeConfig := graph.NewNodeConfig()
	rootNode := graph.NewNode(uuid.V7(), logicProvider.Nodes()[1], rootNodeConfig)
	newGraph := graph.NewGraph(rootNode)

	nextNodeConfig := graph.NewNodeConfig()
	nextNodeConfig.AddInputMapping("edge[]", "rand", "value")
	nextNode := graph.NewNode(uuid.V7(), logicProvider.Nodes()[0], nextNodeConfig)
	newGraph.AddNode(rootNode.ID(), "default", nextNode)

	engine.Start()

	schema := workflow.LoadSchema(uuid.V7(), newGraph)
	engine.AddSchema(schema)
	engine.SendMessage(workflow.NewEngineMessage(workflow.EngineMessageStartWorkflow, schema.ID()))

	quitOnCtrlC()
	engine.Stop()

	return nil
}
