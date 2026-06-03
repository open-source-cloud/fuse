package handlers

import (
	"net/http"

	"ergo.services/ergo/gen"
	"github.com/open-source-cloud/fuse/internal/dtos"
	"github.com/open-source-cloud/fuse/internal/services"
)

const (
	// CredentialsHandlerName is the name of the credentials list handler.
	CredentialsHandlerName = "credentials_handler"
	// CredentialsHandlerPoolName is the name of the credentials list handler pool.
	CredentialsHandlerPoolName = "credentials_handler_pool"
)

type (
	// CredentialsHandlerFactory is the factory for the credentials list handler.
	CredentialsHandlerFactory HandlerFactory[*CredentialsHandler]

	// CredentialsHandler handles the credentials collection endpoint.
	CredentialsHandler struct {
		Handler
		credentialService services.CredentialService
	}
)

// NewCredentialsHandler creates a new credentials list handler factory.
func NewCredentialsHandler(credentialService services.CredentialService) *CredentialsHandlerFactory {
	return &CredentialsHandlerFactory{
		Factory: func() gen.ProcessBehavior {
			return &CredentialsHandler{credentialService: credentialService}
		},
	}
}

// HandleGet lists all credentials, metadata only (GET /v1/credentials)
// @Summary List credentials
// @Description Retrieve all credentials (metadata only; field values are never returned)
// @Tags credentials
// @Accept json
// @Produce json
// @Success 200 {object} dtos.CredentialListResponse
// @Failure 500 {object} dtos.InternalServerErrorResponse
// @Router /v1/credentials [get]
func (h *CredentialsHandler) HandleGet(from gen.PID, w http.ResponseWriter, r *http.Request) error {
	h.Log().Info("received list credentials request from: %v remoteAddr: %s", from, r.RemoteAddr)

	creds, err := h.credentialService.FindAll()
	if err != nil {
		return h.SendInternalError(w, err)
	}

	items := make([]dtos.CredentialDTO, len(creds))
	for i, c := range creds {
		items[i] = dtos.ToCredentialDTO(c)
	}

	return h.SendJSON(w, http.StatusOK, dtos.CredentialListResponse{Items: items})
}
