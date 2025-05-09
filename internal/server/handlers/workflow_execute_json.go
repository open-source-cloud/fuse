package handlers

import (
	"github.com/gofiber/fiber/v3"
	"github.com/open-source-cloud/fuse/internal/messaging"
	"net/http"
)

type WorkflowExecuteJSONHandler struct {
	messageChan chan<- any
}

func NewWorkflowExecuteJSONHandler(messageChan chan<- any) *WorkflowExecuteJSONHandler {
	return &WorkflowExecuteJSONHandler{messageChan: messageChan}
}

func (h *WorkflowExecuteJSONHandler) Handle(ctx fiber.Ctx) error {
	rawJSON := ctx.Body()
	h.messageChan <- messaging.NewWorkflowExecuteJSONMessage(rawJSON)

	return ctx.Status(http.StatusOK).JSON(fiber.Map{})
}
