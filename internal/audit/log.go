// Package audit and logging tools
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

// Workflow log a workflow ID
func (e *Event) Workflow(id string) *Event {
	e.Event = e.Str("workflow", id)
	return e
}

// WorkflowState log a workflow ID and state
func (e *Event) WorkflowState(id string, state any) *Event {
	e.Event = e.Event.Str("workflow", id).Any("state", state)
	return e
}

// WorkflowMessage log a workflow ID and message
func (e *Event) WorkflowMessage(workflowID string, msgType any, msgData any) *Event {
	e.Event = e.Event.Str("workflow", workflowID).Any("msg", msgType).Any("data", msgData)
	return e
}

// Node log a node
func (e *Event) Node(id string) *Event {
	e.Event = e.Str("node", id)
	return e
}

// Nodes log a node array
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
	e.Event = e.Str("nodes", nodesOutput)
	return e
}

// NodeInputOutput log a node and it's input/output
func (e *Event) NodeInputOutput(nodeID string, input any, output any) *Event {
	e.Event = e.Event.Str("node", nodeID).Any("input", input).Any("output", output)
	return e
}

// Trace trace level logging
func Trace() *Event {
	return &Event{log.Trace()}
}

// Debug debug level logging
func Debug() *Event {
	return &Event{log.Debug()}
}

// Info info level logging
func Info() *Event {
	return &Event{log.Info()}
}

// Warn warning level logging
func Warn() *Event {
	return &Event{log.Warn()}
}

// Error error level logging
func Error() *Event {
	return &Event{log.Error()}
}

// Fatal fatal level logging
func Fatal() *Event {
	return &Event{log.Fatal()}
}

// Panic panic level logging
func Panic() *Event {
	return &Event{log.Panic()}
}
