package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"ergo.services/ergo/gen"
	"github.com/open-source-cloud/fuse/internal/actors/actornames"
	"github.com/open-source-cloud/fuse/internal/messaging"
	"github.com/open-source-cloud/fuse/pkg/workflow"
)

type (
	// StreamWorkflowHandler is the handler for streaming workflow execution via SSE
	StreamWorkflowHandler struct {
		Handler
	}
	// StreamWorkflowHandlerFactory is a factory for creating StreamWorkflowHandler actors
	StreamWorkflowHandlerFactory HandlerFactory[*StreamWorkflowHandler]
)

const (
	// StreamWorkflowHandlerName is the name of the StreamWorkflowHandler actor
	StreamWorkflowHandlerName = "stream_workflow_handler"
	// StreamWorkflowHandlerPoolName is the name of the StreamWorkflowHandler pool
	StreamWorkflowHandlerPoolName = "stream_workflow_handler_pool"
)

// NewStreamWorkflowHandlerFactory creates a new StreamWorkflowHandlerFactory
func NewStreamWorkflowHandlerFactory() *StreamWorkflowHandlerFactory {
	return &StreamWorkflowHandlerFactory{
		Factory: func() gen.ProcessBehavior {
			return &StreamWorkflowHandler{}
		},
	}
}

// HandleGet handles the HTTP GET request for streaming workflow execution (GET /v1/workflows/{workflowID}/stream)
func (h *StreamWorkflowHandler) HandleGet(from gen.PID, w http.ResponseWriter, r *http.Request) error {
	h.Log().Info("received stream workflow request from: %v remoteAddr: %s", from, r.RemoteAddr)

	workflowIDStr, err := h.GetPathParam(r, "workflowID")
	if err != nil {
		return h.SendBadRequest(w, err, []string{"workflowID"})
	}

	workflowID := workflow.ID(workflowIDStr)

	// Set up SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	// Flush headers to establish SSE connection
	if flusher, ok := w.(http.Flusher); ok {
		flusher.Flush()
	}

	// Create a channel to receive stream chunks
	chunkChan := make(chan workflow.StreamChunk, 10)
	doneChan := make(chan bool)

	// Create callback that sends chunks to channel
	callback := func(chunk workflow.StreamChunk) error {
		select {
		case chunkChan <- chunk:
			return nil
		case <-doneChan:
			return fmt.Errorf("stream closed")
		default:
			// Channel full, drop chunk (or could block)
			return nil
		}
	}

	// Register callback with WorkflowHandler
	workflowHandlerName := actornames.WorkflowHandlerName(workflowID)
	streamMsg := messaging.NewStreamWorkflowMessage(workflowID, callback)
	if err := h.Send(workflowHandlerName, streamMsg); err != nil {
		h.Log().Error("failed to register stream callback: %s", err)
		return h.SendInternalError(w, err)
	}

	// Send initial connection event
	h.writeSSE(w, "connected", map[string]any{
		"workflow_id": workflowID.String(),
		"message":     "Stream connected",
	})

	// Handle client disconnection
	ctx := r.Context()

	// Start goroutine to handle streaming
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				// Client disconnected
				h.Log().Info("client disconnected for workflow %s", workflowID)
				close(doneChan)
				// Unregister callback
				unregisterMsg := messaging.NewStreamWorkflowMessage(workflowID, nil)
				_ = h.Send(workflowHandlerName, unregisterMsg)
				return
			case chunk, ok := <-chunkChan:
				if !ok {
					return
				}
				h.writeSSEChunk(w, chunk)
				if flusher, ok := w.(http.Flusher); ok {
					flusher.Flush()
				}
			case <-ticker.C:
				// Send keepalive ping
				h.writeSSE(w, "ping", map[string]any{"time": time.Now().Unix()})
				if flusher, ok := w.(http.Flusher); ok {
					flusher.Flush()
				}
			}
		}
	}()

	// Wait for context cancellation (client disconnect)
	<-ctx.Done()

	h.Log().Info("stream closed for workflow %s", workflowID)
	return nil
}

// writeSSE writes an SSE-formatted message
func (h *StreamWorkflowHandler) writeSSE(w http.ResponseWriter, eventType string, data map[string]any) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		h.Log().Error("failed to marshal SSE data: %s", err)
		return
	}

	_, _ = fmt.Fprintf(w, "event: %s\n", eventType)
	_, _ = fmt.Fprintf(w, "data: %s\n\n", jsonData)
}

// writeSSEChunk writes a stream chunk as SSE
func (h *StreamWorkflowHandler) writeSSEChunk(w http.ResponseWriter, chunk workflow.StreamChunk) {
	eventType := "chunk"
	if chunk.Type == workflow.StreamChunkError {
		eventType = "error"
	} else if chunk.Type == workflow.StreamChunkDone {
		eventType = "done"
	}

	data := map[string]any{
		"type": string(chunk.Type),
	}
	if chunk.Data != nil {
		data["data"] = chunk.Data
	}
	if chunk.Error != "" {
		data["error"] = chunk.Error
	}

	h.writeSSE(w, eventType, data)
}
