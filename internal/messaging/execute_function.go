package messaging

import (
	"fmt"
	"github.com/open-source-cloud/fuse/internal/workflow"
	"strings"
)

type ExecuteFunctionMessage struct {
	WorkflowID workflow.ID
	Thread     int
	ExecID     string
	PackageID  string
	FunctionID string
	Input      map[string]any
}

func NewExecuteFunctionMessage(workflowID workflow.ID, execAction *workflow.RunFunctionAction) Message {
	lastSlashIndex := strings.LastIndex(execAction.FunctionID, "/")

	return Message{
		Type: ExecuteFunction,
		Args: ExecuteFunctionMessage{
			WorkflowID: workflowID,
			Thread:     execAction.ThreadID,
			ExecID:     execAction.FunctionExecID,
			PackageID:  execAction.FunctionID[:lastSlashIndex],
			FunctionID: execAction.FunctionID,
			Input:      execAction.Args,
		},
	}
}

func (m Message) ExecuteFunctionMessage() (ExecuteFunctionMessage, error) {
	if m.Type != ExecuteFunction {
		return ExecuteFunctionMessage{}, fmt.Errorf("message type %s is not ExecuteFunction", m.Type)
	}
	return m.Args.(ExecuteFunctionMessage), nil
}
