package messaging

import "github.com/open-source-cloud/fuse/pkg/workflow"

// TimeoutMessage defines an execution timeout message
type TimeoutMessage struct {
	ExecID string
}

// NewTimeoutMessage creates a new execution timeout message
func NewTimeoutMessage(execID string) Message {
	return Message{
		Type: Timeout,
		Args: TimeoutMessage{ExecID: execID},
	}
}

// WorkflowTimeoutMessage defines a workflow-level timeout message
type WorkflowTimeoutMessage struct {
	WorkflowID workflow.ID
}

// NewWorkflowTimeoutMessage creates a new workflow timeout message
func NewWorkflowTimeoutMessage(workflowID workflow.ID) Message {
	return Message{
		Type: WorkflowTimeout,
		Args: WorkflowTimeoutMessage{WorkflowID: workflowID},
	}
}
