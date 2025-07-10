package handlers

import (
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/go-playground/validator/v10"
	"github.com/open-source-cloud/fuse/internal/services"
	"github.com/open-source-cloud/fuse/internal/workflow"

	"ergo.services/ergo/gen"
	"github.com/open-source-cloud/fuse/internal/repositories"
)

type (
	// UpsertWorkflowSchemaHandler is the handler for the UpsertWorkflowSchema endpoint
	UpsertWorkflowSchemaHandler struct {
		Handler
		graphService services.GraphService
	}
	// UpsertWorkflowSchemaHandlerFactory is a factory for creating UpsertWorkflowSchemaHandler actors
	UpsertWorkflowSchemaHandlerFactory HandlerFactory[*UpsertWorkflowSchemaHandler]
)

const (
	// UpsertWorkflowSchemaHandlerName is the name of the UpsertWorkflowSchemaHandler actor
	UpsertWorkflowSchemaHandlerName = "upsert_workflow_schema_handler"
	// UpsertWorkflowSchemaHandlerPoolName is the name of the UpsertWorkflowSchemaHandler pool
	UpsertWorkflowSchemaHandlerPoolName = "upsert_workflow_schema_handler_pool"
)

// NewUpsertWorkflowSchemaHandlerFactory creates a new NewUpsertWorkflowSchemaHandlerFactory
func NewUpsertWorkflowSchemaHandlerFactory(graphService services.GraphService) *UpsertWorkflowSchemaHandlerFactory {
	return &UpsertWorkflowSchemaHandlerFactory{
		Factory: func() gen.ProcessBehavior {
			return &UpsertWorkflowSchemaHandler{
				graphService: graphService,
			}
		},
	}
}

// HandlePut handles the UpsertWorkflowSchema http endpoint -- PUT /v1/schemas/{schemaID}
func (h *UpsertWorkflowSchemaHandler) HandlePut(from gen.PID, w http.ResponseWriter, r *http.Request) error {
	h.Log().Info("received upsert workflow schema request from: %v remoteAddr: %s", from, r.RemoteAddr)

	schemaID, err := h.GetPathParam(r, "schemaID")
	if err != nil {
		return h.SendBadRequest(w, err, []string{"schemaID is required"})
	}

	h.Log().Info("upserting workflow schema", "from", from, "schemaID", schemaID)

	rawJSON, err := io.ReadAll(r.Body)
	if err != nil {
		return h.SendBadRequest(w, err, EmptyFields)
	}

	schema, err := workflow.NewGraphSchemaFromJSON(rawJSON)
	if err != nil {
		return h.SendBadRequest(w, err, EmptyFields)
	}

	_, err = h.graphService.Upsert(schemaID, schema)
	if err != nil {
		if errors.As(err, &validator.ValidationErrors{}) {
			return h.SendValidationErr(w, err)
		}
		if errors.Is(err, repositories.ErrGraphNotFound) {
			return h.SendNotFound(w, fmt.Sprintf("schema %s not found", schemaID), EmptyFields)
		}
		return h.SendInternalError(w, err)
	}

	h.Log().Info("upserted workflow schema", "from", from, "schemaID", schemaID)

	return h.SendJSON(w, http.StatusOK, Response{
		"schemaId": schemaID,
	})
}

// HandleGet returns the graph schema to the client -- GET /schemas/:schemaId
func (h *UpsertWorkflowSchemaHandler) HandleGet(from gen.PID, w http.ResponseWriter, r *http.Request) error {
	h.Log().Info("received get workflow schema request", "from", from, "remoteAddr", r.RemoteAddr)

	schemaID, err := h.GetPathParam(r, "schemaID")
	if err != nil {
		return h.SendJSON(w, http.StatusBadRequest, Response{})
	}

	h.Log().Info("fetching workflow schema", "from", from, "schemaID", schemaID)

	graph, err := h.graphService.FindByID(schemaID)
	if err != nil {
		if errors.As(err, &validator.ValidationErrors{}) {
			return h.SendValidationErr(w, err)
		}
		return h.SendInternalError(w, err)
	}

	h.Log().Info("schema found", "from", from, "schemaID", schemaID)

	return h.SendJSON(w, http.StatusOK, graph.Schema())
}
