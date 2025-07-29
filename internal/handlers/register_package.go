package handlers

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/open-source-cloud/fuse/internal/packages"
	"github.com/open-source-cloud/fuse/internal/repositories"
	"github.com/open-source-cloud/fuse/pkg/workflow"

	"ergo.services/ergo/gen"
)

const (
	// RegisterPackageHandlerName is the name of the register package handler
	RegisterPackageHandlerName = "register_package_handler"
	// RegisterPackageHandlerPoolName is the name of the register package handler pool
	RegisterPackageHandlerPoolName = "register_package_handler_pool"
)

type (
	// RegisterPackageHandlerFactory is the factory for the register package handler
	RegisterPackageHandlerFactory HandlerFactory[*RegisterPackageHandler]

	// RegisterPackageHandler is the handler for the register package endpoint
	RegisterPackageHandler struct {
		Handler
		packageRegistry   packages.Registry
		packageRepository repositories.PackageRepository
	}
)

// NewRegisterPackageHandler creates a new register package handler factory
func NewRegisterPackageHandler(packageRegistry packages.Registry, packageRepository repositories.PackageRepository) *RegisterPackageHandlerFactory {
	return &RegisterPackageHandlerFactory{
		Factory: func() gen.ProcessBehavior {
			return &RegisterPackageHandler{
				packageRegistry:   packageRegistry,
				packageRepository: packageRepository,
			}
		},
	}
}

// HandlePut handles the PUT request for the register package endpoint (PUT /packages/:packageID)
func (h *RegisterPackageHandler) HandlePut(from gen.PID, w http.ResponseWriter, r *http.Request) error {
	h.Log().Info("received register package request from: %v remoteAddr: %s", from, r.RemoteAddr)

	var pkg *workflow.Package
	if err := h.BindJSON(w, r, &pkg); err != nil {
		return h.SendJSON(w, http.StatusBadRequest, Response{
			"message": fmt.Sprintf("invalid request: %s", err),
			"code":    BadRequest,
		})
	}

	if err := pkg.Validate(); err != nil {
		return h.SendJSON(w, http.StatusBadRequest, Response{
			"message": fmt.Sprintf("invalid package: %s", err),
			"code":    BadRequest,
		})
	}
	if err := h.packageRepository.Save(pkg); err != nil {
		return h.SendJSON(w, http.StatusInternalServerError, Response{
			"message": fmt.Sprintf("failed to save package: %s", err),
			"code":    InternalServerError,
		})
	}
	h.packageRegistry.Register(pkg)

	return h.SendJSON(w, http.StatusOK, Response{
		"message":   "Package registered successfully",
		"packageId": pkg.ID,
	})
}

// HandleGet handles the GET request for the package endpoint (GET /packages/:packageID)
func (h *RegisterPackageHandler) HandleGet(from gen.PID, w http.ResponseWriter, r *http.Request) error {
	h.Log().Info("received get package request from: %v remoteAddr: %s", from, r.RemoteAddr)

	packageID, err := h.GetPathParam(r, "packageID")
	if err != nil {
		return h.SendBadRequest(w, err, []string{"packageID is required"})
	}
	pkg, err := h.packageRepository.FindByID(packageID)
	if err != nil {
		if errors.Is(err, repositories.ErrPackageNotFound) {
			return h.SendJSON(w, http.StatusNotFound, Response{
				"message": fmt.Sprintf("package %s not found", packageID),
				"code":    "NOT_FOUND",
			})
		}
		return h.SendJSON(w, http.StatusInternalServerError, Response{
			"message": fmt.Sprintf("failed to get package: %s", err),
			"code":    InternalServerError,
		})
	}

	return h.SendJSON(w, http.StatusOK, pkg)
}
