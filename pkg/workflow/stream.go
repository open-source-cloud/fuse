package workflow

// StreamChunkType defines the type of stream chunk
type StreamChunkType string

const (
	// StreamChunkData indicates a data chunk
	StreamChunkData StreamChunkType = "data"
	// StreamChunkError indicates an error chunk
	StreamChunkError StreamChunkType = "error"
	// StreamChunkDone indicates the stream is complete
	StreamChunkDone StreamChunkType = "done"
)

// StreamChunk represents a chunk of data in a stream
type StreamChunk struct {
	Type  StreamChunkType `json:"type"`
	Data  map[string]any  `json:"data,omitempty"`
	Error string          `json:"error,omitempty"`
}

// NewStreamChunkData creates a new data stream chunk
func NewStreamChunkData(data map[string]any) StreamChunk {
	return StreamChunk{
		Type: StreamChunkData,
		Data: data,
	}
}

// NewStreamChunkError creates a new error stream chunk
func NewStreamChunkError(err error) StreamChunk {
	return StreamChunk{
		Type:  StreamChunkError,
		Error: err.Error(),
	}
}

// NewStreamChunkDone creates a new done stream chunk
func NewStreamChunkDone() StreamChunk {
	return StreamChunk{
		Type: StreamChunkDone,
	}
}

// StreamCallback is a function that receives stream chunks
type StreamCallback func(chunk StreamChunk) error
