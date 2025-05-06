package workflow

// FunctionResult the node result interface that describes the result of a node execution
type FunctionResult interface {
	Async() (<-chan FunctionOutput, bool)
	Output() FunctionOutput
	Map() map[string]any
}

// NewFunctionResult returns a new node result that describes the result of a SYNC node execution with output
func NewFunctionResult(status FunctionOutputStatus, data FunctionOutputData) FunctionResult {
	var outputData FunctionOutputData
	if data != nil {
		outputData = data
	} else {
		outputData = map[string]any{}
	}
	return &nodeResult{
		asyncChan: nil,
		output:    NewFunctionOutput(status, outputData),
	}
}

// NewFunctionResultAsync returns a new node result that describes the result of an ASYNC node execution
func NewFunctionResultAsync(asyncChan <-chan FunctionOutput) FunctionResult {
	return &nodeResult{
		asyncChan: asyncChan,
		output:    nil,
	}
}

type nodeResult struct {
	asyncChan <-chan FunctionOutput
	output    FunctionOutput
}

func (r *nodeResult) Async() (<-chan FunctionOutput, bool) {
	if r.asyncChan != nil {
		return r.asyncChan, true
	}
	return nil, false
}

func (r *nodeResult) Output() FunctionOutput {
	return r.output
}

func (r *nodeResult) Map() map[string]any {
	var status FunctionOutputStatus
	var data FunctionOutputData
	if r.Output() != nil {
		status = r.Output().Status()
		data = r.Output().Data()
	}
	return map[string]any{
		"async":  r.asyncChan != nil,
		"status": status,
		"data":   data,
	}
}
