package events

// Predefined event types for workflow lifecycle events
const (
	EventWorkflowTriggered = "workflow.triggered"
	EventWorkflowCompleted = "workflow.completed"
	EventWorkflowFailed    = "workflow.failed"
	EventWorkflowCancelled = "workflow.cancelled"
	EventFunctionCompleted = "function.completed"
	EventFunctionFailed    = "function.failed"
)
