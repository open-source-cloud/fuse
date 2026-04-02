package messaging

import "github.com/open-source-cloud/fuse/pkg/workflow"

// SleepWakeUpMessage defines a SleepWakeUp message
type SleepWakeUpMessage struct {
	WorkflowID workflow.ID
	ExecID     workflow.ExecID
	ThreadID   uint16
}

// NewSleepWakeUpMessage creates a new SleepWakeUp message
func NewSleepWakeUpMessage(workflowID workflow.ID, execID workflow.ExecID, threadID uint16) Message {
	return Message{
		Type: SleepWakeUp,
		Args: SleepWakeUpMessage{
			WorkflowID: workflowID,
			ExecID:     execID,
			ThreadID:   threadID,
		},
	}
}
