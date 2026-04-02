package handlers

import (
	"net/http"

	"ergo.services/ergo/gen"
	"github.com/open-source-cloud/fuse/internal/actors/actornames"
	"github.com/open-source-cloud/fuse/internal/dtos"
	"github.com/open-source-cloud/fuse/internal/messaging"
	"github.com/open-source-cloud/fuse/internal/repositories"
	internalworkflow "github.com/open-source-cloud/fuse/internal/workflow"
)

type (
	// ResolveAwakeableHandler is the handler for the resolve awakeable endpoint
	ResolveAwakeableHandler struct {
		Handler
		awakeableRepo repositories.AwakeableRepository
	}
	// ResolveAwakeableHandlerFactory is a factory for creating ResolveAwakeableHandler actors
	ResolveAwakeableHandlerFactory HandlerFactory[*ResolveAwakeableHandler]
)

const (
	// ResolveAwakeableHandlerName is the name of the ResolveAwakeableHandler actor
	ResolveAwakeableHandlerName = "resolve_awakeable_handler"
	// ResolveAwakeableHandlerPoolName is the name of the ResolveAwakeableHandler pool
	ResolveAwakeableHandlerPoolName = "resolve_awakeable_handler_pool"
)

// NewResolveAwakeableHandlerFactory creates a new ResolveAwakeableHandlerFactory
func NewResolveAwakeableHandlerFactory(awakeableRepo repositories.AwakeableRepository) *ResolveAwakeableHandlerFactory {
	return &ResolveAwakeableHandlerFactory{
		Factory: func() gen.ProcessBehavior {
			return &ResolveAwakeableHandler{
				awakeableRepo: awakeableRepo,
			}
		},
	}
}

// HandlePost handles the resolve awakeable endpoint (POST /v1/awakeables/{awakeableID}/resolve)
func (h *ResolveAwakeableHandler) HandlePost(_ gen.PID, w http.ResponseWriter, r *http.Request) error {
	awakeableID, err := h.GetPathParam(r, "awakeableID")
	if err != nil {
		return h.SendBadRequest(w, err, EmptyFields)
	}

	var req dtos.ResolveAwakeableRequest
	if err := h.BindJSON(w, r, &req); err != nil {
		return h.SendBadRequest(w, err, []string{"body"})
	}

	awakeable, err := h.awakeableRepo.FindByID(awakeableID)
	if err != nil {
		return h.SendNotFound(w, "awakeable not found", EmptyFields)
	}

	if awakeable.Status != internalworkflow.AwakeablePending {
		return h.SendBadRequest(w, nil, []string{"awakeable is not in pending status"})
	}

	if err := h.awakeableRepo.Resolve(awakeableID, req.Data); err != nil {
		return h.SendInternalError(w, err)
	}

	// Send resolution message to the workflow handler
	handlerName := actornames.WorkflowHandlerName(awakeable.WorkflowID)
	resolvedMsg := messaging.NewAwakeableResolvedMessage(
		awakeable.WorkflowID,
		awakeableID,
		awakeable.ExecID,
		awakeable.ThreadID,
		req.Data,
	)
	if err := h.Send(gen.Atom(handlerName), resolvedMsg); err != nil {
		h.Log().Error("failed to send awakeable resolved message: %s", err)
	}

	return h.SendJSON(w, http.StatusOK, dtos.ResolveAwakeableResponse{
		WorkflowID:  awakeable.WorkflowID.String(),
		AwakeableID: awakeableID,
		Status:      "resolved",
	})
}
