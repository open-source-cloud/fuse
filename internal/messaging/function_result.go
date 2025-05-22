package messaging

import (
	"fmt"
	"github.com/open-source-cloud/fuse/internal/workflow"
	pubworkflow "github.com/open-source-cloud/fuse/pkg/workflow"
)

// FunctionResultMessage defines a FunctionResult message
type FunctionResultMessage struct {
	WorkflowID workflow.ID                `json:"workflow_id"`
	ThreadID   int                        `json:"thread_id"`
	ExecID     string                     `json:"exec_id"`
	Result     pubworkflow.FunctionResult `json:"result"`
}

// NewFunctionResultMessage creates a new FunctionResult message
func NewFunctionResultMessage(workflowID workflow.ID, thread int, execID string, result pubworkflow.FunctionResult) Message {
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

// NewFunctionResultMessage helper func to cast from a generic Message type
func (m Message) NewFunctionResultMessage() (ExecuteFunctionMessage, error) {
	if m.Type != ExecuteFunction {
		return ExecuteFunctionMessage{}, fmt.Errorf("message type %s is not ExecuteFunction", m.Type)
	}
	return m.Args.(ExecuteFunctionMessage), nil
}
