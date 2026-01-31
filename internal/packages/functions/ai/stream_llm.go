package ai

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/open-source-cloud/fuse/internal/packages/transport"
	"github.com/open-source-cloud/fuse/pkg/workflow"
	"github.com/rs/zerolog/log"
)

// StreamLLMFunctionID is the id of the stream LLM function
const StreamLLMFunctionID = "stream_llm"

// StreamLLMFunctionMetadata returns the metadata for the stream LLM function
func StreamLLMFunctionMetadata() workflow.FunctionMetadata {
	return workflow.FunctionMetadata{
		Transport: transport.Internal,
		Input: workflow.InputMetadata{
			CustomParameters: false,
			Parameters: []workflow.ParameterSchema{
				{
					Name:        "prompt",
					Type:        "string",
					Required:    true,
					Description: "The prompt to send to the LLM",
				},
				{
					Name:        "model",
					Type:        "string",
					Required:    false,
					Description: "The LLM model to use (default: 'default')",
					Default:     "default",
				},
				{
					Name:        "temperature",
					Type:        "float",
					Required:    false,
					Description: "The temperature for the LLM (default: 0.7)",
					Default:     0.7,
				},
			},
		},
		Output: workflow.OutputMetadata{
			Parameters: []workflow.ParameterSchema{
				{
					Name:        "response",
					Type:        "string",
					Description: "The complete LLM response",
				},
				{
					Name:        "chunks",
					Type:        "int",
					Description: "The number of chunks received",
				},
			},
		},
	}
}

// StreamLLMFunction simulates streaming LLM responses
// This is an example implementation that demonstrates how to use streaming callbacks
func StreamLLMFunction(execInfo *workflow.ExecutionInfo) (workflow.FunctionResult, error) {
	prompt := execInfo.Input.GetStr("prompt")
	if prompt == "" {
		return workflow.NewFunctionResultError(errors.New("prompt is required"))
	}

	model := execInfo.Input.GetStr("model")
	if model == "" {
		model = "default"
	}

	// If streaming callback is provided, use it
	if execInfo.StreamCallback != nil {
		return streamLLMResponse(execInfo, prompt, model)
	}

	// Fallback to non-streaming response
	log.Info().Str("model", model).Str("prompt", prompt).Msg("executing LLM (non-streaming)")
	response := simulateLLMResponse(prompt)
	return workflow.NewFunctionResultSuccessWith(map[string]any{
		"response": response,
		"chunks":   1,
	}), nil
}

// streamLLMResponse streams LLM response chunks using the callback
func streamLLMResponse(execInfo *workflow.ExecutionInfo, prompt, model string) (workflow.FunctionResult, error) {
	log.Info().Str("model", model).Str("prompt", prompt).Msg("executing LLM (streaming)")

	// Simulate streaming response by splitting into chunks
	response := simulateLLMResponse(prompt)
	words := strings.Fields(response)
	chunkSize := 3 // Send 3 words per chunk

	var chunks []string
	var fullResponse strings.Builder
	chunkCount := 0

	for i := 0; i < len(words); i += chunkSize {
		end := i + chunkSize
		if end > len(words) {
			end = len(words)
		}
		chunk := strings.Join(words[i:end], " ")
		chunks = append(chunks, chunk)
		fullResponse.WriteString(chunk)
		if i+chunkSize < len(words) {
			fullResponse.WriteString(" ")
		}
	}

	// Stream chunks with delay to simulate real streaming
	ctx := context.Background()
	for i, chunk := range chunks {
		chunkCount++
		chunkData := workflow.NewStreamChunkData(map[string]any{
			"chunk":        chunk,
			"chunk_index":  i,
			"total_chunks": len(chunks),
		})

		if err := execInfo.StreamCallback(chunkData); err != nil {
			log.Error().Err(err).Msg("failed to stream chunk")
			return workflow.NewFunctionResultError(fmt.Errorf("streaming failed: %w", err))
		}

		// Simulate network delay
		select {
		case <-ctx.Done():
			return workflow.NewFunctionResultError(ctx.Err())
		case <-time.After(50 * time.Millisecond):
			// Continue
		}
	}

	// Send done chunk
	doneChunk := workflow.NewStreamChunkDone()
	if err := execInfo.StreamCallback(doneChunk); err != nil {
		log.Error().Err(err).Msg("failed to send done chunk")
	}

	return workflow.NewFunctionResultSuccessWith(map[string]any{
		"response": fullResponse.String(),
		"chunks":   chunkCount,
	}), nil
}

// simulateLLMResponse generates a mock LLM response
func simulateLLMResponse(prompt string) string {
	return fmt.Sprintf("This is a simulated response to: %s. The AI agent has processed your request and generated this response.", prompt)
}
