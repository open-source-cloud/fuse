package workflow

// Function represents an executable Function and it's metadata
type Function interface {
	ID() string
	Metadata() FunctionMetadata
	Execute(input *FunctionInput) (FunctionResult, error)
}
