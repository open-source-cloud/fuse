package messaging

import (
	"fmt"
	"github.com/open-source-cloud/fuse/internal/workflow"
	pubworkflow "github.com/open-source-cloud/fuse/pkg/workflow"
)

// AsyncFunctionResultMessage defines an AsyncFunctionResult message
type AsyncFunctionResultMessage struct {
	WorkflowID workflow.ID                `json:"workflow_id"`
	ExecID     workflow.ExecID            `json:"exec_id"`
	Output     pubworkflow.FunctionOutput `json:"output"`
}

// NewAsyncFunctionResultMessage creates a new AsyncFunctionResult message
func NewAsyncFunctionResultMessage(workflowID string, execID string, output pubworkflow.FunctionOutput) Message {
	return Message{
		Type: AsyncFunctionResult,
		Args: AsyncFunctionResultMessage{
			WorkflowID: workflow.ID(workflowID),
			ExecID:     workflow.ExecID(execID),
			Output:     output,
		},
	}
}

// AsyncFunctionResultMessage helper func to cast from a generic Message type
func (m Message) AsyncFunctionResultMessage() (AsyncFunctionResultMessage, error) {
	if m.Type != AsyncFunctionResult {
		return AsyncFunctionResultMessage{}, fmt.Errorf("message type %s is not AsyncFunctionResultMessage", m.Type)
	}
	return m.Args.(AsyncFunctionResultMessage), nil
}
