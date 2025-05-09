package handlers

import (
	"github.com/gofiber/fiber/v3"
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
			"error": err,
		})
	}
	return ctx.Status(http.StatusOK).JSON(fiber.Map{})
}
