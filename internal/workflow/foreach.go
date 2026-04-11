package workflow

import (
	"fmt"

	"github.com/open-source-cloud/fuse/internal/workflow/workflowactions"
	"github.com/open-source-cloud/fuse/pkg/workflow"
)

// StartForEachIteration creates a new dynamic execution thread for a single
// ForEach iteration and returns the RunFunctionAction that dispatches the first
// node in the loop body.
//
// It finds the "each" output edge from the foreach node (identified by nodeID),
// allocates a new thread ID that does not collide with any existing thread, and
// wires up the audit log / journal entries required for tracing.
//
// The returned threadID must be passed to ForEachState.StartBatch so the
// WorkflowHandler can correlate the result message back to the iteration.
//
// Limitations (Phase 4.1):
//   - The loop body must be a linear chain on a single static thread.
//     Parallel forks inside the loop body are not supported.
func (w *Workflow) StartForEachIteration(nodeID string, iterInput map[string]any) (*workflowactions.RunFunctionAction, uint16, error) {
	forEachNode, err := w.graph.FindNode(nodeID)
	if err != nil {
		return nil, 0, fmt.Errorf("foreach node %q not found: %w", nodeID, err)
	}

	eachEdge := findNamedOutputEdge(forEachNode, "each")
	if eachEdge == nil {
		return nil, 0, fmt.Errorf("foreach node %q has no output edge named \"each\"", nodeID)
	}

	iterNode := eachEdge.To()
	dynamicThreadID := w.threads.AllocateDynamicID()

	// Use the dynamic thread ID in the ExecID so that async result messages can
	// be routed back to the correct thread.
	execID := workflow.NewExecID(dynamicThreadID)

	newThread := w.threads.New(dynamicThreadID, execID)

	w.journal.Append(JournalEntry{
		Type:     JournalForEachIterationStarted,
		ThreadID: newThread.ID(),
		ExecID:   execID.String(),
	})
	w.auditLog.NewEntry(newThread.ID(), iterNode.ID(), execID.String(), iterInput)
	w.journal.Append(JournalEntry{
		Type:           JournalStepStarted,
		ThreadID:       newThread.ID(),
		FunctionNodeID: iterNode.ID(),
		ExecID:         execID.String(),
		Input:          iterInput,
	})

	return &workflowactions.RunFunctionAction{
		ThreadID:       newThread.ID(),
		FunctionID:     iterNode.FunctionID(),
		FunctionExecID: execID,
		Args:           iterInput,
	}, newThread.ID(), nil
}

// CompleteForEach marks the foreach node as complete by recording the aggregated
// results in the audit log.  After this call, Next(forEachThreadID) will route
// to the "done" output edge because the conditional output field "_foreach_phase"
// is set to "done".
func (w *Workflow) CompleteForEach(forEachExecID workflow.ExecID, results []any) {
	aggregated := map[string]any{
		"_foreach_phase": "done",
		"results":        results,
	}
	w.SetResultFor(forEachExecID, &workflow.FunctionResult{
		Output: workflow.NewFunctionSuccessOutput(aggregated),
	})
}

// LastResultForThread returns the output data from the last function executed
// on the given thread, or nil if the thread does not exist or has no result.
func (w *Workflow) LastResultForThread(threadID uint16) any {
	t := w.threads.Get(threadID)
	if t == nil {
		return nil
	}
	entry, exists := w.auditLog.Get(t.CurrentExecID().String())
	if !exists || entry.Result == nil {
		return nil
	}
	return entry.Result.Output.Data
}

// findNamedOutputEdge returns the first output edge of node whose condition name
// matches edgeName, or nil if no such edge exists.
func findNamedOutputEdge(node *Node, edgeName string) *Edge {
	for _, edge := range node.OutputEdges() {
		if edge.Condition() != nil && edge.Condition().Name == edgeName {
			return edge
		}
	}
	return nil
}
