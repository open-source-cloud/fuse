package handlers

import (
	"net/http"

	"ergo.services/ergo/gen"
	"github.com/google/uuid"
	"github.com/open-source-cloud/fuse/internal/dtos"
	internalworkflow "github.com/open-source-cloud/fuse/internal/workflow"
	"github.com/open-source-cloud/fuse/internal/messaging"
	"github.com/open-source-cloud/fuse/internal/repositories"
	"github.com/open-source-cloud/fuse/pkg/workflow"
)

type (
	// RetryWorkflowHandler handles POST /v1/workflows/{workflowID}/retry
	RetryWorkflowHandler struct {
		Handler
		workflowRepo repositories.WorkflowRepository
		journalRepo  repositories.JournalRepository
	}
	// RetryWorkflowHandlerFactory is a factory for creating RetryWorkflowHandler actors
	RetryWorkflowHandlerFactory HandlerFactory[*RetryWorkflowHandler]
)

const (
	// RetryWorkflowHandlerName is the name of the RetryWorkflowHandler actor
	RetryWorkflowHandlerName = "retry_workflow_handler"
	// RetryWorkflowHandlerPoolName is the name of the RetryWorkflowHandler pool
	RetryWorkflowHandlerPoolName = "retry_workflow_handler_pool"
)

// NewRetryWorkflowHandlerFactory creates a new RetryWorkflowHandlerFactory
func NewRetryWorkflowHandlerFactory(
	workflowRepo repositories.WorkflowRepository,
	journalRepo repositories.JournalRepository,
) *RetryWorkflowHandlerFactory {
	return &RetryWorkflowHandlerFactory{
		Factory: func() gen.ProcessBehavior {
			return &RetryWorkflowHandler{
				workflowRepo: workflowRepo,
				journalRepo:  journalRepo,
			}
		},
	}
}

// HandlePost handles POST /v1/workflows/{workflowID}/retry
// @Summary Retry a workflow
// @Description Retry a workflow from scratch (new workflow) or from the last failed node.
// @Tags workflows
// @Accept json
// @Produce json
// @Param workflowID path string true "Workflow ID"
// @Param request body dtos.RetryWorkflowRequest false "Retry strategy and optional exec ID"
// @Success 202 {object} dtos.RetryWorkflowResponse
// @Failure 400 {object} dtos.BadRequestError
// @Failure 404 {object} dtos.NotFoundError
// @Failure 500 {object} dtos.InternalServerErrorResponse
// @Router /v1/workflows/{workflowID}/retry [post]
func (h *RetryWorkflowHandler) HandlePost(_ gen.PID, w http.ResponseWriter, r *http.Request) error {
	workflowID, err := h.GetPathParam(r, "workflowID")
	if err != nil {
		return h.SendBadRequest(w, err, EmptyFields)
	}

	var req dtos.RetryWorkflowRequest
	_ = h.BindJSON(w, r, &req) // all fields optional

	wf, getErr := h.workflowRepo.Get(workflowID)
	if getErr != nil {
		return h.SendNotFound(w, "workflow not found", EmptyFields)
	}

	strategy := req.Strategy
	if strategy == "" {
		strategy = "from-failed"
	}

	switch strategy {
	case "from-scratch":
		return h.retryFromScratch(w, wf)
	case "from-failed":
		return h.retryFromFailed(w, workflowID, wf, req.ExecID)
	default:
		return h.SendBadRequest(w, nil, []string{"strategy must be 'from-scratch' or 'from-failed'"})
	}
}

func (h *RetryWorkflowHandler) retryFromScratch(w http.ResponseWriter, wf *internalworkflow.Workflow) error {
	newWfID := workflow.ID(uuid.New().String())
	schemaID := wf.Graph().ID()

	triggerMsg := messaging.NewTriggerWorkflowMessage(schemaID, newWfID)
	if err := h.Send(WorkflowSupervisorName, triggerMsg); err != nil {
		return h.SendInternalError(w, err)
	}

	return h.SendJSON(w, http.StatusAccepted, dtos.RetryWorkflowResponse{
		OriginalWorkflowID: wf.ID().String(),
		NewWorkflowID:      newWfID.String(),
		Status:             "accepted",
	})
}

func (h *RetryWorkflowHandler) retryFromFailed(w http.ResponseWriter, workflowID string, wf *internalworkflow.Workflow, execID string) error {
	if wf.State() != internalworkflow.StateError {
		return h.SendBadRequest(w, nil, []string{"workflow must be in error state for from-failed retry"})
	}

	// If no execID provided, auto-discover the last failed exec
	if execID == "" {
		failed, findErr := h.journalRepo.FindFailed(workflowID)
		if findErr != nil {
			return h.SendInternalError(w, findErr)
		}
		if len(failed) == 0 {
			return h.SendBadRequest(w, nil, []string{"no failed executions found in workflow"})
		}
		execID = failed[len(failed)-1].ExecID
	}

	retryMsg := messaging.NewRetryNodeMessage(workflow.ID(workflowID), workflow.ExecID(execID))
	if err := h.Send(WorkflowSupervisorName, retryMsg); err != nil {
		return h.SendInternalError(w, err)
	}

	return h.SendJSON(w, http.StatusAccepted, dtos.RetryWorkflowResponse{
		OriginalWorkflowID: workflowID,
		Status:             "accepted",
	})
}
