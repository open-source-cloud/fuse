package handlers

import (
	"net/http"

	"ergo.services/ergo/gen"
	"github.com/open-source-cloud/fuse/internal/repositories"
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
		packageRepository repositories.PackageRepository
	}
)

// NewPackagesHandler creates a new packages' handler factory
func NewPackagesHandler(packageRepository repositories.PackageRepository) *PackagesHandlerFactory {
	return &PackagesHandlerFactory{
		Factory: func() gen.ProcessBehavior {
			return &PackagesHandler{
				packageRepository: packageRepository,
			}
		},
	}
}

// HandleGet handles the GET request for the packages' endpoint (GET /packages)
func (h *PackagesHandler) HandleGet(from gen.PID, w http.ResponseWriter, r *http.Request) error {
	h.Log().Info("received list packages request from: %v remoteAddr: %s", from, r.RemoteAddr)

	packages, err := h.packageRepository.FindAll()
	if err != nil {
		return h.SendJSON(w, http.StatusInternalServerError, Response{
			"message": "Failed to list packages",
			"code":    InternalServerError,
		})
	}

	return h.SendJSON(w, http.StatusOK, Response{
		"metadata": PaginationMetadata{
			Total: len(packages),
			Page:  0,
			Size:  0,
		},
		"items": packages,
	})
}
