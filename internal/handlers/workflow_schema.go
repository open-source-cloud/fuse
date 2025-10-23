package handlers

import (
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/go-playground/validator/v10"
	"github.com/open-source-cloud/fuse/internal/dtos"
	"github.com/open-source-cloud/fuse/internal/services"
	"github.com/open-source-cloud/fuse/internal/workflow"

	"ergo.services/ergo/gen"
	"github.com/open-source-cloud/fuse/internal/repositories"
)

type (
	// WorkflowSchemaHandler is the handler for the WorkflowSchemaHandler endpoint
	WorkflowSchemaHandler struct {
		Handler
		graphService services.GraphService
	}
	// WorkflowSchemaHandlerFactory is a factory for creating WorkflowSchemaHandler actors
	WorkflowSchemaHandlerFactory HandlerFactory[*WorkflowSchemaHandler]
)

const (
	// WorkflowSchemaHandlerName is the name of the WorkflowSchemaHandler actor
	WorkflowSchemaHandlerName = "workflow_schema_handler"
	// WorkflowSchemaHandlerPoolName is the name of the WorkflowSchemaHandler pool
	WorkflowSchemaHandlerPoolName = "workflow_schema_handler_pool"
)

// NewWorkflowSchemaHandlerFactory creates a new WorkflowSchemaHandlerFactory
func NewWorkflowSchemaHandlerFactory(graphService services.GraphService) *WorkflowSchemaHandlerFactory {
	return &WorkflowSchemaHandlerFactory{
		Factory: func() gen.ProcessBehavior {
			return &WorkflowSchemaHandler{
				graphService: graphService,
			}
		},
	}
}

// HandlePut handles the UpsertWorkflowSchema http endpoint -- PUT /v1/schemas/{schemaID}
// @Summary Upsert workflow schema
// @Description Create or update a workflow schema
// @Tags schemas
// @Accept json
// @Produce json
// @Param schemaID path string true "Schema ID"
// @Param schema body workflow.GraphSchema true "Workflow Schema"
// @Success 200 {object} dtos.UpsertSchemaResponse
// @Failure 400 {object} dtos.BadRequestError
// @Failure 404 {object} dtos.NotFoundError
// @Failure 500 {object} dtos.InternalServerErrorResponse
// @Router /v1/schemas/{schemaID} [put]
func (h *WorkflowSchemaHandler) HandlePut(from gen.PID, w http.ResponseWriter, r *http.Request) error {
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

	return h.SendJSON(w, http.StatusOK, dtos.UpsertSchemaResponse{
		SchemaID: schemaID,
	})
}

// HandleGet returns the graph schema to the client -- GET /schemas/:schemaId
// @Summary Get workflow schema
// @Description Retrieve a workflow schema by ID
// @Tags schemas
// @Accept json
// @Produce json
// @Param schemaID path string true "Schema ID"
// @Success 200 {object} workflow.GraphSchema
// @Failure 400 {object} dtos.BadRequestError
// @Failure 404 {object} dtos.NotFoundError
// @Failure 500 {object} dtos.InternalServerErrorResponse
// @Router /v1/schemas/{schemaID} [get]
func (h *WorkflowSchemaHandler) HandleGet(from gen.PID, w http.ResponseWriter, r *http.Request) error {
	h.Log().Info("received get workflow schema request", "from", from, "remoteAddr", r.RemoteAddr)

	schemaID, err := h.GetPathParam(r, "schemaID")
	if err != nil {
		return h.SendBadRequest(w, err, []string{"schemaID"})
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
