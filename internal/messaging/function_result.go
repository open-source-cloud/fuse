package messaging

import (
	"fmt"
	"github.com/open-source-cloud/fuse/internal/workflow"
	pubworkflow "github.com/open-source-cloud/fuse/pkg/workflow"
)

// FunctionResultMessage defines a FunctionResult message
type FunctionResultMessage struct {
	WorkflowID workflow.ID                `json:"workflow_id"`
	ThreadID   uint16                     `json:"thread_id"`
	ExecID     workflow.ExecID            `json:"exec_id"`
	Result     pubworkflow.FunctionResult `json:"result"`
}

// NewFunctionResultMessage creates a new FunctionResult message
func NewFunctionResultMessage(workflowID workflow.ID, thread uint16, execID workflow.ExecID, result pubworkflow.FunctionResult) Message {
	return Message{
		Type: FunctionResult,
		Args: FunctionResultMessage{
			WorkflowID: workflowID,
			ThreadID:   thread,
			ExecID:     execID,
			Result:     result,
		},
	}
}

// FunctionResultMessage helper func to cast from a generic Message type
func (m Message) FunctionResultMessage() (FunctionResultMessage, error) {
	if m.Type != FunctionResult {
		return FunctionResultMessage{}, fmt.Errorf("message type %s is not FunctionResultMessage", m.Type)
	}
	return m.Args.(FunctionResultMessage), nil
}
