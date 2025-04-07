package cli

import (
	"time"

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
	rootNodeId := uuid.V7()
	logNodeId := uuid.V7()
	testGraph := graph.NewGraph(rootNodeId, &debug.NullNode{})
	_ = testGraph.AddNode(
		rootNodeId,
		uuid.V7(),
		workflow.NewDefaultEdge(uuid.V7(), nil),
		logNodeId,
		&debug.LogNode{},
	)

	testWorkflow, _ := workflow.LoadSchema(uuid.V7(), testGraph)

	executeSignalChan := make(chan workflow.ExecuteSignal)
	go func() {
		time.Sleep(2 * time.Second)
		executeSignalChan <- workflow.ExecuteSignal{
			Signal: "workflow-start",
			Data:   testWorkflow,
		}
	}()

	workflowEngine := workflow.NewDefaultEngine(executeSignalChan)
	_ = workflowEngine.RegisterNodeProvider(debug.NewNodeProvider())

	_ = workflowEngine.Run()

	return nil
}
