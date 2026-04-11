package handlers

import (
	"net/http"
	"strconv"
	"time"

	"ergo.services/ergo/gen"
	"github.com/open-source-cloud/fuse/internal/dtos"
	"github.com/open-source-cloud/fuse/internal/repositories"
)

type (
	// SchemaTracesHandler is the handler for listing traces by schema
	SchemaTracesHandler struct {
		Handler
		traceRepo repositories.TraceRepository
	}
	// SchemaTracesHandlerFactory is a factory for creating SchemaTracesHandler actors
	SchemaTracesHandlerFactory HandlerFactory[*SchemaTracesHandler]
)

const (
	// SchemaTracesHandlerName is the name of the SchemaTracesHandler actor
	SchemaTracesHandlerName = "schema_traces_handler"
	// SchemaTracesHandlerPoolName is the name of the SchemaTracesHandler pool
	SchemaTracesHandlerPoolName = "schema_traces_handler_pool"
)

// NewSchemaTracesHandlerFactory creates a new SchemaTracesHandlerFactory
func NewSchemaTracesHandlerFactory(traceRepo repositories.TraceRepository) *SchemaTracesHandlerFactory {
	return &SchemaTracesHandlerFactory{
		Factory: func() gen.ProcessBehavior {
			return &SchemaTracesHandler{
				traceRepo: traceRepo,
			}
		},
	}
}

// HandleGet handles the list schema traces endpoint (GET /v1/schemas/{schemaID}/traces)
// @Summary List execution traces for a schema
// @Description Returns paginated execution traces for all executions of a schema
// @Tags schemas
// @Produce json
// @Param schemaID path string true "Schema ID"
// @Param limit query int false "Max results (default 50)"
// @Param offset query int false "Pagination offset"
// @Param status query string false "Filter by status"
// @Param since query string false "Only traces after this time (RFC3339)"
// @Success 200 {object} dtos.TraceListResponse
// @Failure 400 {object} dtos.BadRequestError
// @Router /v1/schemas/{schemaID}/traces [get]
func (h *SchemaTracesHandler) HandleGet(_ gen.PID, w http.ResponseWriter, r *http.Request) error {
	schemaID, err := h.GetPathParam(r, "schemaID")
	if err != nil {
		return h.SendBadRequest(w, err, EmptyFields)
	}

	opts := repositories.TraceQueryOpts{}

	if limitStr, qErr := h.GetQueryParam(r, "limit"); qErr == nil {
		if limit, pErr := strconv.Atoi(limitStr); pErr == nil {
			opts.Limit = limit
		}
	}
	if offsetStr, qErr := h.GetQueryParam(r, "offset"); qErr == nil {
		if offset, pErr := strconv.Atoi(offsetStr); pErr == nil {
			opts.Offset = offset
		}
	}
	if status, qErr := h.GetQueryParam(r, "status"); qErr == nil {
		opts.Status = &status
	}
	if sinceStr, qErr := h.GetQueryParam(r, "since"); qErr == nil {
		if since, pErr := time.Parse(time.RFC3339, sinceStr); pErr == nil {
			opts.Since = &since
		}
	}

	result, err := h.traceRepo.FindBySchemaID(schemaID, opts)
	if err != nil {
		return h.SendInternalError(w, err)
	}

	limit := opts.Limit
	if limit <= 0 {
		limit = 50
	}

	return h.SendJSON(w, http.StatusOK, dtos.TraceListResponse{
		Traces: result.Traces,
		Total:  result.Total,
		Limit:  limit,
		Offset: opts.Offset,
	})
}
