package handlers

import (
	"net/http"

	"ergo.services/ergo/gen"
)

const HealthCheckHandlerName = "health_check_handler"
const HealthCheckHandlerPoolName = "health_check_handler_pool"

type HealthCheckHandlerFactory HandlerFactory[*HealthCheckHandler]

type HealthCheckHandler struct {
	Handler
}

func NewHealthCheckHandler() *HealthCheckHandlerFactory {
	return &HealthCheckHandlerFactory{
		Factory: func() gen.ProcessBehavior {
			return &HealthCheckHandler{}
		},
	}
}

func (h *HealthCheckHandler) HandleGet(from gen.PID, w http.ResponseWriter, r *http.Request) error {
	return h.SendJSON(w, http.StatusOK, Response{
		"message": "OK",
	})
}
