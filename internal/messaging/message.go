// Package messaging defines typed messages exchanged between workflow actors.
package messaging

// MessageType defines the MessageType type
type MessageType string

const (
	// TriggerWorkflow message type
	TriggerWorkflow MessageType = "workflow:trigger"
	// ExecuteFunction message type
	ExecuteFunction MessageType = "function:execute"
	// FunctionResult message type
	FunctionResult MessageType = "function:result"
	// AsyncFunctionResult message type
	AsyncFunctionResult MessageType = "function:async:result"
	// WorkflowCompleted message type
	WorkflowCompleted MessageType = "workflow:completed"
	// RecoverWorkflows message type - triggers startup recovery of in-progress workflows
	RecoverWorkflows MessageType = "workflow:recover"
	// Timeout message type - execution timeout for a single node
	Timeout MessageType = "execution:timeout"
	// WorkflowTimeout message type - total workflow execution timeout
	WorkflowTimeout MessageType = "workflow:timeout"
)

// Message defines the basic Message
type Message struct {
	Type MessageType
	Args any
}
