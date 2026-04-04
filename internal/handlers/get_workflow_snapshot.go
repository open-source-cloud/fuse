package handlers

import (
	"context"
	"net/http"

	"ergo.services/ergo/gen"
	internalworkflow "github.com/open-source-cloud/fuse/internal/workflow"

	"github.com/open-source-cloud/fuse/internal/repositories"
	"github.com/open-source-cloud/fuse/pkg/objectstore"
)

type (
	// GetWorkflowSnapshotHandler is the handler for GET /v1/workflows/{workflowID}/snapshot
	GetWorkflowSnapshotHandler struct {
		Handler
		workflowRepo repositories.WorkflowRepository
		journalRepo  repositories.JournalRepository
		store        objectstore.ObjectStore
	}
	// GetWorkflowSnapshotHandlerFactory is a factory for creating GetWorkflowSnapshotHandler actors
	GetWorkflowSnapshotHandlerFactory HandlerFactory[*GetWorkflowSnapshotHandler]
)

const (
	// GetWorkflowSnapshotHandlerName is the name of the GetWorkflowSnapshotHandler actor
	GetWorkflowSnapshotHandlerName = "get_workflow_snapshot_handler"
	// GetWorkflowSnapshotHandlerPoolName is the name of the GetWorkflowSnapshotHandler pool
	GetWorkflowSnapshotHandlerPoolName = "get_workflow_snapshot_handler_pool"
)

// NewGetWorkflowSnapshotHandlerFactory creates a new GetWorkflowSnapshotHandlerFactory
func NewGetWorkflowSnapshotHandlerFactory(
	workflowRepo repositories.WorkflowRepository,
	journalRepo repositories.JournalRepository,
	store objectstore.ObjectStore,
) *GetWorkflowSnapshotHandlerFactory {
	return &GetWorkflowSnapshotHandlerFactory{
		Factory: func() gen.ProcessBehavior {
			return &GetWorkflowSnapshotHandler{
				workflowRepo: workflowRepo,
				journalRepo:  journalRepo,
				store:        store,
			}
		},
	}
}

// HandleGet handles GET /v1/workflows/{workflowID}/snapshot
// @Summary Get workflow execution snapshot
// @Description Returns the execution snapshot for a workflow. If a persisted snapshot exists, it is returned directly. Otherwise, a live snapshot is built from the journal.
// @Tags workflows
// @Produce json
// @Param workflowID path string true "Workflow ID"
// @Success 200 {object} internalworkflow.ExecutionSnapshot
// @Failure 404 {object} dtos.NotFoundError
// @Router /v1/workflows/{workflowID}/snapshot [get]
func (h *GetWorkflowSnapshotHandler) HandleGet(_ gen.PID, w http.ResponseWriter, r *http.Request) error {
	workflowID, err := h.GetPathParam(r, "workflowID")
	if err != nil {
		return h.SendBadRequest(w, err, EmptyFields)
	}

	// Check if workflow exists
	wf, err := h.workflowRepo.Get(workflowID)
	if err != nil {
		return h.SendNotFound(w, "workflow not found", EmptyFields)
	}

	// Try to serve persisted snapshot first
	snapshotRef, _ := h.workflowRepo.GetSnapshotRef(workflowID)
	if snapshotRef != "" {
		data, getErr := h.store.Get(context.Background(), snapshotRef)
		if getErr == nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(data)
			return nil
		}
		// Fall through to live build if object store read fails
	}

	// Live fallback: build snapshot from journal entries
	entries, loadErr := h.journalRepo.LoadAll(workflowID)
	if loadErr != nil {
		return h.SendInternalError(w, loadErr)
	}

	snap := internalworkflow.BuildExecutionSnapshot(
		workflowID,
		wf.Graph().ID(),
		wf.State(),
		entries,
		wf.AggregatedOutputSnapshot(),
	)

	return h.SendJSON(w, http.StatusOK, snap)
}

