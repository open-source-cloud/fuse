package handlers

import (
	"net/http"
	"strconv"
	"time"

	"ergo.services/ergo/gen"
	"github.com/open-source-cloud/fuse/internal/repositories"
)

type (
	// ListExecutionsHandler handles GET /v1/schemas/{schemaID}/executions
	ListExecutionsHandler struct {
		Handler
		workflowRepo repositories.WorkflowRepository
	}
	// ListExecutionsHandlerFactory is a factory for creating ListExecutionsHandler actors
	ListExecutionsHandlerFactory HandlerFactory[*ListExecutionsHandler]
)

const (
	// ListExecutionsHandlerName is the name of the ListExecutionsHandler actor
	ListExecutionsHandlerName = "list_executions_handler"
	// ListExecutionsHandlerPoolName is the name of the ListExecutionsHandler pool
	ListExecutionsHandlerPoolName = "list_executions_handler_pool"
)

// NewListExecutionsHandlerFactory creates a new ListExecutionsHandlerFactory
func NewListExecutionsHandlerFactory(workflowRepo repositories.WorkflowRepository) *ListExecutionsHandlerFactory {
	return &ListExecutionsHandlerFactory{
		Factory: func() gen.ProcessBehavior {
			return &ListExecutionsHandler{
				workflowRepo: workflowRepo,
			}
		},
	}
}

// HandleGet handles GET /v1/schemas/{schemaID}/executions
// @Summary List workflow executions for a schema
// @Description Returns a paginated list of workflow executions filtered by status and time range
// @Tags schemas
// @Produce json
// @Param schemaID path string true "Schema ID"
// @Param page query int false "Page number (default: 1)"
// @Param size query int false "Page size (default: 20, max: 100)"
// @Param status query string false "Filter by workflow state (e.g. finished, error, running)"
// @Param from query string false "Filter by created_at >= (RFC3339 format)"
// @Param to query string false "Filter by created_at <= (RFC3339 format)"
// @Success 200 {object} object "Paginated list with items, total, page, size, lastPage"
// @Failure 400 {object} dtos.BadRequestError
// @Failure 500 {object} dtos.InternalServerErrorResponse
// @Router /v1/schemas/{schemaID}/executions [get]
func (h *ListExecutionsHandler) HandleGet(_ gen.PID, w http.ResponseWriter, r *http.Request) error {
	schemaID, err := h.GetPathParam(r, "schemaID")
	if err != nil {
		return h.SendBadRequest(w, err, EmptyFields)
	}

	q := r.URL.Query()

	page, _ := strconv.Atoi(q.Get("page"))
	if page < 1 {
		page = 1
	}
	size, _ := strconv.Atoi(q.Get("size"))
	if size < 1 {
		size = 20
	}
	if size > 100 {
		size = 100
	}

	filter := repositories.ExecutionListFilter{
		SchemaID: schemaID,
		Status:   q.Get("status"),
		Page:     page,
		Size:     size,
	}

	if fromStr := q.Get("from"); fromStr != "" {
		t, parseErr := time.Parse(time.RFC3339, fromStr)
		if parseErr != nil {
			return h.SendBadRequest(w, parseErr, []string{"from must be in RFC3339 format"})
		}
		filter.From = t
	}
	if toStr := q.Get("to"); toStr != "" {
		t, parseErr := time.Parse(time.RFC3339, toStr)
		if parseErr != nil {
			return h.SendBadRequest(w, parseErr, []string{"to must be in RFC3339 format"})
		}
		filter.To = t
	}

	result, findErr := h.workflowRepo.FindExecutions(filter)
	if findErr != nil {
		return h.SendInternalError(w, findErr)
	}

	return h.SendJSON(w, http.StatusOK, result)
}
