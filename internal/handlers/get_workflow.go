package handlers

import (
	"net/http"

	"ergo.services/ergo/gen"
	"github.com/open-source-cloud/fuse/internal/dtos"
	"github.com/open-source-cloud/fuse/internal/repositories"
)

type (
	// GetWorkflowHandler is the handler for the get workflow endpoint
	GetWorkflowHandler struct {
		Handler
		workflowRepo repositories.WorkflowRepository
	}
	// GetWorkflowHandlerFactory is a factory for creating GetWorkflowHandler actors
	GetWorkflowHandlerFactory HandlerFactory[*GetWorkflowHandler]
)

const (
	// GetWorkflowHandlerName is the name of the GetWorkflowHandler actor
	GetWorkflowHandlerName = "get_workflow_handler"
	// GetWorkflowHandlerPoolName is the name of the GetWorkflowHandler pool
	GetWorkflowHandlerPoolName = "get_workflow_handler_pool"
)

// NewGetWorkflowHandlerFactory creates a new GetWorkflowHandlerFactory
func NewGetWorkflowHandlerFactory(workflowRepo repositories.WorkflowRepository) *GetWorkflowHandlerFactory {
	return &GetWorkflowHandlerFactory{
		Factory: func() gen.ProcessBehavior {
			return &GetWorkflowHandler{
				workflowRepo: workflowRepo,
			}
		},
	}
}

// HandleGet handles the get workflow endpoint (GET /v1/workflows/{workflowID})
// @Summary Get workflow status
// @Description Returns the current status of a workflow
// @Tags workflows
// @Produce json
// @Param workflowID path string true "Workflow ID"
// @Success 200 {object} dtos.GetWorkflowResponse
// @Failure 400 {object} dtos.BadRequestError
// @Failure 404 {object} dtos.NotFoundError
// @Router /v1/workflows/{workflowID} [get]
func (h *GetWorkflowHandler) HandleGet(_ gen.PID, w http.ResponseWriter, r *http.Request) error {
	workflowID, err := h.GetPathParam(r, "workflowID")
	if err != nil {
		return h.SendBadRequest(w, err, EmptyFields)
	}

	wf, err := h.workflowRepo.Get(workflowID)
	if err != nil {
		return h.SendNotFound(w, "workflow not found", EmptyFields)
	}

	return h.SendJSON(w, http.StatusOK, dtos.GetWorkflowResponse{
		WorkflowID: workflowID,
		Status:     wf.State().String(),
	})
}
