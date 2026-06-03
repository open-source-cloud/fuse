package handlers

import (
	"net/http"

	"ergo.services/ergo/gen"
	"github.com/open-source-cloud/fuse/internal/dtos"
	"github.com/open-source-cloud/fuse/internal/services"
)

const (
	// EnvironmentsHandlerName is the name of the environments list handler.
	EnvironmentsHandlerName = "environments_handler"
	// EnvironmentsHandlerPoolName is the name of the environments list handler pool.
	EnvironmentsHandlerPoolName = "environments_handler_pool"
)

type (
	// EnvironmentsHandlerFactory is the factory for the environments list handler.
	EnvironmentsHandlerFactory HandlerFactory[*EnvironmentsHandler]

	// EnvironmentsHandler handles the environments collection endpoint.
	EnvironmentsHandler struct {
		Handler
		environmentService services.EnvironmentService
	}
)

// NewEnvironmentsHandler creates a new environments list handler factory.
func NewEnvironmentsHandler(environmentService services.EnvironmentService) *EnvironmentsHandlerFactory {
	return &EnvironmentsHandlerFactory{
		Factory: func() gen.ProcessBehavior {
			return &EnvironmentsHandler{
				environmentService: environmentService,
			}
		},
	}
}

// HandleGet lists all declared environments (GET /v1/environments)
// @Summary List environments
// @Description Retrieve all declared environments
// @Tags environments
// @Accept json
// @Produce json
// @Success 200 {object} dtos.EnvironmentListResponse
// @Failure 500 {object} dtos.InternalServerErrorResponse
// @Router /v1/environments [get]
func (h *EnvironmentsHandler) HandleGet(from gen.PID, w http.ResponseWriter, r *http.Request) error {
	h.Log().Info("received list environments request from: %v remoteAddr: %s", from, r.RemoteAddr)

	envs, err := h.environmentService.FindAll()
	if err != nil {
		return h.SendInternalError(w, err)
	}

	items := make([]dtos.EnvironmentDTO, len(envs))
	for i, env := range envs {
		items[i] = dtos.ToEnvironmentDTO(env)
	}

	return h.SendJSON(w, http.StatusOK, dtos.EnvironmentListResponse{Items: items})
}
