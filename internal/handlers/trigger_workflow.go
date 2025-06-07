// Package handlers HTTP Fiber handles
package handlers

import (
	"fmt"
	"net/http"

	"ergo.services/ergo/act"
	"ergo.services/ergo/gen"
	"github.com/open-source-cloud/fuse/internal/messaging"
	"github.com/open-source-cloud/fuse/internal/workflow"
)

type (
	// TriggerWorkflowRequest is the request body for the TriggerWorkflowHandler
	TriggerWorkflowRequest struct {
		SchemaID string `json:"schemaId,omitempty"`
	}
	// TriggerWorkflowHandler is the handler for the TriggerWorkflow endpoint
	TriggerWorkflowHandler struct {
		act.WebWorker
	}
)

// TriggerWorkflowHandlerName is the name of the TriggerWorkflowHandler actor
const TriggerWorkflowHandlerName = "trigger_workflow_handler"

// TriggerWorkflowHandlerFactory is a factory for creating TriggerWorkflowHandler actors
type TriggerWorkflowHandlerFactory HandlerFactory[*TriggerWorkflowHandler]

// NewTriggerWorkflowHandlerFactory creates a new TriggerWorkflowHandlerFactory
func NewTriggerWorkflowHandlerFactory() *TriggerWorkflowHandlerFactory {
	return &TriggerWorkflowHandlerFactory{
		Factory: func() gen.ProcessBehavior {
			return &TriggerWorkflowHandler{}
		},
	}
}

// HandlePost handles the http TriggerWorkflow endpoint
// POST /v1/workflows/{workflowID}/trigger
func (h *TriggerWorkflowHandler) HandlePost(w http.ResponseWriter, r *http.Request) error {
	var req TriggerWorkflowRequest
	if err := BindJSON(w, r, req); err != nil {
		return SendJSON(w, http.StatusBadRequest, Response{
			"message": fmt.Sprintf("invalid request: %s", err),
			"code":    BadRequest,
		})
	}

	if req.SchemaID == "" {
		return SendJSON(w, http.StatusBadRequest, Response{
			"message": "schemaId is required",
			"code":    BadRequest,
		})
	}

	senderID := workflow.NewID()

	err := h.Send(senderID, messaging.NewTriggerWorkflowMessage(req.SchemaID))
	if err != nil {
		return SendJSON(w, http.StatusInternalServerError, Response{
			"message": fmt.Sprintf("failed to send message: %s", err),
			"code":    InternalServerError,
		})
	}

	return SendJSON(w, http.StatusOK, Response{
		"workflowID": senderID,
		"code":       "OK",
	})
}
