package handlers

import (
	"context"
	"fmt"
	"net/http"

	"ergo.services/ergo/gen"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/open-source-cloud/fuse/internal/dtos"
	"github.com/open-source-cloud/fuse/internal/readiness"
)

const (
	// ReadinessHandlerName is the registered name of the readiness handler.
	ReadinessHandlerName = "readiness_handler"
	// ReadinessHandlerPoolName is the registered name of the readiness handler pool.
	ReadinessHandlerPoolName = "readiness_handler_pool"
)

// ReadinessHandlerFactory is the factory for the readiness handler.
type ReadinessHandlerFactory HandlerFactory[*ReadinessHandler]

// ReadinessHandler handles GET /readyz — returns 200 only when the node is ready to serve traffic.
// Readiness checks: actor system initialization, database connectivity (when using the PostgreSQL driver).
type ReadinessHandler struct {
	Handler
	pool          *pgxpool.Pool   // nil when using the memory driver
	readinessFlag *readiness.Flag // signals whether actors are fully started
}

// NewReadinessHandlerFactory creates a new ReadinessHandlerFactory.
// pool may be nil when DB_DRIVER != postgres; the handler is still ready in that case.
func NewReadinessHandlerFactory(pool *pgxpool.Pool, flag *readiness.Flag) *ReadinessHandlerFactory {
	return &ReadinessHandlerFactory{
		Factory: func() gen.ProcessBehavior {
			return &ReadinessHandler{pool: pool, readinessFlag: flag}
		},
	}
}

// HandleGet handles GET /readyz.
// @Summary Readiness probe
// @Description Returns 200 when the node is ready to serve traffic, 503 when a dependency is unhealthy.
// @Tags health
// @Produce json
// @Success 200 {object} dtos.ReadinessResponse
// @Failure 503 {object} dtos.ReadinessResponse
// @Router /readyz [get]
func (h *ReadinessHandler) HandleGet(_ gen.PID, w http.ResponseWriter, _ *http.Request) error {
	resp := h.checkReadiness()
	status := http.StatusOK
	if resp.Status != "ready" {
		status = http.StatusServiceUnavailable
	}
	return h.SendJSON(w, status, resp)
}

func (h *ReadinessHandler) checkReadiness() dtos.ReadinessResponse {
	resp := dtos.ReadinessResponse{
		Status: "ready",
		Checks: make(map[string]string),
	}

	// Check actor system readiness
	if h.readinessFlag.IsNotReady() {
		resp.Status = "not_ready"
		resp.Checks["actors"] = "initializing"
		return resp
	}

	resp.Checks["actors"] = "ok"

	if h.pool != nil {
		if err := h.pool.Ping(context.Background()); err != nil {
			resp.Status = "not_ready"
			resp.Checks["database"] = fmt.Sprintf("error: %s", err)
		} else {
			resp.Checks["database"] = "ok"
		}
	}
	return resp
}
