package workflow

// Function describes an executable workflow Function
type Function func(*ExecutionInfo, *FunctionInput) (FunctionResult, error)
