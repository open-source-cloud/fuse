package messaging

import (
	"fmt"
	"strings"

	"ergo.services/ergo/gen"
	"github.com/open-source-cloud/fuse/internal/workflow/workflowactions"
	"github.com/open-source-cloud/fuse/pkg/workflow"
)

// ExecuteFunctionMessage defines a ExecuteFunction message
type ExecuteFunctionMessage struct {
	WorkflowID workflow.ID     `json:"workflow_id"`
	ExecID     workflow.ExecID `json:"exec_id"`
	ThreadID   uint16          `json:"thread_id"`
	PackageID  string          `json:"package_id"`
	FunctionID string          `json:"function_id"`
	Input      map[string]any  `json:"input"`
	// HandlerPID is the workflow handler process; used for async internal completions from pool workers.
	HandlerPID gen.PID `json:"handlerPid"`
}

// NewExecuteFunctionMessage creates a new ExecuteFunction message
func NewExecuteFunctionMessage(workflowID workflow.ID, execAction *workflowactions.RunFunctionAction, handlerPID gen.PID) Message {
	lastSlashIndex := strings.LastIndex(execAction.FunctionID, "/")

	return Message{
		Type: ExecuteFunction,
		Args: ExecuteFunctionMessage{
			WorkflowID: workflowID,
			ExecID:     execAction.FunctionExecID,
			ThreadID:   execAction.ThreadID,
			PackageID:  execAction.FunctionID[:lastSlashIndex],
			FunctionID: execAction.FunctionID,
			Input:      execAction.Args,
			HandlerPID: handlerPID,
		},
	}
}

// ExecuteFunctionMessage helper function to cast from a generic Message type
func (m Message) ExecuteFunctionMessage() (ExecuteFunctionMessage, error) {
	if m.Type != ExecuteFunction {
		return ExecuteFunctionMessage{}, fmt.Errorf("message type %s is not ExecuteFunction", m.Type)
	}
	return m.Args.(ExecuteFunctionMessage), nil
}
