package packages

import (
	"fmt"

	"github.com/open-source-cloud/fuse/internal/actors/actor"
	"github.com/open-source-cloud/fuse/internal/packages/functions/ai"
	"github.com/open-source-cloud/fuse/internal/packages/functions/logic"
	"github.com/open-source-cloud/fuse/internal/packages/functions/system"
	"github.com/open-source-cloud/fuse/internal/packages/transport"
	"github.com/open-source-cloud/fuse/pkg/workflow"
)

// interceptedOrAsyncFunctionIDs lists full function ids that must never be
// exposed to an agent as tools in Phase B (ADR-0007). These either complete
// asynchronously or are intercepted by the WorkflowHandler, so their result is
// routed away from the agent goroutine rather than returned inline. The whole
// ai package is excluded separately in ListTools (all ai/* nodes are async).
var interceptedOrAsyncFunctionIDs = map[string]struct{}{
	system.SleepFullFunctionID:                    {}, // intercepted by WorkflowHandler
	system.WaitFullFunctionID:                     {}, // intercepted
	system.SubWorkflowFullFunctionID:              {}, // intercepted
	system.ForEachFullFunctionID:                  {}, // intercepted
	logic.PackageID + "/" + logic.TimerFunctionID: {}, // async (delivers via Finish)
}

// AgentToolRegistry adapts the package Registry to the ai.ToolRegistry port the
// ai/agent node depends on. It lives in package packages (not ai) so that ai
// does not import internal/packages, which would create an import cycle.
type AgentToolRegistry struct {
	registry Registry
}

// compile-time assertion that the adapter satisfies the port.
var _ ai.ToolRegistry = (*AgentToolRegistry)(nil)

// NewAgentToolRegistry creates an adapter over the package registry. It holds the
// Registry interface (not a snapshot); ListTools reads it lazily at agent-execution
// time, after the registry has been populated at startup.
func NewAgentToolRegistry(registry Registry) *AgentToolRegistry {
	return &AgentToolRegistry{registry: registry}
}

// ListTools returns the synchronous, declared-parameter functions eligible to be
// exposed to the model as tools.
func (a *AgentToolRegistry) ListTools() []ai.ToolDescriptor {
	pkgs, err := a.registry.List()
	if err != nil {
		return nil
	}

	tools := make([]ai.ToolDescriptor, 0)
	for _, pkg := range pkgs {
		if pkg.ID == ai.PackageID {
			continue // every ai/* function (chat, agent) is async — never a tool in Phase B
		}
		for fullID, fn := range pkg.Functions {
			if !isExposableTool(fullID, fn) {
				continue
			}
			params := make([]workflow.ParameterSchema, 0, len(fn.Metadata.Input.Parameters))
			for _, p := range fn.Metadata.Input.Parameters {
				params = append(params, p)
			}
			tools = append(tools, ai.ToolDescriptor{
				FunctionID:  fullID,
				MangledName: ai.MangleToolName(fullID),
				Description: fmt.Sprintf("FUSE function %s", fullID),
				Parameters:  ai.ParameterSchemaToJSONSchema(params),
			})
		}
	}
	return tools
}

// InvokeTool runs the function with the given full id in-process and returns its
// result. The function must belong to a registered package; for a synchronous
// function the result is returned inline (Async == false).
func (a *AgentToolRegistry) InvokeTool(handle actor.Handle, functionID string, execInfo *workflow.ExecutionInfo) (workflow.FunctionResult, error) {
	pkgs, err := a.registry.List()
	if err != nil {
		return workflow.FunctionResult{}, err
	}
	for _, pkg := range pkgs {
		if _, ok := pkg.Functions[functionID]; ok {
			return pkg.ExecuteFunction(handle, functionID, execInfo)
		}
	}
	return workflow.FunctionResult{}, fmt.Errorf("tool function %q not found", functionID)
}

// isExposableTool reports whether a function may be offered to the model as a tool:
// it must be invocable in-process (has an internal transport), use declared (not
// schemaless/CustomParameters) inputs, and not be on the intercepted/async denylist.
func isExposableTool(fullID string, fn *LoadedFunction) bool {
	if fn == nil || fn.Metadata == nil {
		return false
	}
	// Only internal functions can be invoked in-process; non-internal functions
	// have a nil Transport in the registry and cannot be executed by the agent.
	if fn.Transport == nil || fn.Metadata.Transport != transport.Internal {
		return false
	}
	if fn.Metadata.Input.CustomParameters {
		return false
	}
	if _, excluded := interceptedOrAsyncFunctionIDs[fullID]; excluded {
		return false
	}
	return true
}
