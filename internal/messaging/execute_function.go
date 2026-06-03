package messaging

import (
	"fmt"
	"strings"

	"github.com/open-source-cloud/fuse/internal/workflow/workflowactions"
	"github.com/open-source-cloud/fuse/pkg/workflow"
)

// ExecuteFunctionMessage defines a ExecuteFunction message
type ExecuteFunctionMessage struct {
	WorkflowID  workflow.ID     `json:"workflow_id"`
	ExecID      workflow.ExecID `json:"exec_id"`
	ThreadID    uint16          `json:"thread_id"`
	PackageID   string          `json:"package_id"`
	FunctionID  string          `json:"function_id"`
	Input       map[string]any  `json:"input"`
	Environment string          `json:"environment"`
}

// NewExecuteFunctionMessage creates a new ExecuteFunction message.
// environment is the workflow's resolution scope (ADR-0031), carried to the function so the
// engine can resolve per-context capabilities. Pass a non-nil traceCarrier to propagate the
// calling span's context to the worker.
func NewExecuteFunctionMessage(workflowID workflow.ID, execAction *workflowactions.RunFunctionAction, environment string, traceCarrier map[string]string) Message {
	lastSlashIndex := strings.LastIndex(execAction.FunctionID, "/")

	return Message{
		Type:         ExecuteFunction,
		TraceCarrier: traceCarrier,
		Args: ExecuteFunctionMessage{
			WorkflowID:  workflowID,
			ExecID:      execAction.FunctionExecID,
			ThreadID:    execAction.ThreadID,
			PackageID:   execAction.FunctionID[:lastSlashIndex],
			FunctionID:  execAction.FunctionID,
			Input:       execAction.Args,
			Environment: environment,
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
