package workflow

// ExecutionInfo contains the execution context info for a workflow when executing a Function
type ExecutionInfo struct {
	WorkflowID ID
	ExecID     ExecID
	Finish     func(FunctionOutput)
}
