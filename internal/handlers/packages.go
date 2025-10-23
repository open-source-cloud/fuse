package handlers

import (
	"net/http"

	"ergo.services/ergo/gen"
	"github.com/open-source-cloud/fuse/internal/dtos"
	"github.com/open-source-cloud/fuse/internal/services"
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
		packageService services.PackageService
	}
)

// NewPackagesHandler creates a new packages' handler factory
func NewPackagesHandler(packageService services.PackageService) *PackagesHandlerFactory {
	return &PackagesHandlerFactory{
		Factory: func() gen.ProcessBehavior {
			return &PackagesHandler{
				packageService: packageService,
			}
		},
	}
}

// HandleGet handles the GET request for the packages' endpoint (GET /packages)
// @Summary List all packages
// @Description Retrieve all registered packages
// @Tags packages
// @Accept json
// @Produce json
// @Success 200 {object} dtos.PackageListResponse
// @Failure 500 {object} dtos.InternalServerErrorResponse
// @Router /v1/packages [get]
func (h *PackagesHandler) HandleGet(from gen.PID, w http.ResponseWriter, r *http.Request) error {
	h.Log().Info("received list packages request from: %v remoteAddr: %s", from, r.RemoteAddr)

	packages, err := h.packageService.FindAll(services.PackageOptions{
		Load: true,
	})
	if err != nil {
		h.Log().Error("failed to list packages", "error", err, "from", from, "remoteAddr", r.RemoteAddr)
		return h.SendInternalError(w, err)
	}

	h.Log().Info("packages listed", "from", from, "remoteAddr", r.RemoteAddr, "packages", len(packages))

	// Convert packages to DTOs
	items := make([]dtos.PackageDTO, len(packages))
	for i, pkg := range packages {
		items[i] = dtos.ToPackageDTO(pkg)
	}

	return h.SendJSON(w, http.StatusOK, dtos.PackageListResponse{
		Metadata: dtos.PaginationMetadata{
			Total: len(packages),
			Page:  0,
			Size:  len(packages),
		},
		Items: items,
	})
}
