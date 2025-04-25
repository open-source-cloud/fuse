package audit

import (
	"github.com/open-source-cloud/fuse/pkg/graph"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// Event wraps zerolog.Event and allows custom methods.
type Event struct {
	*zerolog.Event
}

func (e *Event) Workflow(id string) *Event {
	e.Event = e.Event.Str("workflow", id)
	return e
}

func (e *Event) WorkflowState(id string, state any) *Event {
	e.Event = e.Event.Str("workflow", id).Any("state", state)
	return e
}

func (e *Event) WorkflowMessage(workflowId string, msgType any, msgData any) *Event {
	e.Event = e.Event.Str("workflow", workflowId).Any("msg", msgType).Any("data", msgData)
	return e
}

func (e *Event) Node(id string) *Event {
	e.Event = e.Event.Str("node", id)
	return e
}

func (e *Event) Nodes(nodes []graph.Node) *Event {
	nodesOutput := ""
	switch len(nodes) {
	case 0:
		nodesOutput = "[]"
	case 1:
		nodesOutput = nodes[0].ID()
	default:
		nodesOutput = "["
		for _, node := range nodes {
			nodesOutput += node.ID() + ","
		}
		nodesOutput = nodesOutput[:len(nodesOutput)-1] + "]"
	}
	e.Event = e.Event.Str("nodes", nodesOutput)
	return e
}

func (e *Event) NodeInputOutput(nodeId string, input any, output any) *Event {
	e.Event = e.Event.Str("node", nodeId).Any("input", input).Any("output", output)
	return e
}

func Info() *Event {
	return &Event{log.Info()}
}

func Debug() *Event {
	return &Event{log.Debug()}
}

func Warn() *Event {
	return &Event{log.Warn()}
}

func Error() *Event {
	return &Event{log.Error()}
}
