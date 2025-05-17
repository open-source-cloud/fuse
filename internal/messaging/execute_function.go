package messaging

import (
	"fmt"
	"github.com/open-source-cloud/fuse/internal/workflow"
	"strings"
)

type ExecuteFunctionMessage struct {
	WorkflowID workflow.ID    `json:"workflow_id"`
	ThreadID   int            `json:"thread_id"`
	ExecID     string         `json:"exec_id"`
	PackageID  string         `json:"package_id"`
	FunctionID string         `json:"function_id"`
	Input      map[string]any `json:"input"`
}

func NewExecuteFunctionMessage(workflowID workflow.ID, execAction *workflow.RunFunctionAction) Message {
	lastSlashIndex := strings.LastIndex(execAction.FunctionID, "/")

	return Message{
		Type: ExecuteFunction,
		Args: ExecuteFunctionMessage{
			WorkflowID: workflowID,
			ThreadID:   execAction.ThreadID,
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
