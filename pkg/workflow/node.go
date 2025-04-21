package workflow

// Node represents an executable Node and it's metadata
type Node interface {
	ID() string
	Metadata() NodeMetadata
	Execute(input NodeInput) (NodeResult, error)
}

const (
	// NodeOutputStatusSuccess Success status
	NodeOutputStatusSuccess NodeOutputStatus = "success"
	// NodeOutputStatusError Error status
	NodeOutputStatusError NodeOutputStatus = "error"
)

// NodeInput node input type
type NodeInput map[string]any

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

// NodeResult the node result interface that describes the result of a node execution
type NodeResult interface {
	Async() (chan NodeOutput, bool)
	Output() NodeOutput
}

// NewNodeResult returns a new node result that describes the result of a SYNC node execution with output
func NewNodeResult(status NodeOutputStatus, data NodeOutputData) NodeResult {
	var outputData NodeOutputData
	if data != nil {
		outputData = data
	} else {
		outputData = map[string]any{}
	}
	return &nodeResult{
		asyncChan: nil,
		output:    NewNodeOutput(status, outputData),
	}
}

// NewNodeResultAsync returns a new node result that describes the result of an ASYNC node execution
func NewNodeResultAsync(asyncChan chan NodeOutput) NodeResult {
	return &nodeResult{
		asyncChan: asyncChan,
		output:    nil,
	}
}

type nodeResult struct {
	asyncChan chan NodeOutput
	output    NodeOutput
}

func (r *nodeResult) Async() (chan NodeOutput, bool) {
	if r.asyncChan != nil {
		return r.asyncChan, true
	}
	return nil, false
}

func (r *nodeResult) Output() NodeOutput {
	return r.output
}
