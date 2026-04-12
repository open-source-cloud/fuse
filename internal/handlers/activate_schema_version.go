package handlers

import (
	"errors"
	"net/http"
	"strconv"

	"ergo.services/ergo/gen"
	"github.com/open-source-cloud/fuse/internal/dtos"
	"github.com/open-source-cloud/fuse/internal/repositories"
	"github.com/open-source-cloud/fuse/internal/services"
)

const (
	// ActivateSchemaVersionHandlerName is the actor name for activating a schema version.
	ActivateSchemaVersionHandlerName = "activate_schema_version_handler"
	// ActivateSchemaVersionHandlerPoolName is the worker pool for ActivateSchemaVersionHandler.
	ActivateSchemaVersionHandlerPoolName = "activate_schema_version_handler_pool"
)

type (
	// ActivateSchemaVersionHandlerFactory creates ActivateSchemaVersionHandler actors.
	ActivateSchemaVersionHandlerFactory HandlerFactory[*ActivateSchemaVersionHandler]
	// ActivateSchemaVersionHandler serves POST /v1/schemas/{schemaID}/versions/{version}/activate.
	ActivateSchemaVersionHandler struct {
		Handler
		graphService services.GraphService
	}
)

// NewActivateSchemaVersionHandlerFactory builds a factory for ActivateSchemaVersionHandler.
func NewActivateSchemaVersionHandlerFactory(graphService services.GraphService) *ActivateSchemaVersionHandlerFactory {
	return &ActivateSchemaVersionHandlerFactory{
		Factory: func() gen.ProcessBehavior {
			return &ActivateSchemaVersionHandler{
				graphService: graphService,
			}
		},
	}
}

// HandlePost handles POST /v1/schemas/{schemaID}/versions/{version}/activate.
// @Summary Activate a schema version
// @Description Set a specific version as the active version for new workflow executions
// @Tags schemas
// @Accept json
// @Produce json
// @Param schemaID path string true "Schema ID"
// @Param version path int true "Version number to activate"
// @Success 200 {object} dtos.ActivateVersionResponse
// @Failure 400 {object} dtos.BadRequestError
// @Failure 404 {object} dtos.NotFoundError
// @Failure 500 {object} dtos.InternalServerErrorResponse
// @Router /v1/schemas/{schemaID}/versions/{version}/activate [post]
func (h *ActivateSchemaVersionHandler) HandlePost(from gen.PID, w http.ResponseWriter, r *http.Request) error {
	h.Log().Info("received activate schema version request", "from", from, "remoteAddr", r.RemoteAddr)

	schemaID, err := h.GetPathParam(r, "schemaID")
	if err != nil {
		return h.SendBadRequest(w, err, []string{"schemaID is required"})
	}

	versionStr, err := h.GetPathParam(r, "version")
	if err != nil {
		return h.SendBadRequest(w, err, []string{"version is required"})
	}

	version, err := strconv.Atoi(versionStr)
	if err != nil || version < 1 {
		return h.SendBadRequest(w, errors.New("version must be a positive integer"), []string{"version"})
	}

	// Capture previous active version before activating
	history, err := h.graphService.GetVersionHistory(schemaID)
	if err != nil {
		if errors.Is(err, repositories.ErrGraphNotFound) {
			return h.SendNotFound(w, "schema not found", EmptyFields)
		}
		return h.SendInternalError(w, err)
	}
	previousVersion := history.ActiveVersion

	if err := h.graphService.SetActiveVersion(schemaID, version); err != nil {
		if errors.Is(err, repositories.ErrSchemaVersionNotFound) {
			return h.SendNotFound(w, "schema version not found", EmptyFields)
		}
		return h.SendInternalError(w, err)
	}

	h.Log().Info("activated schema version", "schemaID", schemaID, "version", version, "previousVersion", previousVersion)

	return h.SendJSON(w, http.StatusOK, dtos.ActivateVersionResponse{
		SchemaID:        schemaID,
		ActiveVersion:   version,
		PreviousVersion: previousVersion,
	})
}
