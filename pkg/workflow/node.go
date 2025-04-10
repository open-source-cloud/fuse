package workflow

type Node interface {
	ID() string
	Params() Params
	Execute(input NodeInput) (NodeResult, error)
}

const (
	NodeOutputStatusSuccess NodeOutputStatus = "success"
	NodeOutputStatusError   NodeOutputStatus = "error"
)

type NodeInput map[string]interface{}
type NodeOutputStatus string
type NodeOutputData interface{}
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
	return &nodeResult{
		asyncChan: nil,
		output:    NewNodeOutput(status, data),
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
