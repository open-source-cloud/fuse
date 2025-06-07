package handlers

import (
	"fmt"
	"io"
	"net/http"

	"ergo.services/ergo/act"
	"ergo.services/ergo/gen"
	"github.com/open-source-cloud/fuse/internal/repos"
	"github.com/open-source-cloud/fuse/internal/workflow"
)

type (
	// UpsertWorkflowSchemaHandler is the handler for the UpsertWorkflowSchema endpoint
	UpsertWorkflowSchemaHandler struct {
		act.WebWorker

		graphFactory *workflow.GraphFactory
		graphRepo    repos.GraphRepo
	}
)

// UpsertWorkflowSchemaHandlerName is the name of the UpsertWorkflowSchemaHandler actor
const UpsertWorkflowSchemaHandlerName = "upsert_workflow_schema_handler"
const UpsertWorkflowSchemaHandlerPoolName = "upsert_workflow_schema_handler_pool"

// UpsertWorkflowSchemaHandlerFactory is a factory for creating UpsertWorkflowSchemaHandler actors
type UpsertWorkflowSchemaHandlerFactory HandlerFactory[*UpsertWorkflowSchemaHandler]

// NewUpsertWorkflowSchemaHandler creates a new UpsertWorkflowSchemaHandler
func NewUpsertWorkflowSchemaHandlerFactory(graphFactory *workflow.GraphFactory, graphRepo repos.GraphRepo) *UpsertWorkflowSchemaHandlerFactory {
	return &UpsertWorkflowSchemaHandlerFactory{
		Factory: func() gen.ProcessBehavior {
			return &UpsertWorkflowSchemaHandler{
				graphFactory: graphFactory,
				graphRepo:    graphRepo,
			}
		},
	}
}

func (h *UpsertWorkflowSchemaHandler) Init(args ...any) error {
	h.Log().Info("starting upsert workflow schema handler")
	return nil
}

// UpsertWorkflowSchema handles the UpsertWorkflowSchema http endpoint
// PUT /v1/schemas/{schemaID}
func (h *UpsertWorkflowSchemaHandler) HandlePut(w http.ResponseWriter, r *http.Request) error {
	rawJSON, err := io.ReadAll(r.Body)
	if err != nil {
		return SendJSON(w, http.StatusBadRequest, Response{
			"message": fmt.Sprintf("failed to read request body: %s", err),
			"code":    BadRequest,
		})
	}

	graph, err := h.graphFactory.NewGraphFromJSON(rawJSON)
	if err != nil {
		return SendJSON(w, http.StatusBadRequest, Response{
			"message": fmt.Sprintf("failed to parse request body: %s", err),
			"code":    BadRequest,
		})
	}

	err = h.graphRepo.Save(graph)
	if err != nil {
		return SendJSON(w, http.StatusInternalServerError, Response{
			"message": fmt.Sprintf("failed to save graph: %s", err),
			"code":    InternalServerError,
		})
	}

	return SendJSON(w, http.StatusOK, Response{
		"workflowID": graph.ID,
	})
}
