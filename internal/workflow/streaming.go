package workflow

import (
	"github.com/open-source-cloud/fuse/pkg/workflow"
)

// StreamEventType defines the type of workflow stream event
type StreamEventType string

const (
	// StreamEventWorkflowStarted indicates workflow has started
	StreamEventWorkflowStarted StreamEventType = "workflow:started"
	// StreamEventNodeExecuting indicates a node is executing
	StreamEventNodeExecuting StreamEventType = "node:executing"
	// StreamEventNodeResult indicates a node has produced a result
	StreamEventNodeResult StreamEventType = "node:result"
	// StreamEventWorkflowStateChanged indicates workflow state changed
	StreamEventWorkflowStateChanged StreamEventType = "workflow:state_changed"
	// StreamEventWorkflowCompleted indicates workflow completed successfully
	StreamEventWorkflowCompleted StreamEventType = "workflow:completed"
	// StreamEventWorkflowError indicates workflow encountered an error
	StreamEventWorkflowError StreamEventType = "workflow:error"
)

// StreamEvent represents an event emitted during workflow execution
type StreamEvent struct {
	Type      StreamEventType        `json:"type"`
	WorkflowID workflow.ID            `json:"workflow_id"`
	NodeID    string                 `json:"node_id,omitempty"`
	ThreadID  uint16                 `json:"thread_id,omitempty"`
	ExecID    workflow.ExecID         `json:"exec_id,omitempty"`
	State     State                  `json:"state,omitempty"`
	Data      map[string]any         `json:"data,omitempty"`
	Error     string                 `json:"error,omitempty"`
}

// StreamEmitter defines the interface for emitting workflow stream events
type StreamEmitter interface {
	Emit(event StreamEvent) error
}

// streamEmitter is the default implementation of StreamEmitter
type streamEmitter struct {
	callbacks []workflow.StreamCallback
}

// NewStreamEmitter creates a new stream emitter
func NewStreamEmitter() *streamEmitter {
	return &streamEmitter{
		callbacks: make([]workflow.StreamCallback, 0),
	}
}

// AddCallback adds a callback to receive stream events
func (e *streamEmitter) AddCallback(callback workflow.StreamCallback) {
	e.callbacks = append(e.callbacks, callback)
}

// Emit emits a stream event to all registered callbacks
func (e *streamEmitter) Emit(event StreamEvent) error {
	chunk := workflow.StreamChunk{
		Type: workflow.StreamChunkData,
		Data: map[string]any{
			"event_type":   string(event.Type),
			"workflow_id":  event.WorkflowID.String(),
			"node_id":      event.NodeID,
			"thread_id":    event.ThreadID,
			"exec_id":      event.ExecID.String(),
			"state":        string(event.State),
			"data":         event.Data,
			"error":        event.Error,
		},
	}

	for _, callback := range e.callbacks {
		if err := callback(chunk); err != nil {
			return err
		}
	}
	return nil
}

// EmitError emits an error event
func (e *streamEmitter) EmitError(event StreamEvent, err error) error {
	event.Error = err.Error()
	chunk := workflow.NewStreamChunkError(err)
	return e.Emit(event)
}
