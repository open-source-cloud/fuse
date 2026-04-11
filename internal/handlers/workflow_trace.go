package handlers

import (
	"net/http"

	"ergo.services/ergo/gen"
	"github.com/open-source-cloud/fuse/internal/repositories"
)

type (
	// WorkflowTraceHandler is the handler for the workflow trace endpoint
	WorkflowTraceHandler struct {
		Handler
		traceRepo repositories.TraceRepository
	}
	// WorkflowTraceHandlerFactory is a factory for creating WorkflowTraceHandler actors
	WorkflowTraceHandlerFactory HandlerFactory[*WorkflowTraceHandler]
)

const (
	// WorkflowTraceHandlerName is the name of the WorkflowTraceHandler actor
	WorkflowTraceHandlerName = "workflow_trace_handler"
	// WorkflowTraceHandlerPoolName is the name of the WorkflowTraceHandler pool
	WorkflowTraceHandlerPoolName = "workflow_trace_handler_pool"
)

// NewWorkflowTraceHandlerFactory creates a new WorkflowTraceHandlerFactory
func NewWorkflowTraceHandlerFactory(traceRepo repositories.TraceRepository) *WorkflowTraceHandlerFactory {
	return &WorkflowTraceHandlerFactory{
		Factory: func() gen.ProcessBehavior {
			return &WorkflowTraceHandler{
				traceRepo: traceRepo,
			}
		},
	}
}

// HandleGet handles the get workflow trace endpoint (GET /v1/workflows/{workflowID}/trace)
// @Summary Get workflow execution trace
// @Description Returns the full execution trace for a workflow
// @Tags workflows
// @Produce json
// @Param workflowID path string true "Workflow ID"
// @Success 200 {object} workflow.ExecutionTrace
// @Failure 400 {object} dtos.BadRequestError
// @Failure 404 {object} dtos.NotFoundError
// @Router /v1/workflows/{workflowID}/trace [get]
func (h *WorkflowTraceHandler) HandleGet(_ gen.PID, w http.ResponseWriter, r *http.Request) error {
	workflowID, err := h.GetPathParam(r, "workflowID")
	if err != nil {
		return h.SendBadRequest(w, err, EmptyFields)
	}

	trace, err := h.traceRepo.FindByWorkflowID(workflowID)
	if err != nil {
		return h.SendNotFound(w, "trace not found", EmptyFields)
	}

	return h.SendJSON(w, http.StatusOK, trace)
}
