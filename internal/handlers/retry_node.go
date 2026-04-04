package handlers

import (
	"net/http"

	"ergo.services/ergo/gen"
	"github.com/open-source-cloud/fuse/internal/dtos"
	"github.com/open-source-cloud/fuse/internal/messaging"
	"github.com/open-source-cloud/fuse/internal/repositories"
	"github.com/open-source-cloud/fuse/pkg/workflow"
)

type (
	// RetryNodeHandler handles POST /v1/workflows/{workflowID}/retry-node
	RetryNodeHandler struct {
		Handler
		workflowRepo repositories.WorkflowRepository
	}
	// RetryNodeHandlerFactory is a factory for creating RetryNodeHandler actors
	RetryNodeHandlerFactory HandlerFactory[*RetryNodeHandler]
)

const (
	// RetryNodeHandlerName is the name of the RetryNodeHandler actor
	RetryNodeHandlerName = "retry_node_handler"
	// RetryNodeHandlerPoolName is the name of the RetryNodeHandler pool
	RetryNodeHandlerPoolName = "retry_node_handler_pool"
)

// NewRetryNodeHandlerFactory creates a new RetryNodeHandlerFactory
func NewRetryNodeHandlerFactory(workflowRepo repositories.WorkflowRepository) *RetryNodeHandlerFactory {
	return &RetryNodeHandlerFactory{
		Factory: func() gen.ProcessBehavior {
			return &RetryNodeHandler{
				workflowRepo: workflowRepo,
			}
		},
	}
}

// HandlePost handles POST /v1/workflows/{workflowID}/retry-node
// @Summary Retry a specific failed node
// @Description Manually retries a specific failed node execution. The workflow must be in error state.
// @Tags workflows
// @Accept json
// @Produce json
// @Param workflowID path string true "Workflow ID"
// @Param request body dtos.RetryNodeRequest true "Execution ID of the failed node"
// @Success 202 {object} dtos.RetryNodeResponse
// @Failure 400 {object} dtos.BadRequestError
// @Failure 404 {object} dtos.NotFoundError
// @Failure 500 {object} dtos.InternalServerErrorResponse
// @Router /v1/workflows/{workflowID}/retry-node [post]
func (h *RetryNodeHandler) HandlePost(_ gen.PID, w http.ResponseWriter, r *http.Request) error {
	workflowID, err := h.GetPathParam(r, "workflowID")
	if err != nil {
		return h.SendBadRequest(w, err, EmptyFields)
	}

	var req dtos.RetryNodeRequest
	if err := h.BindJSON(w, r, &req); err != nil {
		return err
	}

	// Validate workflow exists and is in error state
	wf, getErr := h.workflowRepo.Get(workflowID)
	if getErr != nil {
		return h.SendNotFound(w, "workflow not found", EmptyFields)
	}
	if wf.State().String() != "error" {
		return h.SendBadRequest(w, nil, []string{"workflow must be in error state to retry a node"})
	}

	retryMsg := messaging.NewRetryNodeMessage(workflow.ID(workflowID), workflow.ExecID(req.ExecID))
	if err := h.Send(WorkflowSupervisorName, retryMsg); err != nil {
		return h.SendInternalError(w, err)
	}

	return h.SendJSON(w, http.StatusAccepted, dtos.RetryNodeResponse{
		WorkflowID: workflowID,
		ExecID:     req.ExecID,
		Status:     "accepted",
	})
}
