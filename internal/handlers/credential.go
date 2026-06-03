package handlers

import (
	"errors"
	"fmt"
	"net/http"

	"ergo.services/ergo/gen"
	"github.com/open-source-cloud/fuse/internal/app/config"
	"github.com/open-source-cloud/fuse/internal/dtos"
	"github.com/open-source-cloud/fuse/internal/repositories"
	"github.com/open-source-cloud/fuse/internal/services"
	"github.com/open-source-cloud/fuse/pkg/workflow"
)

const (
	// CredentialHandlerName is the name of the single-credential handler.
	CredentialHandlerName = "credential_handler" //nolint:gosec // G101 false positive: actor name, not a secret
	// CredentialHandlerPoolName is the name of the single-credential handler pool.
	CredentialHandlerPoolName = "credential_handler_pool" //nolint:gosec // G101 false positive: pool name, not a secret
)

type (
	// CredentialHandlerFactory is the factory for the single-credential handler.
	CredentialHandlerFactory HandlerFactory[*CredentialHandler]

	// CredentialHandler handles a single credential resource.
	CredentialHandler struct {
		Handler
		credentialService  services.CredentialService
		defaultEnvironment string
	}
)

// NewCredentialHandler creates a new single-credential handler factory.
func NewCredentialHandler(credentialService services.CredentialService, cfg *config.Config) *CredentialHandlerFactory {
	return &CredentialHandlerFactory{
		Factory: func() gen.ProcessBehavior {
			return &CredentialHandler{
				credentialService:  credentialService,
				defaultEnvironment: cfg.Environment,
			}
		},
	}
}

// environmentParam returns the ?environment= query value or the engine default.
func (h *CredentialHandler) environmentParam(r *http.Request) string {
	if env := r.URL.Query().Get("environment"); env != "" {
		return env
	}
	return h.defaultEnvironment
}

// HandleGet retrieves a single credential's metadata (GET /v1/credentials/{id})
// @Summary Get credential by id
// @Description Retrieve a credential's metadata (field values are never returned)
// @Tags credentials
// @Accept json
// @Produce json
// @Param id path string true "Credential id"
// @Success 200 {object} dtos.CredentialDTO
// @Failure 404 {object} dtos.NotFoundError
// @Failure 500 {object} dtos.InternalServerErrorResponse
// @Router /v1/credentials/{id} [get]
func (h *CredentialHandler) HandleGet(from gen.PID, w http.ResponseWriter, r *http.Request) error {
	h.Log().Info("received get credential request from: %v remoteAddr: %s", from, r.RemoteAddr)

	id, err := h.GetPathParam(r, "id")
	if err != nil {
		return h.SendBadRequest(w, err, []string{"id is required"})
	}

	cred, err := h.credentialService.FindByID(id)
	if err != nil {
		if errors.Is(err, repositories.ErrCredentialNotFound) {
			return h.SendNotFound(w, fmt.Sprintf("credential %s not found", id), []string{"id"})
		}
		return h.SendInternalError(w, err)
	}

	return h.SendJSON(w, http.StatusOK, dtos.ToCredentialDTO(cred))
}

// HandlePut creates or updates a credential and its field values for an environment.
// @Summary Create or update credential
// @Description Upsert a credential; field values are stored in the SecretStore for the target environment (?environment=)
// @Tags credentials
// @Accept json
// @Produce json
// @Param id path string true "Credential id"
// @Param environment query string false "Environment scope for the field values (defaults to FUSE_ENVIRONMENT)"
// @Param credential body dtos.UpsertCredentialRequest true "Credential data"
// @Success 200 {object} dtos.UpsertCredentialResponse
// @Failure 400 {object} dtos.BadRequestError
// @Failure 500 {object} dtos.InternalServerErrorResponse
// @Router /v1/credentials/{id} [put]
func (h *CredentialHandler) HandlePut(from gen.PID, w http.ResponseWriter, r *http.Request) error {
	h.Log().Info("received upsert credential request from: %v remoteAddr: %s", from, r.RemoteAddr)

	id, err := h.GetPathParam(r, "id")
	if err != nil {
		return h.SendBadRequest(w, err, []string{"id is required"})
	}

	var req dtos.UpsertCredentialRequest
	if bindErr := h.BindJSON(w, r, &req); bindErr != nil {
		return h.SendBadRequest(w, bindErr, []string{"body"})
	}

	cred := workflow.NewCredential(id, req.Type, req.Description, nil)
	if _, saveErr := h.credentialService.Save(cred, req.Fields, h.environmentParam(r)); saveErr != nil {
		if errors.Is(saveErr, services.ErrReadOnlySecretStore) {
			return h.SendBadRequest(w, saveErr, []string{"SECRETS_DRIVER"})
		}
		// Validation errors (id/type/field format) are client errors.
		if cred.Validate() != nil {
			return h.SendBadRequest(w, saveErr, []string{"credential"})
		}
		return h.SendInternalError(w, saveErr)
	}

	return h.SendJSON(w, http.StatusOK, dtos.UpsertCredentialResponse{
		Message:    "Credential saved successfully",
		Credential: id,
	})
}

// HandleDelete removes a credential and its field values for an environment.
// @Summary Delete credential
// @Description Delete a credential's metadata and its field values for the target environment (?environment=)
// @Tags credentials
// @Accept json
// @Produce json
// @Param id path string true "Credential id"
// @Param environment query string false "Environment scope for the field values (defaults to FUSE_ENVIRONMENT)"
// @Success 204 "No Content"
// @Failure 404 {object} dtos.NotFoundError
// @Failure 500 {object} dtos.InternalServerErrorResponse
// @Router /v1/credentials/{id} [delete]
func (h *CredentialHandler) HandleDelete(from gen.PID, w http.ResponseWriter, r *http.Request) error {
	h.Log().Info("received delete credential request from: %v remoteAddr: %s", from, r.RemoteAddr)

	id, err := h.GetPathParam(r, "id")
	if err != nil {
		return h.SendBadRequest(w, err, []string{"id is required"})
	}

	if delErr := h.credentialService.Delete(id, h.environmentParam(r)); delErr != nil {
		if errors.Is(delErr, repositories.ErrCredentialNotFound) {
			return h.SendNotFound(w, fmt.Sprintf("credential %s not found", id), []string{"id"})
		}
		return h.SendInternalError(w, delErr)
	}

	return h.SendJSON(w, http.StatusNoContent, nil)
}
