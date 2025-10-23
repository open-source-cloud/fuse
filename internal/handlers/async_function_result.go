// Package handlers HTTP Fiber handles
package handlers

import (
	"net/http"

	"github.com/open-source-cloud/fuse/internal/actors/actornames"

	"ergo.services/ergo/gen"
	"github.com/open-source-cloud/fuse/internal/dtos"
	"github.com/open-source-cloud/fuse/internal/messaging"
	"github.com/open-source-cloud/fuse/pkg/workflow"
)

type (
	// AsyncFunctionHandler Fiber http handler
	AsyncFunctionHandler struct {
		Handler
	}
	// AsyncFunctionResultHandlerFactory is a factory for creating AsyncFunctionHandler actors
	AsyncFunctionResultHandlerFactory HandlerFactory[*AsyncFunctionHandler]
)

const (
	// AsyncFunctionResultHandlerName is the name of the AsyncFunctionResultHandler actor
	AsyncFunctionResultHandlerName = "async_function_result_handler"
	// AsyncFunctionResultHandlerPoolName is the name of the AsyncFunctionResultHandler pool
	AsyncFunctionResultHandlerPoolName = "async_function_result_handler_pool"
)

// NewAsyncFunctionResultHandlerFactory creates a new AsyncFunctionResultHandlerFactory
func NewAsyncFunctionResultHandlerFactory() *AsyncFunctionResultHandlerFactory {
	return &AsyncFunctionResultHandlerFactory{
		Factory: func() gen.ProcessBehavior {
			return &AsyncFunctionHandler{}
		},
	}
}

// HandlePost handles the http AsyncFunctionResult endpoint (POST /v1/workflows/{workflowID}/execs/{execID})
// @Summary Submit async function result
// @Description Submit the result of an async function execution
// @Tags workflows
// @Accept json
// @Produce json
// @Param workflowID path string true "Workflow ID"
// @Param execID path string true "Execution ID"
// @Param result body dtos.AsyncFunctionRequest true "Function Result"
// @Success 200 {object} dtos.AsyncFunctionResultResponse
// @Failure 400 {object} dtos.BadRequestError
// @Failure 500 {object} dtos.InternalServerErrorResponse
// @Router /v1/workflows/{workflowID}/execs/{execID} [post]
func (h *AsyncFunctionHandler) HandlePost(from gen.PID, w http.ResponseWriter, r *http.Request) error {
	h.Log().Info("received async functionRequestData result from: %v remoteAddr: %s", from, r.RemoteAddr)

	strWorkflowID, err := h.GetPathParam(r, "workflowID")
	if err != nil {
		return h.SendBadRequest(w, err, []string{"workflowID"})
	}
	workflowID := workflow.ID(strWorkflowID)

	strExecID, err := h.GetPathParam(r, "execID")
	if err != nil {
		return h.SendBadRequest(w, err, []string{"execID"})
	}
	execID := workflow.ExecID(strExecID)

	var req dtos.AsyncFunctionRequest
	if err := h.BindJSON(w, r, &req); err != nil {
		return h.SendBadRequest(w, err, []string{"body"})
	}

	if err = h.Send(
		actornames.WorkflowHandlerName(workflowID),
		messaging.NewAsyncFunctionResultMessage(workflowID, execID, req.Result),
	); err != nil {
		return h.SendInternalError(w, err)
	}

	return h.SendJSON(w, http.StatusOK, dtos.AsyncFunctionResultResponse{
		WorkflowID: workflowID.String(),
		ExecID:     execID.String(),
		Code:       "OK",
	})
}
