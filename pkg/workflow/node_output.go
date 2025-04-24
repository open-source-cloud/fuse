package workflow

const (
	// NodeOutputStatusSuccess Success status
	NodeOutputStatusSuccess NodeOutputStatus = "success"
	// NodeOutputStatusError Error status
	NodeOutputStatusError NodeOutputStatus = "error"
)

// NodeOutputStatus node output status type
type NodeOutputStatus string

// NodeOutputData node output data type
type NodeOutputData any

// NodeOutput node output interface that should provide status and data accessors
type NodeOutput interface {
	Status() NodeOutputStatus
	Data() NodeOutputData
}

type nodeOutput struct {
	status NodeOutputStatus
	data   NodeOutputData
}

// NewNodeOutput creates a new node output object with status and data with the result of the execution
func NewNodeOutput(status NodeOutputStatus, data NodeOutputData) NodeOutput {
	return &nodeOutput{
		status: status,
		data:   data,
	}
}

func (o *nodeOutput) Status() NodeOutputStatus {
	return o.status
}

func (o *nodeOutput) Data() NodeOutputData {
	return o.data
}
