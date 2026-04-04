package handlers

import (
	"net/http"

	"ergo.services/ergo/gen"
	"github.com/open-source-cloud/fuse/internal/dtos"
	"github.com/open-source-cloud/fuse/internal/services"
)

const (
	// ListSchemasHandlerName is the actor name for listing workflow schemas.
	ListSchemasHandlerName = "list_schemas_handler"
	// ListSchemasHandlerPoolName is the worker pool for ListSchemasHandler.
	ListSchemasHandlerPoolName = "list_schemas_handler_pool"
)

type (
	// ListSchemasHandlerFactory creates ListSchemasHandler actors.
	ListSchemasHandlerFactory HandlerFactory[*ListSchemasHandler]
	// ListSchemasHandler serves GET /v1/schemas.
	ListSchemasHandler struct {
		Handler
		graphService services.GraphService
	}
)

// NewListSchemasHandlerFactory builds a factory for ListSchemasHandler.
func NewListSchemasHandlerFactory(graphService services.GraphService) *ListSchemasHandlerFactory {
	return &ListSchemasHandlerFactory{
		Factory: func() gen.ProcessBehavior {
			return &ListSchemasHandler{
				graphService: graphService,
			}
		},
	}
}

// HandleGet handles GET /v1/schemas.
// @Summary List workflow schemas
// @Description List all workflow graph schemas registered in the app
// @Tags schemas
// @Accept json
// @Produce json
// @Success 200 {object} dtos.SchemaListResponse
// @Failure 500 {object} dtos.InternalServerErrorResponse
// @Router /v1/schemas [get]
func (h *ListSchemasHandler) HandleGet(from gen.PID, w http.ResponseWriter, r *http.Request) error {
	h.Log().Info("received list schemas request from: %v remoteAddr: %s", from, r.RemoteAddr)

	items, err := h.graphService.ListSchemas()
	if err != nil {
		h.Log().Error("failed to list schemas", "error", err, "from", from)
		return h.SendInternalError(w, err)
	}

	dtoItems := make([]dtos.GraphSchemaSummaryDTO, len(items))
	for i, it := range items {
		dtoItems[i] = dtos.GraphSchemaSummaryDTO{SchemaID: it.SchemaID, Name: it.Name}
	}

	h.Log().Info("schemas listed", "from", from, "count", len(items))

	return h.SendJSON(w, http.StatusOK, dtos.SchemaListResponse{
		Metadata: dtos.PaginationMetadata{
			Total: len(items),
			Page:  0,
			Size:  len(items),
		},
		Items: dtoItems,
	})
}
