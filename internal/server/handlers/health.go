// Package handlers provides handlers for the http server
package handlers

import (
	"github.com/gofiber/fiber/v3"
	"github.com/open-source-cloud/fuse/internal/database"
	"net/http"
)

// HealthCheckHandler is responsible for handling health check HTTP requests and providing the system's health status.
type HealthCheckHandler struct {
	db *database.ArangoClient
}

// NewHealthCheckHandler creates and returns a new instance of HealthCheckHandler.
func NewHealthCheckHandler(db *database.ArangoClient) *HealthCheckHandler {
	return &HealthCheckHandler{
		db,
	}
}

// Handle handles HTTP requests for health check endpoints and sends a response using Fiber's context.
func (hc *HealthCheckHandler) Handle(ctx fiber.Ctx) error {
	var dbStatus string = "ok"
	if err := hc.db.Ping(); err != nil {
		dbStatus = "error"
	}

	return ctx.Status(http.StatusOK).JSON(fiber.Map{
		"status": "ok",
		"services": map[string]string{
			"db":     dbStatus,
			"engine": "ok",
		},
	})
}
