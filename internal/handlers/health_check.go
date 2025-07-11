package handlers

import (
	"net/http"

	"ergo.services/ergo/gen"
)

const (
	// HealthCheckHandlerName is the name of the health check handler
	HealthCheckHandlerName = "health_check_handler"
	// HealthCheckHandlerPoolName is the name of the health check handler pool
	HealthCheckHandlerPoolName = "health_check_handler_pool"
)

// HealthCheckHandlerFactory is the factory for the health check handler
type HealthCheckHandlerFactory HandlerFactory[*HealthCheckHandler]

// HealthCheckHandler is the handler for the health check endpoint
type HealthCheckHandler struct {
	Handler
}

// NewHealthCheckHandler creates a new health check handler factory
func NewHealthCheckHandler() *HealthCheckHandlerFactory {
	return &HealthCheckHandlerFactory{
		Factory: func() gen.ProcessBehavior {
			return &HealthCheckHandler{}
		},
	}
}

// HandleGet handles the GET request for the health check endpoint (GET /health)
func (h *HealthCheckHandler) HandleGet(from gen.PID, w http.ResponseWriter, r *http.Request) error {
	h.Log().Info("received health check request from: %v remoteAddr: %s", from, r.RemoteAddr)

	return h.SendJSON(w, http.StatusOK, Response{
		"message": "OK",
	})
}
