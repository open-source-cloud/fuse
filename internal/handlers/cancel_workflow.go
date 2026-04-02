package handlers

import (
	"net/http"
	"time"

	"ergo.services/ergo/gen"
	"github.com/open-source-cloud/fuse/internal/dtos"
	"github.com/open-source-cloud/fuse/internal/messaging"
	"github.com/open-source-cloud/fuse/pkg/workflow"
)

type (
	// CancelWorkflowHandler is the handler for the cancel workflow endpoint
	CancelWorkflowHandler struct {
		Handler
	}
	// CancelWorkflowHandlerFactory is a factory for creating CancelWorkflowHandler actors
	CancelWorkflowHandlerFactory HandlerFactory[*CancelWorkflowHandler]
)

const (
	// CancelWorkflowHandlerName is the name of the CancelWorkflowHandler actor
	CancelWorkflowHandlerName = "cancel_workflow_handler"
	// CancelWorkflowHandlerPoolName is the name of the CancelWorkflowHandler pool
	CancelWorkflowHandlerPoolName = "cancel_workflow_handler_pool"
)

// NewCancelWorkflowHandlerFactory creates a new CancelWorkflowHandlerFactory
func NewCancelWorkflowHandlerFactory() *CancelWorkflowHandlerFactory {
	return &CancelWorkflowHandlerFactory{
		Factory: func() gen.ProcessBehavior {
			return &CancelWorkflowHandler{}
		},
	}
}

// HandlePost handles the cancel workflow endpoint (POST /v1/workflows/{workflowID}/cancel)
func (h *CancelWorkflowHandler) HandlePost(_ gen.PID, w http.ResponseWriter, r *http.Request) error {
	workflowID, err := h.GetPathParam(r, "workflowID")
	if err != nil {
		return h.SendBadRequest(w, err, EmptyFields)
	}

	var req dtos.CancelWorkflowRequest
	_ = h.BindJSON(w, r, &req) // reason is optional

	cancelMsg := messaging.NewCancelWorkflowMessage(workflow.ID(workflowID), req.Reason)
	if err := h.Send(WorkflowSupervisorName, cancelMsg); err != nil {
		return h.SendInternalError(w, err)
	}

	return h.SendJSON(w, http.StatusOK, dtos.CancelWorkflowResponse{
		WorkflowID:  workflowID,
		Status:      "cancelled",
		CancelledAt: time.Now().Format(time.RFC3339),
	})
}
