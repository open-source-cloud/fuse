package workflow

//goland:noinspection GoUnusedConst
const (
	// FunctionStatusNil nil function output
	FunctionStatusNil FunctionOutputStatus = "nil"
	// FunctionSuccess Success Status
	FunctionSuccess FunctionOutputStatus = "success"
	// FunctionError Error Status
	FunctionError FunctionOutputStatus = "error"
)

// FunctionOutputStatus node output Status type
type FunctionOutputStatus string

// FunctionOutput defines a function output
type FunctionOutput struct {
	Status FunctionOutputStatus `json:"status"`
	Data   map[string]any       `json:"data"`
}

// NewFunctionOutput creates a new node output object with Status and Data with the result of the execution
func NewFunctionOutput(status FunctionOutputStatus, data map[string]any) FunctionOutput {
	return FunctionOutput{
		Status: status,
		Data:   data,
	}
}

// NewFunctionSuccessOutput returns a success function output object
func NewFunctionSuccessOutput(data map[string]any) FunctionOutput {
	return NewFunctionOutput(FunctionSuccess, data)
}
