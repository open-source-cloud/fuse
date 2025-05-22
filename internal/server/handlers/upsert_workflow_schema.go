package handlers

import (
	"github.com/gofiber/fiber/v3"
	"github.com/open-source-cloud/fuse/internal/repos"
	"github.com/open-source-cloud/fuse/internal/workflow"
	"net/http"
)

// NewUpsertWorkflowSchemaHandler creates a new UpsertWorkflowSchemaHandler http handler
func NewUpsertWorkflowSchemaHandler(graphFactory *workflow.GraphFactory, graphRepo repos.GraphRepo) *UpsertWorkflowSchemaHandler {
	return &UpsertWorkflowSchemaHandler{
		graphFactory: graphFactory,
		graphRepo: graphRepo,
	}
}

// UpsertWorkflowSchemaHandler fiber http handler
type UpsertWorkflowSchemaHandler struct {
	graphFactory *workflow.GraphFactory
	graphRepo repos.GraphRepo
}

// Handle handles the UpsertWorkflowSchema http endpoint
func (h *UpsertWorkflowSchemaHandler) Handle(ctx fiber.Ctx) error {
	rawJSON := ctx.Body()

	graph, err := h.graphFactory.NewGraphFromJSON(rawJSON)
	if err != nil {
		return err
	}
	err = h.graphRepo.Save(graph)
	if err != nil {
		return err
	}

	return ctx.Status(http.StatusOK).JSON(fiber.Map{})
}
