package messaging

import (
	"testing"

	"github.com/open-source-cloud/fuse/pkg/workflow"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewTriggerWorkflowMessage(t *testing.T) {
	wfID := workflow.NewID()
	msg := NewTriggerWorkflowMessage("schema-1", wfID)

	assert.Equal(t, TriggerWorkflow, msg.Type)

	triggerMsg, err := msg.TriggerWorkflowMessage()
	require.NoError(t, err)
	assert.Equal(t, "schema-1", triggerMsg.SchemaID)
	assert.Equal(t, wfID, triggerMsg.WorkflowID)
	assert.Nil(t, triggerMsg.Input)
}

func TestNewTriggerWorkflowWithInputMessage(t *testing.T) {
	wfID := workflow.NewID()
	input := map[string]any{"event": "push", "ref": "main"}
	msg := NewTriggerWorkflowWithInputMessage("schema-2", wfID, input)

	assert.Equal(t, TriggerWorkflow, msg.Type)

	triggerMsg, err := msg.TriggerWorkflowMessage()
	require.NoError(t, err)
	assert.Equal(t, "schema-2", triggerMsg.SchemaID)
	assert.Equal(t, wfID, triggerMsg.WorkflowID)
	assert.Equal(t, "push", triggerMsg.Input["event"])
	assert.Equal(t, "main", triggerMsg.Input["ref"])
}

func TestTriggerWorkflowMessage_WrongType(t *testing.T) {
	msg := Message{Type: CancelWorkflow, Args: nil}

	_, err := msg.TriggerWorkflowMessage()
	assert.Error(t, err)
}
