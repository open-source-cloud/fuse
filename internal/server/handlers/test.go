package handlers
//
import (
	"github.com/gofiber/fiber/v3"
	"net/http"
)

type TestHandler struct {
}

func NewTestHandler() *TestHandler {
	return &TestHandler{}
}

func (hc *TestHandler) Handle(ctx fiber.Ctx) error {
	return ctx.Status(http.StatusOK).JSON(fiber.Map{
		"status": "ok",
		"services": map[string]string{
			"engine": "ok",
		},
	})
}
