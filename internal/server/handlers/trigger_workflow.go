package handlers

import (
	"fmt"
	"github.com/gofiber/fiber/v3"
	"github.com/open-source-cloud/fuse/internal/messaging"
	"net/http"
)

func NewTriggerWorkflowHandler(messageChan chan<- any) *TriggerWorkflowHandler {
	return &TriggerWorkflowHandler{messageChan: messageChan}
}

type (
	TriggerWorkflowHandler struct {
		messageChan chan<- any
	}

	triggerWorkflowRequest struct {
		SchemaID   string `json:"schemaId,omitempty"`
	}
)

func (h *TriggerWorkflowHandler) Handle(ctx fiber.Ctx) error {
	var req triggerWorkflowRequest
	if err := ctx.Bind().Body(&req); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fmt.Sprintf("invalid request: %s", err),
		})
	}

	if req.SchemaID == "" {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "schemaId is required",
		})
	}

	h.messageChan <- messaging.NewTriggerWorkflowMessage(req.SchemaID)
	return ctx.Status(http.StatusOK).JSON(fiber.Map{})
}
