package workflow

// NewExecutionInfo creates a new ExecutionInfo to pass to an executable function
func NewExecutionInfo(workflowID ID, execID ExecID, environment string, input *FunctionInput) *ExecutionInfo {
	return &ExecutionInfo{
		WorkflowID:  workflowID,
		ExecID:      execID,
		Environment: environment,
		Input:       input,
	}
}

// ExecutionInfo contains the execution context info for a workflow when executing a Function
type ExecutionInfo struct {
	WorkflowID ID
	ExecID     ExecID
	// Environment is the resolution scope (ADR-0031) of the running workflow. It lets the engine
	// resolve per-context capabilities (e.g. LLM provider keys) without the function touching the
	// secret store; it is scope data, not a secret.
	Environment string
	Input       *FunctionInput
	Finish      func(FunctionOutput)
}
