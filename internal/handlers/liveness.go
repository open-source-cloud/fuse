package handlers

import (
	"net/http"

	"ergo.services/ergo/gen"

	"github.com/open-source-cloud/fuse/internal/dtos"
)

const (
	// LivenessHandlerName is the registered name of the liveness handler.
	LivenessHandlerName = "liveness_handler"
	// LivenessHandlerPoolName is the registered name of the liveness handler pool.
	LivenessHandlerPoolName = "liveness_handler_pool"
)

// LivenessHandlerFactory is the factory for the liveness handler.
type LivenessHandlerFactory HandlerFactory[*LivenessHandler]

// LivenessHandler handles GET /healthz — always returns 200 while the process is alive.
type LivenessHandler struct {
	Handler
}

// NewLivenessHandler creates a new LivenessHandlerFactory.
func NewLivenessHandler() *LivenessHandlerFactory {
	return &LivenessHandlerFactory{
		Factory: func() gen.ProcessBehavior {
			return &LivenessHandler{}
		},
	}
}

// HandleGet handles GET /healthz.
// @Summary Liveness probe
// @Description Returns 200 as long as the process is alive; used by orchestrators to detect crashes.
// @Tags health
// @Produce json
// @Success 200 {object} dtos.LivenessResponse
// @Router /healthz [get]
func (h *LivenessHandler) HandleGet(_ gen.PID, w http.ResponseWriter, _ *http.Request) error {
	return h.SendJSON(w, http.StatusOK, dtos.LivenessResponse{Status: "ok"})
}
