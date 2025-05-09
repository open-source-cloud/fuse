package handlers

import (
	"github.com/gofiber/fiber/v3"
	"github.com/open-source-cloud/fuse/internal/repos"
	"github.com/open-source-cloud/fuse/internal/workflow"
	"net/http"
)

func NewUpsertWorkflowSchemaHandler(graphRepo repos.GraphRepo) *UpsertWorkflowSchemaHandler {
	return &UpsertWorkflowSchemaHandler{
		graphRepo: graphRepo,
	}
}

type UpsertWorkflowSchemaHandler struct {
	graphRepo repos.GraphRepo
}

func (h *UpsertWorkflowSchemaHandler) Handle(ctx fiber.Ctx) error {
	rawJSON := ctx.Body()

	graph, err := workflow.NewGraphFromJSON(rawJSON)
	if err != nil {
		return err
	}
	err = h.graphRepo.Save(graph)
	if err != nil {
		return err
	}

	return ctx.Status(http.StatusOK).JSON(fiber.Map{})
}
