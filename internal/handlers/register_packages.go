package handlers

import (
	"fmt"
	. "github.com/open-source-cloud/fuse/internal/packages"
	"github.com/open-source-cloud/fuse/pkg/packages"
	"net/http"

	"ergo.services/ergo/gen"
)

const (
	// RegisterPackagesHandlerName is the name of the register packages handler
	RegisterPackagesHandlerName = "register_packages_handler"
	// RegisterPackagesHandlerPoolName is the name of the register packages handler pool
	RegisterPackagesHandlerPoolName = "register_packages_handler_pool"
)

type (
	// RegisterPackagesHandlerFactory is the factory for the register packages handler
	RegisterPackagesHandlerFactory HandlerFactory[*RegisterPackagesHandler]

	// RegisterPackagesHandler is the handler for the register packages endpoint
	RegisterPackagesHandler struct {
		Handler
		packageRegistry Registry
	}

	// RegisterPackagesRequest register packages request data
	RegisterPackagesRequest struct {
		Packages []*packages.Package `json:"packages"`
	}
)

// NewRegisterPackagesHandler creates a new packages' handler factory
func NewRegisterPackagesHandler(packageRegistry Registry) *RegisterPackagesHandlerFactory {
	return &RegisterPackagesHandlerFactory{
		Factory: func() gen.ProcessBehavior {
			return &RegisterPackagesHandler{
				packageRegistry: packageRegistry,
			}
		},
	}
}

// HandlePut handles the PUT request for the register packages endpoint (PUT /packages/register)
func (h *RegisterPackagesHandler) HandlePut(from gen.PID, w http.ResponseWriter, r *http.Request) error {
	h.Log().Info("received register packages request from: %v remoteAddr: %s", from, r.RemoteAddr)

	var req RegisterPackagesRequest
	if err := h.BindJSON(w, r, &req); err != nil {
		return h.SendJSON(w, http.StatusBadRequest, Response{
			"message": fmt.Sprintf("invalid request: %s", err),
			"code":    BadRequest,
		})
	}

	for _, pkgData := range req.Packages {
		h.packageRegistry.Register(pkgData)
	}

	return h.SendJSON(w, http.StatusOK, Response{
		"message": "OK",
	})
}
