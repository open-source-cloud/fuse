package handlers

import (
	"errors"
	"net/http"

	"ergo.services/ergo/gen"
	"github.com/open-source-cloud/fuse/internal/dtos"
	"github.com/open-source-cloud/fuse/internal/repositories"
	"github.com/open-source-cloud/fuse/internal/services"
)

const (
	// RollbackSchemaHandlerName is the actor name for rolling back a schema.
	RollbackSchemaHandlerName = "rollback_schema_handler"
	// RollbackSchemaHandlerPoolName is the worker pool for RollbackSchemaHandler.
	RollbackSchemaHandlerPoolName = "rollback_schema_handler_pool"
)

type (
	// RollbackSchemaHandlerFactory creates RollbackSchemaHandler actors.
	RollbackSchemaHandlerFactory HandlerFactory[*RollbackSchemaHandler]
	// RollbackSchemaHandler serves POST /v1/schemas/{schemaID}/rollback.
	RollbackSchemaHandler struct {
		Handler
		graphService services.GraphService
	}
)

// NewRollbackSchemaHandlerFactory builds a factory for RollbackSchemaHandler.
func NewRollbackSchemaHandlerFactory(graphService services.GraphService) *RollbackSchemaHandlerFactory {
	return &RollbackSchemaHandlerFactory{
		Factory: func() gen.ProcessBehavior {
			return &RollbackSchemaHandler{
				graphService: graphService,
			}
		},
	}
}

// HandlePost handles POST /v1/schemas/{schemaID}/rollback.
// @Summary Rollback schema to a previous version
// @Description Create a new version with the content of an older version and activate it
// @Tags schemas
// @Accept json
// @Produce json
// @Param schemaID path string true "Schema ID"
// @Param request body dtos.RollbackRequest true "Rollback Request"
// @Success 200 {object} dtos.RollbackResponse
// @Failure 400 {object} dtos.BadRequestError
// @Failure 404 {object} dtos.NotFoundError
// @Failure 500 {object} dtos.InternalServerErrorResponse
// @Router /v1/schemas/{schemaID}/rollback [post]
func (h *RollbackSchemaHandler) HandlePost(from gen.PID, w http.ResponseWriter, r *http.Request) error {
	h.Log().Info("received rollback schema request", "from", from, "remoteAddr", r.RemoteAddr)

	schemaID, err := h.GetPathParam(r, "schemaID")
	if err != nil {
		return h.SendBadRequest(w, err, []string{"schemaID is required"})
	}

	var req dtos.RollbackRequest
	if err := h.BindJSON(w, r, &req); err != nil {
		return h.SendBadRequest(w, err, []string{"body"})
	}
	if req.Version < 1 {
		return h.SendBadRequest(w, errors.New("version must be a positive integer"), []string{"version"})
	}

	sv, err := h.graphService.Rollback(schemaID, req.Version, req.Comment)
	if err != nil {
		if errors.Is(err, repositories.ErrSchemaVersionNotFound) {
			return h.SendNotFound(w, "schema version not found", EmptyFields)
		}
		if errors.Is(err, repositories.ErrGraphNotFound) {
			return h.SendNotFound(w, "schema not found", EmptyFields)
		}
		return h.SendInternalError(w, err)
	}

	h.Log().Info("rolled back schema", "schemaID", schemaID, "newVersion", sv.Version, "restoredFrom", req.Version)

	return h.SendJSON(w, http.StatusOK, dtos.RollbackResponse{
		SchemaID:     schemaID,
		NewVersion:   sv.Version,
		RestoredFrom: req.Version,
	})
}
