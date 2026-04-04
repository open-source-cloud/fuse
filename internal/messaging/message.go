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
	// CancelWorkflow message type - cancel a running workflow
	CancelWorkflow MessageType = "workflow:cancel"
	// SleepWakeUp message type - wake up a sleeping workflow
	SleepWakeUp MessageType = "workflow:sleep:wakeup"
	// AwakeableResolvedMsg message type - an awakeable has been resolved
	AwakeableResolvedMsg MessageType = "workflow:awakeable:resolved"
	// SubWorkflowCompleted message type - a sub-workflow has completed
	SubWorkflowCompleted MessageType = "workflow:subworkflow:completed"
	// PublishGraphSchemaUpsert message type - local schema saved; replication actor should SendEvent
	PublishGraphSchemaUpsert MessageType = "schema:publish-upsert"
	// RetryNode message type - manually retry a specific failed node
	RetryNode MessageType = "workflow:retry-node"
)

// Message defines the basic Message
type Message struct {
	Type MessageType
	Args any
}
