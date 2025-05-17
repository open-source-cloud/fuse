package workflow

// Function describes an executable workflow Function
type Function func(input *FunctionInput) (FunctionResult, error)
