// Package ai provides LLM-backed workflow nodes: ai/chat (a single completion)
// and, in later phases, ai/agent (a tool-calling reasoning loop). Both slot into
// the workflow graph as ordinary internal functions.
package ai

import (
	"github.com/open-source-cloud/fuse/pkg/llm"
	"github.com/open-source-cloud/fuse/pkg/workflow"
)

// PackageID is the id of the ai function package.
const PackageID = "fuse/pkg/ai"

// New creates a new ai Package. The LLM provider registry is closed over by the
// function implementations so they can resolve providers at execution time; the
// tool registry lets the agent expose existing functions as tools and invoke them.
func New(providers llm.Registry, tools ToolRegistry) *workflow.Package {
	return workflow.NewPackage(
		PackageID,
		workflow.NewFunction(ChatFunctionID, ChatFunctionMetadata(), makeChatFunction(providers)),
		workflow.NewFunction(AgentFunctionID, AgentFunctionMetadata(), makeAgentFunction(providers, tools)),
	)
}
