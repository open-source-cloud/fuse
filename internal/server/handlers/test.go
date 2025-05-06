package handlers
//
import (
	"github.com/gofiber/fiber/v3"
	"net/http"
)

type TestHandler struct {
	messageChan chan<- any
}

func NewTestHandler(messageChan chan<- any) *TestHandler {
	return &TestHandler{messageChan: messageChan}
}

func (hc *TestHandler) Handle(ctx fiber.Ctx) error {
	hc.messageChan <- "test"

	return ctx.Status(http.StatusOK).JSON(fiber.Map{
		"status": "ok",
		"services": map[string]string{
			"engine": "ok",
		},
	})
}
