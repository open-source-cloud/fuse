package handlers

import (
	"net/http"

	"ergo.services/ergo/act"
	"ergo.services/ergo/gen"
)

const HealthCheckHandlerName = "health_check_handler"
const HealthCheckHandlerPoolName = "health_check_handler_pool"

type HealthCheckHandlerFactory HandlerFactory[*HealthCheckHandler]

type HealthCheckHandler struct {
	act.WebWorker
}

func NewHealthCheckHandler() *HealthCheckHandlerFactory {
	return &HealthCheckHandlerFactory{
		Factory: func() gen.ProcessBehavior {
			return &HealthCheckHandler{}
		},
	}
}

func (h *HealthCheckHandler) HandleGet(from gen.PID, w http.ResponseWriter, r *http.Request) error {
	return SendJSON(w, http.StatusOK, Response{
		"message": "OK",
	})
}
