package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

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
// SendJSON sends a JSON response to the client
func SendJSON(w http.ResponseWriter, status int, v Response) error {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Accept", "application/json")

	body, err := json.Marshal(v)
	if err != nil {
		log.Println("failed to marshal response", v, err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(fmt.Sprintf(`{"message": "%s", "code": "%s"}`, err.Error(), InternalServerError)))
		return err
	}

	w.WriteHeader(status) // Move this before w.Write
	_, err = w.Write(body)
	if err != nil {
		log.Println("failed to write response", err)
		return err
	}

	return nil
}
