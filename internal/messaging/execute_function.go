package messaging

import (
	"fmt"
	"github.com/open-source-cloud/fuse/internal/workflow"
	"github.com/open-source-cloud/fuse/pkg/uuid"
	"strings"
)

type ExecuteFunctionMessage struct {
	WorkflowID workflow.ID
	ExecID     string
	PackageID  string
	FunctionID string
	Input      map[string]any
}

func NewExecuteFunctionMessage(workflowID workflow.ID, functionID string, input map[string]any) Message {
	lastSlashIndex := strings.LastIndex(functionID, "/")

	return Message{
		Type: ExecuteFunction,
		Args: ExecuteFunctionMessage{
			WorkflowID: workflowID,
			ExecID:     uuid.V7(),
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
