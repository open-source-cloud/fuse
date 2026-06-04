package ai

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/open-source-cloud/fuse/pkg/llm"
	"github.com/open-source-cloud/fuse/pkg/workflow"
)

// respondToolName is the synthetic tool the model is forced to call to deliver structured output.
const respondToolName = "respond"

// structuredRepairAttempts bounds how many times structuredOutput re-asks on invalid output.
const structuredRepairAttempts = 2

// parseOutputSchema reads the optional `outputSchema` input (a list of ParameterSchema-shaped
// objects) into typed schema params. Absent/invalid returns nil (free-form output).
func parseOutputSchema(input *workflow.FunctionInput) []workflow.ParameterSchema {
	raw := input.GetAnySliceOrDefault("outputSchema", nil)
	if len(raw) == 0 {
		return nil
	}
	b, err := json.Marshal(raw)
	if err != nil {
		return nil
	}
	var schema []workflow.ParameterSchema
	if json.Unmarshal(b, &schema) != nil {
		return nil
	}
	return schema
}

// structuredOutput coerces a completion into a typed object matching schema (ADR-0030). It offers a
// single forced "respond" tool whose parameters are the JSON Schema of the output, parses the tool
// call's arguments, and validates them; on invalid/malformed output it re-asks once. Tokens are
// recorded via usage. This tool-forced path works on every provider (all support tool calling);
// provider-native response_format is a deferred optimization.
func structuredOutput(ctx context.Context, provider llm.Provider, model string, messages []llm.Message, schema []workflow.ParameterSchema, usage UsageRecorder, function string) (map[string]any, llm.Usage, error) {
	respondTool := llm.Tool{
		Name:        respondToolName,
		Description: "Return the final answer as JSON arguments matching the required schema.",
		Parameters:  ParameterSchemaToJSONSchema(schema),
	}

	convo := messages
	var total llm.Usage
	var lastErr error
	for attempt := 0; attempt < structuredRepairAttempts; attempt++ {
		resp, err := provider.Chat(ctx, llm.ChatRequest{
			Model:      model,
			Messages:   convo,
			Tools:      []llm.Tool{respondTool},
			ToolChoice: "required",
		})
		if err != nil {
			usage.RecordCall(function, provider.Name(), model, "error")
			return nil, total, fmt.Errorf("structured output call failed: %w", err)
		}
		usage.RecordCall(function, provider.Name(), model, "success")
		usage.RecordUsage(function, provider.Name(), model, resp.Usage)
		total.PromptTokens += resp.Usage.PromptTokens
		total.CompletionTokens += resp.Usage.CompletionTokens
		total.TotalTokens += resp.Usage.TotalTokens

		obj, perr := parseRespond(resp.Message)
		if perr != nil {
			lastErr = perr
		} else if verr := validateAgainstSchema(obj, schema); verr != nil {
			lastErr = verr
		} else {
			return obj, total, nil
		}
		// Repair: re-ask from the ORIGINAL messages plus a note (avoids a dangling tool_call turn).
		convo = append(append([]llm.Message{}, messages...), llm.Message{
			Role:    llm.RoleUser,
			Content: fmt.Sprintf("The previous response was not valid (%v). Call the %q tool again with arguments that match the schema.", lastErr, respondToolName),
		})
	}
	return nil, total, fmt.Errorf("structured output did not match the schema after %d attempts: %w", structuredRepairAttempts, lastErr)
}

// parseRespond extracts the structured object from a model turn: preferring the `respond` tool
// call's arguments, falling back to parsing the message content as a JSON object.
func parseRespond(msg llm.Message) (map[string]any, error) {
	for _, tc := range msg.ToolCalls {
		if tc.Name == respondToolName || len(msg.ToolCalls) == 1 {
			var obj map[string]any
			if err := json.Unmarshal(tc.Arguments, &obj); err != nil {
				return nil, fmt.Errorf("tool arguments are not a JSON object: %w", err)
			}
			return obj, nil
		}
	}
	if msg.Content != "" {
		var obj map[string]any
		if err := json.Unmarshal([]byte(msg.Content), &obj); err == nil {
			return obj, nil
		}
	}
	return nil, fmt.Errorf("model did not return structured output")
}

// validateAgainstSchema checks required fields are present and that present fields are of a
// compatible JSON type. It is intentionally lenient (a guard, not a full JSON-Schema validator).
func validateAgainstSchema(obj map[string]any, schema []workflow.ParameterSchema) error {
	for _, p := range schema {
		v, ok := obj[p.Name]
		if !ok {
			if p.Required {
				return fmt.Errorf("missing required field %q", p.Name)
			}
			continue
		}
		if !valueMatchesType(v, p.Type) {
			return fmt.Errorf("field %q has wrong type (want %s)", p.Name, p.Type)
		}
	}
	return nil
}

// valueMatchesType reports whether a JSON-decoded value is compatible with a FUSE schema type.
func valueMatchesType(v any, t string) bool {
	switch t {
	case "", "any":
		return true
	case "string":
		_, ok := v.(string)
		return ok
	case "bool":
		_, ok := v.(bool)
		return ok
	case "int", "float":
		_, ok := v.(float64) // JSON numbers decode to float64
		return ok
	case "array":
		_, ok := v.([]any)
		return ok
	case "map", "object":
		_, ok := v.(map[string]any)
		return ok
	default:
		return true
	}
}
