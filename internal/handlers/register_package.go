package handlers

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/go-playground/validator/v10"
	"github.com/open-source-cloud/fuse/internal/dtos"
	"github.com/open-source-cloud/fuse/internal/repositories"
	"github.com/open-source-cloud/fuse/internal/services"

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
		packageService services.PackageService
	}
)

// NewRegisterPackageHandler creates a new register package handler factory
func NewRegisterPackageHandler(packageService services.PackageService) *RegisterPackageHandlerFactory {
	return &RegisterPackageHandlerFactory{
		Factory: func() gen.ProcessBehavior {
			return &RegisterPackageHandler{
				packageService: packageService,
			}
		},
	}
}

// HandlePut handles the PUT request for the register package endpoint (PUT /packages/:packageID)
// @Summary Register or update package
// @Description Register a new package or update existing one
// @Tags packages
// @Accept json
// @Produce json
// @Param packageID path string true "Package ID"
// @Param package body dtos.PackageDTO true "Package Data"
// @Success 200 {object} dtos.RegisterPackageResponse
// @Failure 400 {object} dtos.BadRequestError
// @Failure 404 {object} dtos.NotFoundError
// @Failure 500 {object} dtos.InternalServerErrorResponse
// @Router /v1/packages/{packageID} [put]
func (h *RegisterPackageHandler) HandlePut(from gen.PID, w http.ResponseWriter, r *http.Request) error {
	h.Log().Info("received register package request from: %v remoteAddr: %s", from, r.RemoteAddr)

	packageID, err := h.GetPathParam(r, "packageID")
	if err != nil {
		return h.SendBadRequest(w, err, []string{"packageID is required"})
	}

	var pkgDTO dtos.PackageDTO
	if err := h.BindJSON(w, r, &pkgDTO); err != nil {
		return h.SendBadRequest(w, err, []string{"body"})
	}

	// Convert DTO to domain model
	pkg := dtos.FromPackageDTO(pkgDTO)

	pkg, err = h.packageService.Save(pkg)
	if err != nil {
		h.Log().Error("failed to save package", "error", err, "packageID", packageID)
		if errors.As(err, &validator.ValidationErrors{}) {
			return h.SendValidationErr(w, err)
		}
		if errors.Is(err, repositories.ErrPackageNotFound) {
			return h.SendNotFound(w, fmt.Sprintf("package %s not found", packageID), []string{"packageID"})
		}
		return h.SendInternalError(w, err)
	}

	return h.SendJSON(w, http.StatusOK, dtos.RegisterPackageResponse{
		Message:   "Package registered successfully",
		PackageID: pkg.ID,
	})
}

// HandleGet handles the GET request for the package endpoint (GET /packages/:packageID)
// @Summary Get package by ID
// @Description Retrieve a specific package by ID
// @Tags packages
// @Accept json
// @Produce json
// @Param packageID path string true "Package ID"
// @Success 200 {object} dtos.PackageDTO
// @Failure 400 {object} dtos.BadRequestError
// @Failure 404 {object} dtos.NotFoundError
// @Failure 500 {object} dtos.InternalServerErrorResponse
// @Router /v1/packages/{packageID} [get]
func (h *RegisterPackageHandler) HandleGet(from gen.PID, w http.ResponseWriter, r *http.Request) error {
	h.Log().Info("received get package request from: %v remoteAddr: %s", from, r.RemoteAddr)

	packageID, err := h.GetPathParam(r, "packageID")
	if err != nil {
		return h.SendBadRequest(w, err, []string{"packageID is required"})
	}

	h.Log().Info("fetching package", "packageID", packageID)

	pkg, err := h.packageService.FindByID(packageID, services.PackageOptions{
		Load: true,
	})
	if err != nil {
		h.Log().Error("failed to get package", "error", err, "packageID", packageID)
		if errors.Is(err, repositories.ErrPackageNotFound) {
			return h.SendNotFound(w, fmt.Sprintf("package %s not found", packageID), []string{"packageID"})
		}
		return h.SendInternalError(w, err)
	}

	h.Log().Info("package fetched", "packageID", packageID)

	// Convert to DTO
	pkgDTO := dtos.ToPackageDTO(pkg)
	return h.SendJSON(w, http.StatusOK, pkgDTO)
}
