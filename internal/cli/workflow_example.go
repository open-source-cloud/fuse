package cli

import (
	"fmt"
	"github.com/open-source-cloud/fuse/internal/graph"
	"github.com/open-source-cloud/fuse/internal/providers/debug"
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

	debugProvider := debug.NewNodeProvider()
	logicProvider := logic.NewNodeProvider()

	rootNodeConfig := graph.NewNodeConfig()
	rootNode := graph.NewNode(uuid.V7(), debugProvider.Nodes()[0], rootNodeConfig)
	newGraph := graph.NewGraph(rootNode)

	randNode1Config := graph.NewNodeConfig()
	randNode1 := graph.NewNode(uuid.V7(), logicProvider.Nodes()[1], randNode1Config)
	randNodeEdgeID1 := uuid.V7()
	newGraph.AddNode(rootNode.ID(), randNodeEdgeID1, randNode1)
	randNode2Config := graph.NewNodeConfig()
	randNodeEdgeID2 := uuid.V7()
	randNode2 := graph.NewNode(uuid.V7(), logicProvider.Nodes()[1], randNode2Config)
	newGraph.AddNode(rootNode.ID(), randNodeEdgeID2, randNode2)

	sumNodeConfig := graph.NewNodeConfig()
	sumNodeConfig.AddInputMapping(fmt.Sprintf("edge[%s]", randNodeEdgeID1), "rand", "values")
	sumNodeConfig.AddInputMapping(fmt.Sprintf("edge[%s]", randNodeEdgeID2), "rand", "values")
	sumNode := graph.NewNode(uuid.V7(), logicProvider.Nodes()[0], sumNodeConfig)
	newGraph.AddNodeMultipleParents([]string{randNode1.ID(), randNode2.ID()}, uuid.V7(), sumNode)

	engine.Start()

	schema := workflow.LoadSchema(uuid.V7(), newGraph)
	engine.AddSchema(schema)
	engine.SendMessage(workflow.NewEngineMessage(workflow.EngineMessageStartWorkflow, schema.ID()))

	quitOnCtrlC()
	engine.Stop()

	return nil
}
