package workflow

const (
	FunctionStatusNil FunctionOutputStatus = "nil"
	// FunctionSuccess Success Status
	FunctionSuccess FunctionOutputStatus = "success"
	// FunctionError Error Status
	FunctionError FunctionOutputStatus = "error"
)

// FunctionOutputStatus node output Status type
type FunctionOutputStatus string

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
