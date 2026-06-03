package handlers

import (
	"errors"
	"fmt"
	"net/http"

	"ergo.services/ergo/gen"
	"github.com/open-source-cloud/fuse/internal/dtos"
	"github.com/open-source-cloud/fuse/internal/repositories"
	"github.com/open-source-cloud/fuse/internal/services"
	"github.com/open-source-cloud/fuse/pkg/workflow"
)

const (
	// EnvironmentHandlerName is the name of the single-environment handler.
	EnvironmentHandlerName = "environment_handler"
	// EnvironmentHandlerPoolName is the name of the single-environment handler pool.
	EnvironmentHandlerPoolName = "environment_handler_pool"
)

type (
	// EnvironmentHandlerFactory is the factory for the single-environment handler.
	EnvironmentHandlerFactory HandlerFactory[*EnvironmentHandler]

	// EnvironmentHandler handles a single environment resource.
	EnvironmentHandler struct {
		Handler
		environmentService services.EnvironmentService
	}
)

// NewEnvironmentHandler creates a new single-environment handler factory.
func NewEnvironmentHandler(environmentService services.EnvironmentService) *EnvironmentHandlerFactory {
	return &EnvironmentHandlerFactory{
		Factory: func() gen.ProcessBehavior {
			return &EnvironmentHandler{
				environmentService: environmentService,
			}
		},
	}
}

// HandleGet retrieves a single environment (GET /v1/environments/{name})
// @Summary Get environment by name
// @Description Retrieve a single environment
// @Tags environments
// @Accept json
// @Produce json
// @Param name path string true "Environment name"
// @Success 200 {object} dtos.EnvironmentDTO
// @Failure 404 {object} dtos.NotFoundError
// @Failure 500 {object} dtos.InternalServerErrorResponse
// @Router /v1/environments/{name} [get]
func (h *EnvironmentHandler) HandleGet(from gen.PID, w http.ResponseWriter, r *http.Request) error {
	h.Log().Info("received get environment request from: %v remoteAddr: %s", from, r.RemoteAddr)

	name, err := h.GetPathParam(r, "name")
	if err != nil {
		return h.SendBadRequest(w, err, []string{"name is required"})
	}

	env, err := h.environmentService.FindByID(name)
	if err != nil {
		if errors.Is(err, repositories.ErrEnvironmentNotFound) {
			return h.SendNotFound(w, fmt.Sprintf("environment %s not found", name), []string{"name"})
		}
		return h.SendInternalError(w, err)
	}

	return h.SendJSON(w, http.StatusOK, dtos.ToEnvironmentDTO(env))
}

// HandlePut creates or updates an environment (PUT /v1/environments/{name})
// @Summary Create or update environment
// @Description Upsert an environment; the path name is authoritative
// @Tags environments
// @Accept json
// @Produce json
// @Param name path string true "Environment name"
// @Param environment body dtos.EnvironmentDTO true "Environment data"
// @Success 200 {object} dtos.UpsertEnvironmentResponse
// @Failure 400 {object} dtos.BadRequestError
// @Failure 500 {object} dtos.InternalServerErrorResponse
// @Router /v1/environments/{name} [put]
func (h *EnvironmentHandler) HandlePut(from gen.PID, w http.ResponseWriter, r *http.Request) error {
	h.Log().Info("received upsert environment request from: %v remoteAddr: %s", from, r.RemoteAddr)

	name, err := h.GetPathParam(r, "name")
	if err != nil {
		return h.SendBadRequest(w, err, []string{"name is required"})
	}

	var dto dtos.EnvironmentDTO
	if bindErr := h.BindJSON(w, r, &dto); bindErr != nil {
		return h.SendBadRequest(w, bindErr, []string{"body"})
	}

	// The path name is authoritative over the body.
	dto.Name = name
	env := dtos.FromEnvironmentDTO(dto)
	if validateErr := env.Validate(); validateErr != nil {
		return h.SendBadRequest(w, validateErr, []string{"name"})
	}

	if _, saveErr := h.environmentService.Save(env); saveErr != nil {
		return h.SendInternalError(w, saveErr)
	}

	return h.SendJSON(w, http.StatusOK, dtos.UpsertEnvironmentResponse{
		Message:     "Environment saved successfully",
		Environment: env.Name,
	})
}

// HandleDelete removes an environment (DELETE /v1/environments/{name})
// @Summary Delete environment
// @Description Delete an environment (the default environment cannot be deleted)
// @Tags environments
// @Accept json
// @Produce json
// @Param name path string true "Environment name"
// @Success 204 "No Content"
// @Failure 400 {object} dtos.BadRequestError
// @Failure 500 {object} dtos.InternalServerErrorResponse
// @Router /v1/environments/{name} [delete]
func (h *EnvironmentHandler) HandleDelete(from gen.PID, w http.ResponseWriter, r *http.Request) error {
	h.Log().Info("received delete environment request from: %v remoteAddr: %s", from, r.RemoteAddr)

	name, err := h.GetPathParam(r, "name")
	if err != nil {
		return h.SendBadRequest(w, err, []string{"name is required"})
	}

	if name == workflow.DefaultEnvironmentName {
		return h.SendBadRequest(w, fmt.Errorf("the default environment cannot be deleted"), []string{"name"})
	}

	if delErr := h.environmentService.Delete(name); delErr != nil {
		return h.SendInternalError(w, delErr)
	}

	return h.SendJSON(w, http.StatusNoContent, nil)
}
