package handlers

import (
	"net/http"

	"ergo.services/ergo/gen"
)

const (
	// PackagesHandlerName is the name of the packages' handler
	PackagesHandlerName = "packages_handler"
	// PackagesHandlerPoolName is the name of the packages' handler pool
	PackagesHandlerPoolName = "packages_handler_pool"
)

type (
	// PackagesHandlerFactory is the factory for the packages' handler
	PackagesHandlerFactory HandlerFactory[*PackagesHandler]

	// PackagesHandler is the handler for the packages' endpoint
	PackagesHandler struct {
		Handler
	}
)

// NewPackagesHandler creates a new packages' handler factory
func NewPackagesHandler() *PackagesHandlerFactory {
	return &PackagesHandlerFactory{
		Factory: func() gen.ProcessBehavior {
			return &PackagesHandler{}
		},
	}
}

// HandleGet handles the GET request for the packages' endpoint (GET /packages)
func (h *PackagesHandler) HandleGet(from gen.PID, w http.ResponseWriter, r *http.Request) error {
	h.Log().Info("received list packages request from: %v remoteAddr: %s", from, r.RemoteAddr)

	return h.SendJSON(w, http.StatusOK, Response{
		"message": "OK",
	})
}
