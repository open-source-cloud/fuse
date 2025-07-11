// Package handlers HTTP Fiber handles
package handlers

import (
	"fmt"
	"github.com/open-source-cloud/fuse/internal/actors/actornames"
	"net/http"

	"ergo.services/ergo/gen"
	"github.com/open-source-cloud/fuse/internal/messaging"
	"github.com/open-source-cloud/fuse/pkg/workflow"
)

type (
	// AsyncFunctionRequest is the request body for the AsyncFunctionHandler
	AsyncFunctionRequest struct {
		Result workflow.FunctionOutput `json:"result"`
	}
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
func (h *AsyncFunctionHandler) HandlePost(from gen.PID, w http.ResponseWriter, r *http.Request) error {
	h.Log().Info("received async functionRequestData result from: %v remoteAddr: %s", from, r.RemoteAddr)

	strWorkflowID, err := h.GetPathParam(r, "workflowID")
	if err != nil {
		return h.SendJSON(w, http.StatusBadRequest, Response{
			"message": fmt.Sprintf("invalid request: %s", err),
			"code":    BadRequest,
		})
	}
	workflowID := workflow.ID(strWorkflowID)

	strExecID, err := h.GetPathParam(r, "execID")
	if err != nil {
		return h.SendJSON(w, http.StatusBadRequest, Response{
			"message": fmt.Sprintf("invalid request: %s", err),
			"code":    BadRequest,
		})
	}
	execID := workflow.ExecID(strExecID)

	var req AsyncFunctionRequest
	if err := h.BindJSON(w, r, &req); err != nil {
		return h.SendJSON(w, http.StatusBadRequest, Response{
			"message": fmt.Sprintf("invalid request: %s", err),
			"code":    BadRequest,
		})
	}

	if err = h.Send(
		actornames.WorkflowHandlerName(workflowID),
		messaging.NewAsyncFunctionResultMessage(workflowID, execID, req.Result),
	); err != nil {
		return h.SendJSON(w, http.StatusInternalServerError, Response{
			"message": fmt.Sprintf("failed to send message: %s", err),
			"code":    InternalServerError,
		})
	}

	return h.SendJSON(w, http.StatusOK, Response{
		"workflowID": workflowID,
		"execID":     execID,
		"code":       "OK",
	})
}
