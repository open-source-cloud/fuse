package workflow

const (
	// FunctionSuccess Success status
	FunctionSuccess FunctionOutputStatus = "success"
	// FunctionError Error status
	FunctionError FunctionOutputStatus = "error"
)

// FunctionOutputStatus node output status type
type FunctionOutputStatus string

// FunctionOutputData node output data type
type FunctionOutputData map[string]any

// FunctionOutput node output interface that should provide status and data accessors
type FunctionOutput interface {
	Status() FunctionOutputStatus
	Data() FunctionOutputData
}

type functionOutput struct {
	status FunctionOutputStatus
	data   FunctionOutputData
}

// NewFunctionOutput creates a new node output object with status and data with the result of the execution
func NewFunctionOutput(status FunctionOutputStatus, data FunctionOutputData) FunctionOutput {
	return &functionOutput{
		status: status,
		data:   data,
	}
}

func (o *functionOutput) Status() FunctionOutputStatus {
	return o.status
}

func (o *functionOutput) Data() FunctionOutputData {
	return o.data
}
