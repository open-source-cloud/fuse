package handlers

import (
	"github.com/gofiber/fiber/v3"
	"github.com/open-source-cloud/fuse/internal/repos"
	"github.com/open-source-cloud/fuse/internal/workflow"
	"net/http"
)

func NewUpsertWorkflowSchemaHandler(graphFactory *workflow.GraphFactory, graphRepo repos.GraphRepo) *UpsertWorkflowSchemaHandler {
	return &UpsertWorkflowSchemaHandler{
		graphFactory: graphFactory,
		graphRepo: graphRepo,
	}
}

type UpsertWorkflowSchemaHandler struct {
	graphFactory *workflow.GraphFactory
	graphRepo repos.GraphRepo
}

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
