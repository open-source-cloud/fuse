package messaging

import (
	"fmt"

	"github.com/open-source-cloud/fuse/pkg/workflow"
)

// RetryNodeMessage requests a manual retry of a specific failed node execution.
type RetryNodeMessage struct {
	WorkflowID workflow.ID
	ExecID     workflow.ExecID
}

// NewRetryNodeMessage creates a new RetryNode message.
func NewRetryNodeMessage(workflowID workflow.ID, execID workflow.ExecID) Message {
	return Message{
		Type: RetryNode,
		Args: RetryNodeMessage{
			WorkflowID: workflowID,
			ExecID:     execID,
		},
	}
}

// RetryNodeMessage helper func to cast from a generic Message type.
func (m Message) RetryNodeMessage() (RetryNodeMessage, error) {
	if m.Type != RetryNode {
		return RetryNodeMessage{}, fmt.Errorf("message type %s is not RetryNode", m.Type)
	}
	return m.Args.(RetryNodeMessage), nil
}
