package workflow

// ExecutionInfo contains the execution context info for a workflow when executing a Function
type ExecutionInfo struct {
	WorkflowID string
	ExecID     string
	Finish     func(FunctionOutput)
}
