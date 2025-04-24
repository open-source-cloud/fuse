package workflow

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
