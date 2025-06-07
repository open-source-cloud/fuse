package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"ergo.services/ergo/gen"
)

const (
	BadRequest          = "BAD_REQUEST"
	InternalServerError = "INTERNAL_SERVER_ERROR"
)

// HandlerFactory defines the factory type that all Handler Factories must implement
type HandlerFactory[T gen.ProcessBehavior] struct {
	Factory func() gen.ProcessBehavior
}

// Response is the type for all responses
type Response = map[string]any

// BindJSON binds a JSON request to the given struct
func BindJSON(w http.ResponseWriter, r *http.Request, v any) error {
	return json.NewDecoder(r.Body).Decode(v)
}

// SendJSON sends a JSON response to the client
func SendJSON(w http.ResponseWriter, status int, v any) error {
	w.WriteHeader(status)
	w.Header().Set("Content-Type", "application/json")
	return json.NewEncoder(w).Encode(v)
}

type Worker struct {
	WorkerName string
	Pattern    string
	Timeout    time.Duration
}

func Workers() []Worker {
	return []Worker{
		{
			WorkerName: TriggerWorkflowHandlerName,
			Pattern:    "/v1/workflows/{workflowID}/trigger",
			Timeout:    10 * time.Second,
		},
		{
			WorkerName: AsyncFunctionResultHandlerName,
			Pattern:    "/v1/workflows/{workflowID}/async-function-result",
			Timeout:    10 * time.Second,
		},
		{
			WorkerName: UpsertWorkflowSchemaHandlerName,
			Pattern:    "/v1/schemas/{schemaID}",
			Timeout:    10 * time.Second,
		},
	}
}
