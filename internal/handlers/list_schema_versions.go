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
	// ListSchemaVersionsHandlerName is the actor name for listing schema versions.
	ListSchemaVersionsHandlerName = "list_schema_versions_handler"
	// ListSchemaVersionsHandlerPoolName is the worker pool for ListSchemaVersionsHandler.
	ListSchemaVersionsHandlerPoolName = "list_schema_versions_handler_pool"
)

type (
	// ListSchemaVersionsHandlerFactory creates ListSchemaVersionsHandler actors.
	ListSchemaVersionsHandlerFactory HandlerFactory[*ListSchemaVersionsHandler]
	// ListSchemaVersionsHandler serves GET /v1/schemas/{schemaID}/versions.
	ListSchemaVersionsHandler struct {
		Handler
		graphService services.GraphService
	}
)

// NewListSchemaVersionsHandlerFactory builds a factory for ListSchemaVersionsHandler.
func NewListSchemaVersionsHandlerFactory(graphService services.GraphService) *ListSchemaVersionsHandlerFactory {
	return &ListSchemaVersionsHandlerFactory{
		Factory: func() gen.ProcessBehavior {
			return &ListSchemaVersionsHandler{
				graphService: graphService,
			}
		},
	}
}

// HandleGet handles GET /v1/schemas/{schemaID}/versions.
// @Summary List schema versions
// @Description List all versions of a workflow schema
// @Tags schemas
// @Accept json
// @Produce json
// @Param schemaID path string true "Schema ID"
// @Success 200 {object} dtos.SchemaVersionListResponse
// @Failure 400 {object} dtos.BadRequestError
// @Failure 404 {object} dtos.NotFoundError
// @Failure 500 {object} dtos.InternalServerErrorResponse
// @Router /v1/schemas/{schemaID}/versions [get]
func (h *ListSchemaVersionsHandler) HandleGet(from gen.PID, w http.ResponseWriter, r *http.Request) error {
	h.Log().Info("received list schema versions request", "from", from, "remoteAddr", r.RemoteAddr)

	schemaID, err := h.GetPathParam(r, "schemaID")
	if err != nil {
		return h.SendBadRequest(w, err, []string{"schemaID is required"})
	}

	versions, err := h.graphService.ListVersions(schemaID)
	if err != nil {
		if errors.Is(err, repositories.ErrGraphNotFound) {
			return h.SendNotFound(w, "schema not found", EmptyFields)
		}
		return h.SendInternalError(w, err)
	}

	history, err := h.graphService.GetVersionHistory(schemaID)
	if err != nil {
		return h.SendInternalError(w, err)
	}

	summaries := make([]dtos.SchemaVersionSummary, len(versions))
	for i, sv := range versions {
		summaries[i] = dtos.ToSchemaVersionSummary(sv)
	}

	return h.SendJSON(w, http.StatusOK, dtos.SchemaVersionListResponse{
		SchemaID:      schemaID,
		ActiveVersion: history.ActiveVersion,
		LatestVersion: history.LatestVersion,
		Versions:      summaries,
	})
}
