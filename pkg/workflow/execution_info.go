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
}
