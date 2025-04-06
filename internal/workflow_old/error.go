package workflow

// ExecutionContext holds the state during workflow execution
type ExecutionContext struct {
	WorkflowID string
	NodeID     string
	Input      interface{}
	Output     interface{}
	Error      error
}

// Error represents a workflow execution error
type Error struct {
	WorkflowID string
	NodeID     string
	Message    string
	Err        error
}

func (e *Error) Error() string {
	return e.Message
}
