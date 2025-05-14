package messaging

import (
	"fmt"
	"github.com/open-source-cloud/fuse/internal/workflow"
	"github.com/stretchr/objx"
	"strings"
)

type ExecuteFunctionMessage struct {
	WorkflowID workflow.ID
	ExecID     string
	PackageID  string
	FunctionID string
	Input      objx.Map
}

func NewExecuteFunctionMessage(workflowID workflow.ID, functionID string, functionExecID string, input objx.Map) Message {
	lastSlashIndex := strings.LastIndex(functionID, "/")

	return Message{
		Type: ExecuteFunction,
		Args: ExecuteFunctionMessage{
			WorkflowID: workflowID,
			ExecID:     functionExecID,
			PackageID:  functionID[:lastSlashIndex],
			FunctionID: functionID,
			Input:      input,
		},
	}
}

func (m Message) ExecuteFunctionMessage() (ExecuteFunctionMessage, error) {
	if m.Type != ExecuteFunction {
		return ExecuteFunctionMessage{}, fmt.Errorf("message type %s is not ExecuteFunction", m.Type)
	}
	return m.Args.(ExecuteFunctionMessage), nil
}
