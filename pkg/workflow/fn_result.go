package workflow

// FunctionResult the node result interface that describes the result of a node execution
type FunctionResult struct {
	Async  bool           `json:"async"`
	Output FunctionOutput `json:"output"`
}

// NewFunctionResult returns a new node result that describes the result of a SYNC node execution with output
func NewFunctionResult(status FunctionOutputStatus, data map[string]any) FunctionResult {
	var outputData map[string]any
	if data != nil {
		outputData = data
	} else {
		outputData = map[string]any{}
	}
	return FunctionResult{
		Output: NewFunctionOutput(status, outputData),
	}
}

// NewFunctionResultSuccess creates a new function result as success, with nil data
func NewFunctionResultSuccess() FunctionResult {
	return NewFunctionResult(FunctionSuccess, nil)
}

// NewFunctionResultSuccessWith creates a new function result as success, with provided data
func NewFunctionResultSuccessWith(data map[string]any) FunctionResult {
	return NewFunctionResult(FunctionSuccess, data)
}

// NewFunctionResultError creates a new function result as error, with provided error.
// The Go error return is nil: logical failure is carried in FunctionResult.Output (FunctionError).
// The error return from workflow.Function is reserved for unexpected execution failures
// (the workflow actor must still receive a FunctionResultMessage for FunctionError).
func NewFunctionResultError(err error) (FunctionResult, error) {
	if err == nil {
		return NewFunctionResult(FunctionError, map[string]any{"error": "unknown error"}), nil
	}
	return NewFunctionResult(FunctionError, map[string]any{"error": err.Error()}), nil
}

// NewFunctionResultAsync returns a new node result that describes the result of an ASYNC node execution
func NewFunctionResultAsync() FunctionResult {
	return FunctionResult{
		Async:  true,
		Output: NewFunctionOutput(FunctionSuccess, nil),
	}
}
