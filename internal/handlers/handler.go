package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"

	"ergo.services/ergo/act"
	"ergo.services/ergo/gen"
	"github.com/gorilla/mux"
)

var (
	// ErrQueryParamNotFound is returned when a query param is not found
	ErrQueryParamNotFound = errors.New("query param not found")
	// ErrQueryParamEmpty is returned when a query param is empty
	ErrQueryParamEmpty = errors.New("query param is empty")
	// ErrPathParamNotFound is returned when a path param is not found
	ErrPathParamNotFound = errors.New("path param not found")
)

const (
	// BadRequest is the error code for bad requests
	BadRequest = "BAD_REQUEST"
	// InternalServerError is the error code for internal server errors
	InternalServerError = "INTERNAL_SERVER_ERROR"
)

type (
	// HandlerFactory defines the factory type that all Handler Factories must implement
	HandlerFactory[T gen.ProcessBehavior] struct {
		Factory func() gen.ProcessBehavior
	}
	// Handler is the base handler for all handlers that implement the WebWorker interface from Ergo
	// It provides a base implementation for all handlers that need to interact with the HTTP server
	Handler struct {
		act.WebWorker
	}
	// Response is the type for all responses
	Response = map[string]any
)

// BindJSON binds a JSON request to the given struct
func (h *Handler) BindJSON(_ http.ResponseWriter, r *http.Request, v any) error {
	return json.NewDecoder(r.Body).Decode(v)
}

// SendJSON sends a JSON response to the client
func (h *Handler) SendJSON(w http.ResponseWriter, status int, v Response) error {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Accept", "application/json")

	body, err := json.Marshal(v)
	if err != nil {
		log.Println("failed to marshal response", v, err)
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = fmt.Fprintf(w, `{"message": "%s", "code": "%s"}`, err.Error(), InternalServerError)
		return err
	}

	w.WriteHeader(status)
	_, err = w.Write(body)
	if err != nil {
		log.Println("failed to write response", err)
		return err
	}

	return nil
}

// GetQueryParam gets a query param from the request
func (h *Handler) GetQueryParam(r *http.Request, key string) (string, error) {
	values, ok := r.URL.Query()[key]
	if !ok {
		return "", ErrQueryParamNotFound
	}
	if len(values) == 0 {
		return "", ErrQueryParamEmpty
	}
	return values[0], nil
}

// GetPathParam gets a path param from the request
func (h *Handler) GetPathParam(r *http.Request, key string) (string, error) {
	vars := mux.Vars(r)
	value, ok := vars[key]
	if !ok {
		return "", ErrPathParamNotFound
	}
	return value, nil
}
