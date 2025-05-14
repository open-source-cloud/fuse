package workflow

const (
	FunctionStatusNil FunctionOutputStatus = "nil"
	// FunctionSuccess Success status
	FunctionSuccess FunctionOutputStatus = "success"
	// FunctionError Error status
	FunctionError FunctionOutputStatus = "error"
)

// FunctionOutputStatus node output status type
type FunctionOutputStatus string

// FunctionOutputData node output data type
type FunctionOutputData map[string]any

type FunctionOutput struct {
	status FunctionOutputStatus
	data   FunctionOutputData
}

// NewFunctionOutput creates a new node output object with status and data with the result of the execution
func NewFunctionOutput(status FunctionOutputStatus, data FunctionOutputData) FunctionOutput {
	return FunctionOutput{
		status: status,
		data:   data,
	}
}

func (o FunctionOutput) Status() FunctionOutputStatus {
	return o.status
}

func (o FunctionOutput) Data() FunctionOutputData {
	return o.data
}
