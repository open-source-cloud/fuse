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
	// GetSchemaVersionHandlerName is the actor name for getting a specific schema version.
	GetSchemaVersionHandlerName = "get_schema_version_handler"
	// GetSchemaVersionHandlerPoolName is the worker pool for GetSchemaVersionHandler.
	GetSchemaVersionHandlerPoolName = "get_schema_version_handler_pool"
)

type (
	// GetSchemaVersionHandlerFactory creates GetSchemaVersionHandler actors.
	GetSchemaVersionHandlerFactory HandlerFactory[*GetSchemaVersionHandler]
	// GetSchemaVersionHandler serves GET /v1/schemas/{schemaID}/versions/{version}.
	GetSchemaVersionHandler struct {
		Handler
		graphService services.GraphService
	}
)

// NewGetSchemaVersionHandlerFactory builds a factory for GetSchemaVersionHandler.
func NewGetSchemaVersionHandlerFactory(graphService services.GraphService) *GetSchemaVersionHandlerFactory {
	return &GetSchemaVersionHandlerFactory{
		Factory: func() gen.ProcessBehavior {
			return &GetSchemaVersionHandler{
				graphService: graphService,
			}
		},
	}
}

// HandleGet handles GET /v1/schemas/{schemaID}/versions/{version}.
// @Summary Get specific schema version
// @Description Retrieve a specific version of a workflow schema
// @Tags schemas
// @Accept json
// @Produce json
// @Param schemaID path string true "Schema ID"
// @Param version path int true "Version number"
// @Success 200 {object} dtos.SchemaVersionResponse
// @Failure 400 {object} dtos.BadRequestError
// @Failure 404 {object} dtos.NotFoundError
// @Failure 500 {object} dtos.InternalServerErrorResponse
// @Router /v1/schemas/{schemaID}/versions/{version} [get]
func (h *GetSchemaVersionHandler) HandleGet(from gen.PID, w http.ResponseWriter, r *http.Request) error {
	h.Log().Info("received get schema version request", "from", from, "remoteAddr", r.RemoteAddr)

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

	versions, err := h.graphService.ListVersions(schemaID)
	if err != nil {
		if errors.Is(err, repositories.ErrGraphNotFound) {
			return h.SendNotFound(w, "schema not found", EmptyFields)
		}
		return h.SendInternalError(w, err)
	}

	for _, sv := range versions {
		if sv.Version == version {
			return h.SendJSON(w, http.StatusOK, dtos.ToSchemaVersionResponse(sv))
		}
	}

	return h.SendNotFound(w, "schema version not found", EmptyFields)
}
