// Package handlers HTTP Fiber handles
package handlers

import (
	"fmt"
	"github.com/open-source-cloud/fuse/internal/workflow"
	"net/http"

	"ergo.services/ergo/gen"
	"github.com/open-source-cloud/fuse/internal/messaging"
)

type (
	// TriggerWorkflowHandler is the handler for the TriggerWorkflow endpoint
	TriggerWorkflowHandler struct {
		Handler
	}
	// TriggerWorkflowHandlerFactory is a factory for creating TriggerWorkflowHandler actors
	TriggerWorkflowHandlerFactory HandlerFactory[*TriggerWorkflowHandler]

	// TriggerWorkflowRequest represents the data structure for this request
	TriggerWorkflowRequest struct {
		SchemaID string `json:"schemaID"`
	}
)

const (
	// TriggerWorkflowHandlerName is the name of the TriggerWorkflowHandler actor
	TriggerWorkflowHandlerName = "trigger_workflow_handler"
	// TriggerWorkflowHandlerPoolName is the name of the TriggerWorkflowHandler pool
	TriggerWorkflowHandlerPoolName = "trigger_workflow_handler_pool"
	// WorkflowSupervisorName is the name of the WorkflowSupervisor actor
	WorkflowSupervisorName = "workflow_sup"
)

// NewTriggerWorkflowHandlerFactory creates a new TriggerWorkflowHandlerFactory
func NewTriggerWorkflowHandlerFactory() *TriggerWorkflowHandlerFactory {
	return &TriggerWorkflowHandlerFactory{
		Factory: func() gen.ProcessBehavior {
			return &TriggerWorkflowHandler{}
		},
	}
}

// HandlePost handles the http TriggerWorkflow endpoint (POST /v1/workflows/{schemaID}/trigger)
func (h *TriggerWorkflowHandler) HandlePost(from gen.PID, w http.ResponseWriter, r *http.Request) error {
	h.Log().Info("received trigger workflow request", "from", from, "remoteAddr", r.RemoteAddr)

	var req TriggerWorkflowRequest
	if err := h.BindJSON(w, r, &req); err != nil {
		return h.SendJSON(w, http.StatusBadRequest, Response{
			"message": fmt.Sprintf("invalid request: %s", err),
			"code":    BadRequest,
		})
	}

	if req.SchemaID == "" {
		return h.SendJSON(w, http.StatusBadRequest, Response{
			"message": "schemaId is required",
			"code":    BadRequest,
		})
	}

	workflowID := workflow.NewID()
	if err := h.Send(WorkflowSupervisorName, messaging.NewTriggerWorkflowMessage(req.SchemaID, workflowID)); err != nil {
		return h.SendJSON(w, http.StatusInternalServerError, Response{
			"message": fmt.Sprintf("failed to send message: %s", err),
			"code":    InternalServerError,
		})
	}

	return h.SendJSON(w, http.StatusOK, Response{
		"schemaId":   req.SchemaID,
		"workflowId": workflowID,
		"code":        "OK",
	})
}
