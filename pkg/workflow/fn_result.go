package workflow

// FunctionResult the node result interface that describes the result of a node execution
type FunctionResult struct {
	Async  bool           `json:"async"`
	Output FunctionOutput `json:"output"`
}

// NewFunctionResult returns a new node result that describes the result of a SYNC node execution with output
func NewFunctionResult(status FunctionOutputStatus, data FunctionOutputData) FunctionResult {
	var outputData FunctionOutputData
	if data != nil {
		outputData = data
	} else {
		outputData = map[string]any{}
	}
	return FunctionResult{
		Output: NewFunctionOutput(status, outputData),
	}
}

func NewFunctionResultSuccess() FunctionResult {
	return NewFunctionResult(FunctionSuccess, nil)
}
func NewFunctionResultSuccessWith(data FunctionOutputData) FunctionResult {
	return NewFunctionResult(FunctionSuccess, data)
}

func NewFunctionResultError(err error) (FunctionResult, error) {
	return NewFunctionResult(FunctionError, map[string]any{"error": err}), err
}

// NewFunctionResultAsync returns a new node result that describes the result of an ASYNC node execution
func NewFunctionResultAsync() FunctionResult {
	return FunctionResult{
		Async:  true,
		Output: NewFunctionOutput(FunctionSuccess, nil),
	}
}
