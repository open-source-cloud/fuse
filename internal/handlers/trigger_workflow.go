// Package handlers HTTP Fiber handles
package handlers

import (
	"fmt"
	"net/http"

	"ergo.services/ergo/act"
	"ergo.services/ergo/gen"
	"github.com/gorilla/mux"
	"github.com/open-source-cloud/fuse/internal/messaging"
	"github.com/open-source-cloud/fuse/internal/workflow"
)

type (
	// TriggerWorkflowHandler is the handler for the TriggerWorkflow endpoint
	TriggerWorkflowHandler struct {
		act.WebWorker
	}
)

// TriggerWorkflowHandlerName is the name of the TriggerWorkflowHandler actor
const TriggerWorkflowHandlerName = "trigger_workflow_handler"
const TriggerWorkflowHandlerPoolName = "trigger_workflow_handler_pool"

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
// POST /v1/workflows/{schemaID}/trigger
func (h *TriggerWorkflowHandler) HandlePost(from gen.PID, w http.ResponseWriter, r *http.Request) error {
	vars := mux.Vars(r)

	schemaID, ok := vars["schemaID"]
	if !ok {
		return SendJSON(w, http.StatusBadRequest, Response{
			"message": "schemaId is required",
			"code":    BadRequest,
		})
	}

	workflowID := workflow.NewID()

	err := h.Send(workflowID, messaging.NewTriggerWorkflowMessage(schemaID))
	if err != nil {
		return SendJSON(w, http.StatusInternalServerError, Response{
			"message": fmt.Sprintf("failed to send message: %s", err),
			"code":    InternalServerError,
		})
	}

	return SendJSON(w, http.StatusOK, Response{
		"schemaID": schemaID,
		"code":     "OK",
	})
}
