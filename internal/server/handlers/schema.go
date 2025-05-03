package handlers

import (
	"github.com/gofiber/fiber/v3"
	"github.com/open-source-cloud/fuse/internal/graph"
	"net/http"
)

// SchemaHandler is responsible for handling schema-related HTTP requests.
type SchemaHandler struct {
}

// NewSchemaHandler creates and returns a new instance of SchemaHandler.
func NewSchemaHandler() *SchemaHandler {
	return &SchemaHandler{}
}

// Handle processes the schema-related operations associated with the SchemaHandler instance.
// GET /v1/schema
func (sh *SchemaHandler) Handle(ctx fiber.Ctx) error {
	schema := graph.JSONSchema()
	return ctx.Status(http.StatusOK).JSON(schema)
}
