package workflow

import "context"

// AIAgentPackage defines the interface for AI agent packages that support streaming
type AIAgentPackage interface {
	Package
	// StreamResponse streams AI agent responses using the provided callback
	// The callback is called for each chunk of the streaming response
	StreamResponse(ctx context.Context, input *FunctionInput, callback StreamCallback) error
	// SupportsStreaming returns true if this package supports streaming responses
	SupportsStreaming() bool
}
