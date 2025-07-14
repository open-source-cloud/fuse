package workflow

// Function describes an executable workflow Function
type Function func(*ExecutionInfo) (FunctionResult, error)
