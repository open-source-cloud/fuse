// Package handlers HTTP Fiber handles
package handlers

import (
	"fmt"
	"net/http"

	"github.com/open-source-cloud/fuse/pkg/workflow"

	"ergo.services/ergo/gen"
	"github.com/open-source-cloud/fuse/internal/dtos"
	"github.com/open-source-cloud/fuse/internal/messaging"
)

type (
	// TriggerWorkflowHandler is the handler for the TriggerWorkflow endpoint
	TriggerWorkflowHandler struct {
		Handler
	}
	// TriggerWorkflowHandlerFactory is a factory for creating TriggerWorkflowHandler actors
	TriggerWorkflowHandlerFactory HandlerFactory[*TriggerWorkflowHandler]
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

// HandlePost handles the http TriggerWorkflow endpoint (POST /v1/workflows/trigger)
// @Summary Trigger workflow execution
// @Description Triggers a new workflow instance from a schema
// @Tags workflows
// @Accept json
// @Produce json
// @Param request body dtos.TriggerWorkflowRequest true "Trigger Request"
// @Success 200 {object} dtos.TriggerWorkflowResponse
// @Failure 400 {object} dtos.BadRequestError
// @Failure 500 {object} dtos.InternalServerErrorResponse
// @Router /v1/workflows/trigger [post]
func (h *TriggerWorkflowHandler) HandlePost(from gen.PID, w http.ResponseWriter, r *http.Request) error {
	h.Log().Info("received trigger workflow request from: %v remoteAddr: %s", from, r.RemoteAddr)

	var req dtos.TriggerWorkflowRequest
	if err := h.BindJSON(w, r, &req); err != nil {
		return h.SendBadRequest(w, err, []string{"body"})
	}

	if req.SchemaID == "" {
		return h.SendBadRequest(w, fmt.Errorf("schemaID is required"), []string{"schemaID"})
	}

	workflowID := workflow.NewID()
	if err := h.Send(WorkflowSupervisorName, messaging.NewTriggerWorkflowMessage(req.SchemaID, workflowID)); err != nil {
		return h.SendInternalError(w, err)
	}

	return h.SendJSON(w, http.StatusOK, dtos.TriggerWorkflowResponse{
		SchemaID:   req.SchemaID,
		WorkflowID: workflowID.String(),
		Code:       "OK",
	})
}
