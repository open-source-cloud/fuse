// Package handlers HTTP Fiber handles
package handlers

import (
	"fmt"
	"github.com/gofiber/fiber/v3"
	"github.com/open-source-cloud/fuse/internal/messaging"
	"github.com/open-source-cloud/fuse/pkg/workflow"
	"net/http"
)

// NewAsyncFunctionResultHandler creates a new TriggerWorkflowHandler http handler
func NewAsyncFunctionResultHandler(messageChan chan<- any) *AsyncFunctionHandler {
	return &AsyncFunctionHandler{messageChan: messageChan}
}

type (
	// AsyncFunctionHandler Fiber http handler
	AsyncFunctionHandler struct {
		messageChan chan<- any
	}

	asyncFunctionRequest struct {
		ExecID string                  `json:"execID"`
		Result workflow.FunctionOutput `json:"result"`
	}
)

// Handle handles the http TriggerWorkflow endpoint
func (h *AsyncFunctionHandler) Handle(ctx fiber.Ctx) error {
	workflowID := ctx.Params("workflowID")
	var req asyncFunctionRequest
	if err := ctx.Bind().Body(&req); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fmt.Sprintf("invalid request: %s", err),
		})
	}

	if req.ExecID == "" {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "execID is required",
		})
	}

	h.messageChan <- messaging.NewAsyncFunctionResultMessage(workflowID, req.ExecID, req.Result)
	return ctx.Status(http.StatusOK).JSON(fiber.Map{})
}
