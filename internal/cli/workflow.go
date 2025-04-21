package cli

import (
	"fmt"

	"github.com/open-source-cloud/fuse/internal/graph/memory"
	"github.com/open-source-cloud/fuse/internal/providers/debug"
	"github.com/open-source-cloud/fuse/internal/providers/logic"
	"github.com/open-source-cloud/fuse/internal/workflow"
	"github.com/open-source-cloud/fuse/pkg/uuid"
	"github.com/spf13/cobra"
)

// Workflow example command
var workflowCmd = &cobra.Command{
	Use:   "workflow",
	Short: "Workflow runner",
	RunE:  workflowRunner,
}

// Workflow example runner
func workflowRunner(_ *cobra.Command, _ []string) error {
	engine := workflow.NewEngine()

	debugProvider := debug.NewNodeProvider()
	logicProvider := logic.NewNodeProvider()

	nilNode, err := debugProvider.GetNode(debug.NilNodeID)
	if err != nil {
		return err
	}

	rootNodeConfig := memory.NewNodeConfig()
	rootNode := memory.NewNode(uuid.V7(), nilNode, rootNodeConfig)
	newGraph := memory.NewGraph(rootNode)

	randNode, err := logicProvider.GetNode(logic.RandNodeID)
	if err != nil {
		return err
	}

	randNode1Config := memory.NewNodeConfig()
	randNode1 := memory.NewNode(uuid.V7(), randNode, randNode1Config)
	randNodeEdgeID1 := uuid.V7()
	newGraph.AddNode(rootNode.ID(), randNodeEdgeID1, randNode1)

	randNode2Config := memory.NewNodeConfig()
	randNodeEdgeID2 := uuid.V7()
	randNode2 := memory.NewNode(uuid.V7(), randNode, randNode2Config)
	newGraph.AddNode(rootNode.ID(), randNodeEdgeID2, randNode2)

	sumNode, err := logicProvider.GetNode(logic.SumNodeID)
	if err != nil {
		return err
	}

	sumNodeConfig := memory.NewNodeConfig()
	sumNodeConfig.AddInputMapping(fmt.Sprintf("edge[%s]", randNodeEdgeID1), "rand", "values")
	sumNodeConfig.AddInputMapping(fmt.Sprintf("edge[%s]", randNodeEdgeID2), "rand", "values")
	sumNodeWorkflow := memory.NewNode(uuid.V7(), sumNode, sumNodeConfig)
	newGraph.AddNodeMultipleParents([]string{randNode1.ID(), randNode2.ID()}, uuid.V7(), sumNodeWorkflow)

	engine.Start()

	schema := workflow.LoadSchema(uuid.V7(), newGraph)
	engine.AddSchema(schema)
	engine.SendMessage(workflow.NewEngineMessage(workflow.EngineMessageStartWorkflow, schema.ID()))

	quitOnCtrlC()
	engine.Stop()

	return nil
}
