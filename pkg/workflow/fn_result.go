package workflow

// FunctionResult the node result interface that describes the result of a node execution
type FunctionResult struct {
	async bool
	output    FunctionOutput
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
		output:    NewFunctionOutput(status, outputData),
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
		async:     true,
		output:    nil,
	}
}


func (r FunctionResult) IsAsync() bool {
	return r.async
}

func (r FunctionResult) Output() FunctionOutput {
	return r.output
}

func (r FunctionResult) Raw() map[string]any {
	var status FunctionOutputStatus
	var data FunctionOutputData
	if r.Output() != nil {
		status = r.Output().Status()
		data = r.Output().Data()
	}
	return map[string]any{
		"async":  r.async,
		"status": status,
		"data":   data,
	}
}
