package messaging

import (
	"fmt"
	"github.com/open-source-cloud/fuse/internal/workflow"
	pubworkflow "github.com/open-source-cloud/fuse/pkg/workflow"
)

type FunctionResultMessage struct {
	WorkflowID workflow.ID
	ExecID string
	Result pubworkflow.FunctionResult
}

func NewFunctionResultMessage(workflowID workflow.ID, execID string, result pubworkflow.FunctionResult) Message {
	return Message{
		Type: FunctionResult,
		Args: FunctionResultMessage{
			WorkflowID: workflowID,
			ExecID:     execID,
			Result:     result,
		},
	}
}

func (m Message) NewFunctionResultMessage() (ExecuteFunctionMessage, error) {
	if m.Type != ExecuteFunction {
		return ExecuteFunctionMessage{}, fmt.Errorf("message type %s is not ExecuteFunction", m.Type)
	}
	return m.Args.(ExecuteFunctionMessage), nil
}
