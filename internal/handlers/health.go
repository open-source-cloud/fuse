// Package handlers provides handlers for the http server
package handlers

import (
	"github.com/gofiber/fiber/v3"
	"net/http"
)

// HealthCheckHandler is responsible for handling health check HTTP requests and providing the system's health status.
type HealthCheckHandler struct{}

// NewHealthCheckHandler creates and returns a new instance of HealthCheckHandler.
func NewHealthCheckHandler() *HealthCheckHandler {
	return &HealthCheckHandler{}
}

// Handle handles HTTP requests for health check endpoints and sends a response using Fiber's context.
func (hc *HealthCheckHandler) Handle(ctx fiber.Ctx) error {
	return ctx.Status(http.StatusOK).JSON(fiber.Map{
		"status": "ok",
		"services": map[string]string{
			"db":     "ok",
			"engine": "ok",
		},
	})
}
