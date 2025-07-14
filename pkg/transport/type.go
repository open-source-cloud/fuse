// Package transport communication between core<->package-providers
package transport

// Type defines the type of transport driver to use for core<->package-provider communication
type Type string

// default supported types
const (
	HTTP Type = "http"
	gRPC Type = "grpc"
)
