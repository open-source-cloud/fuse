package messaging

import (
	"testing"

	"github.com/open-source-cloud/fuse/pkg/workflow"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCancelWorkflowMessage(t *testing.T) {
	wfID := workflow.NewID()
	msg := NewCancelWorkflowMessage(wfID, "user requested")

	assert.Equal(t, CancelWorkflow, msg.Type)
	cancelMsg, ok := msg.Args.(CancelWorkflowMessage)
	require.True(t, ok)
	assert.Equal(t, wfID, cancelMsg.WorkflowID)
	assert.Equal(t, "user requested", cancelMsg.Reason)
}

func TestMessage_CancelWorkflowMessage_Success(t *testing.T) {
	wfID := workflow.NewID()
	msg := NewCancelWorkflowMessage(wfID, "test")

	result, err := msg.CancelWorkflowMessage()

	require.NoError(t, err)
	assert.Equal(t, wfID, result.WorkflowID)
	assert.Equal(t, "test", result.Reason)
}

func TestMessage_CancelWorkflowMessage_WrongType(t *testing.T) {
	msg := Message{Type: TriggerWorkflow, Args: nil}

	_, err := msg.CancelWorkflowMessage()

	assert.Error(t, err)
}
