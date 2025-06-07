// Package handlers HTTP Fiber handles
package handlers

import (
	"fmt"
	"net/http"

	"ergo.services/ergo/act"
	"ergo.services/ergo/gen"
	"github.com/open-source-cloud/fuse/internal/messaging"
	"github.com/open-source-cloud/fuse/pkg/workflow"
)

type (
	// AsyncFunctionRequest is the request body for the AsyncFunctionHandler
	AsyncFunctionRequest struct {
		ExecID string                  `json:"execID"`
		Result workflow.FunctionOutput `json:"result"`
	}
	// AsyncFunctionHandler Fiber http handler
	AsyncFunctionHandler struct {
		act.WebWorker
	}
)

// AsyncFunctionResultHandlerName is the name of the AsyncFunctionResultHandler actor
const AsyncFunctionResultHandlerName = "async_function_result_handler"

// AsyncFunctionResultHandlerFactory is a factory for creating AsyncFunctionHandler actors
type AsyncFunctionResultHandlerFactory HandlerFactory[*AsyncFunctionHandler]

// NewAsyncFunctionResultHandlerFactory creates a new AsyncFunctionResultHandlerFactory
func NewAsyncFunctionResultHandlerFactory() *AsyncFunctionResultHandlerFactory {
	return &AsyncFunctionResultHandlerFactory{
		Factory: func() gen.ProcessBehavior {
			return &AsyncFunctionHandler{}
		},
	}
}

// HandlePost handles the http TriggerWorkflow endpoint
// POST /v1/workflows/{workflowID}/async-function-result
func (h *AsyncFunctionHandler) HandlePost(w http.ResponseWriter, r *http.Request) error {
	workflowID := r.URL.Query().Get("workflowID")
	var req AsyncFunctionRequest
	if err := BindJSON(w, r, &req); err != nil {
		return SendJSON(w, http.StatusBadRequest, Response{
			"message": fmt.Sprintf("invalid request: %s", err),
			"code":    BadRequest,
		})
	}

	if req.ExecID == "" {
		return SendJSON(w, http.StatusBadRequest, Response{
			"message": "execID is required",
			"code":    BadRequest,
		})
	}

	err := h.Send(workflowID, messaging.NewAsyncFunctionResultMessage(workflowID, req.ExecID, req.Result))
	if err != nil {
		return SendJSON(w, http.StatusInternalServerError, Response{
			"message": fmt.Sprintf("failed to send message: %s", err),
			"code":    InternalServerError,
		})
	}

	return SendJSON(w, http.StatusOK, Response{
		"workflowID": workflowID,
		"execID":     req.ExecID,
		"code":       "OK",
	})
}
