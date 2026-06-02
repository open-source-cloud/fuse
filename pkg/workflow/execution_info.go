package workflow

// NewExecutionInfo creates a new ExecutionInfo to pass to an executable function
func NewExecutionInfo(workflowID ID, execID ExecID, input *FunctionInput) *ExecutionInfo {
	return &ExecutionInfo{
		WorkflowID: workflowID,
		ExecID:     execID,
		Input:      input,
	}
}

// ExecutionInfo contains the execution context info for a workflow when executing a Function
type ExecutionInfo struct {
	WorkflowID ID
	ExecID     ExecID
	Input      *FunctionInput
	Finish     func(FunctionOutput)
	// Handle is the worker actor handle, populated by the internal function transport
	// so that long-running functions (e.g. ai/agent) can invoke other functions
	// in-process and reach the node from a goroutine. It is typed as any to keep
	// pkg/workflow free of any internal/* dependency; consumers type-assert it to
	// their own minimal interface. It is nil outside the internal transport path
	// (for example in unit tests that construct ExecutionInfo directly).
	Handle any
}
