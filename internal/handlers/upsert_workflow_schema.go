package handlers

import (
	"fmt"
	"io"
	"net/http"

	"ergo.services/ergo/gen"
	"github.com/gorilla/mux"
	"github.com/open-source-cloud/fuse/internal/repos"
	"github.com/open-source-cloud/fuse/internal/workflow"
)

type (
	// UpsertWorkflowSchemaHandler is the handler for the UpsertWorkflowSchema endpoint
	UpsertWorkflowSchemaHandler struct {
		Handler

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
func (h *UpsertWorkflowSchemaHandler) HandlePut(from gen.PID, w http.ResponseWriter, r *http.Request) error {
	vars := mux.Vars(r)

	schemaID, ok := vars["schemaID"]
	if !ok {
		return h.SendJSON(w, http.StatusBadRequest, Response{
			"message": "schemaID is required",
			"code":    BadRequest,
		})
	}

	h.Log().Info("upserting workflow schema", "from", from, "schemaID", schemaID)

	rawJSON, err := io.ReadAll(r.Body)
	if err != nil {
		msg := fmt.Sprintf("failed to read request body: %s", err)
		h.Log().Error(msg)
		return h.SendJSON(w, http.StatusBadRequest, Response{
			"message": msg,
			"code":    BadRequest,
		})
	}

	graph, err := h.graphFactory.NewGraphFromJSON(rawJSON)
	if err != nil {
		msg := fmt.Sprintf("failed to parse request body: %s", err)
		h.Log().Error(msg)
		return h.SendJSON(w, http.StatusBadRequest, Response{
			"message": msg,
			"code":    BadRequest,
		})
	}

	err = h.graphRepo.Save(graph)
	if err != nil {
		msg := fmt.Sprintf("failed to save graph: %s", err)
		h.Log().Error(msg)
		return h.SendJSON(w, http.StatusInternalServerError, Response{
			"message": msg,
			"code":    InternalServerError,
		})
	}

	h.Log().Info("upserted workflow schema", "from", from, "schemaID", schemaID, "workflowID", graph.ID())

	return h.SendJSON(w, http.StatusOK, Response{
		"schemaID": schemaID,
		"code":     "OK",
	})
}
