package workflow

type Node interface {
	ID() string
	Metadata() NodeMetadata
	Execute(input NodeInput) (NodeResult, error)
}

const (
	NodeOutputStatusSuccess NodeOutputStatus = "success"
	NodeOutputStatusError   NodeOutputStatus = "error"
)

type NodeInput map[string]any
type NodeOutputStatus string
type NodeOutputData any
type NodeOutput interface {
	Status() NodeOutputStatus
	Data() NodeOutputData
}

type nodeOutput struct {
	status NodeOutputStatus
	data   NodeOutputData
}

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

type NodeResult interface {
	Async() (chan NodeOutput, bool)
	Output() NodeOutput
}

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
