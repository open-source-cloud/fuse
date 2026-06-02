package ai

import (
	"sort"
	"strings"

	"github.com/open-source-cloud/fuse/pkg/workflow"
)

// ToolRegistry is the minimal slice of the package registry that the ai/agent
// node needs to turn existing FUSE functions into LLM tools and to invoke them.
//
// It is declared here (in package ai) and implemented by an adapter in package
// packages on purpose: package packages already imports ai (to register the ai
// package), so ai must NOT import internal/packages or it would create an import
// cycle. Depending on this interface — which ai owns — keeps the dependency
// pointing the right way (packages -> ai).
type ToolRegistry interface {
	// ListTools returns the functions eligible to be exposed to the model as
	// tools. Per ADR-0007 (Phase B) this is synchronous, declared-parameter
	// functions only; asynchronous/intercepted and schemaless functions are
	// excluded by the implementation.
	ListTools() []ToolDescriptor
	// InvokeTool runs the function identified by its full id (e.g.
	// "fuse/pkg/logic/sum") synchronously in-process and returns its result inline
	// (FunctionResult.Async == false). No worker handle is involved: Phase B exposes
	// only synchronous tools, so the actor system is never reached.
	InvokeTool(functionID string, execInfo *workflow.ExecutionInfo) (workflow.FunctionResult, error)
}

// ToolDescriptor describes one function exposed to the model as a tool.
type ToolDescriptor struct {
	// FunctionID is the real, full function id, e.g. "fuse/pkg/logic/sum".
	FunctionID string
	// MangledName is the provider-safe tool name, e.g. "fuse__pkg__logic__sum".
	MangledName string
	// Description is shown to the model.
	Description string
	// Parameters is a JSON Schema object describing the tool's arguments.
	Parameters map[string]any
}

// toolNameSeparator replaces "/" in tool names because most providers restrict
// tool names to [A-Za-z0-9_-].
const toolNameSeparator = "__"

// JSON Schema primitive type names reused by the converter below.
const (
	jsonTypeObject = "object"
	jsonTypeArray  = "array"
)

// MangleToolName converts a function id into a provider-safe tool name by
// replacing "/" with "__".
func MangleToolName(functionID string) string {
	return strings.ReplaceAll(functionID, "/", toolNameSeparator)
}

// DemangleToolName is the inverse of MangleToolName.
func DemangleToolName(toolName string) string {
	return strings.ReplaceAll(toolName, toolNameSeparator, "/")
}

// ParameterSchemaToJSONSchema converts a function's declared input parameters
// into a JSON Schema "object" suitable for llm.Tool.Parameters. Parameters with
// no declared fields produce an empty-properties object so zero-arg tools remain
// callable. The "required" list is sorted for deterministic output.
func ParameterSchemaToJSONSchema(params []workflow.ParameterSchema) map[string]any {
	properties := make(map[string]any, len(params))
	required := make([]string, 0, len(params))

	for _, p := range params {
		prop := map[string]any{"type": jsonSchemaType(p.Type)}
		if p.Description != "" {
			prop["description"] = p.Description
		}
		if p.Default != nil {
			prop["default"] = p.Default
		}
		properties[p.Name] = prop
		if p.Required {
			required = append(required, p.Name)
		}
	}

	schema := map[string]any{
		"type":       jsonTypeObject,
		"properties": properties,
	}
	if len(required) > 0 {
		sort.Strings(required)
		schema["required"] = required
	}
	return schema
}

// jsonSchemaType maps FUSE parameter type strings (which mirror Go types) onto
// JSON Schema primitive types. Slice types (prefixed "[]") and the generic
// "array"/"slice" map to "array"; unknown types fall back to "string".
func jsonSchemaType(t string) string {
	if strings.HasPrefix(t, "[]") {
		return jsonTypeArray
	}
	switch t {
	case "int", "int8", "int16", "int32", "int64",
		"uint", "uint8", "uint16", "uint32", "uint64", "integer":
		return "integer"
	case "float", "float32", "float64", "number":
		return "number"
	case "bool", "boolean":
		return "boolean"
	case "map", jsonTypeObject:
		return jsonTypeObject
	case jsonTypeArray, "slice":
		return jsonTypeArray
	default:
		return "string"
	}
}
