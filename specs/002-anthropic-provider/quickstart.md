# Quickstart: Using Claude via the Anthropic provider

## Enable Anthropic

```bash
export LLM_ANTHROPIC_ENABLED=true
export LLM_ANTHROPIC_API_KEY=sk-ant-...          # your Anthropic API key
export LLM_ANTHROPIC_MODEL=claude-sonnet-4-5     # default model for this provider
# optional: export LLM_ANTHROPIC_BASE_URL=https://your-gateway/   # proxy/gateway override
# optional: make it the engine default
export LLM_DEFAULT_PROVIDER=anthropic
```

## Use it from a node

Any `ai/chat` or `ai/agent` node targets it with `"provider": "anthropic"` (or omit `provider` to
use the default). No workflow or code change is needed beyond configuration:

```json
{ "input": "Use your tools to add 2 and 3.", "provider": "anthropic", "maxIterations": 5 }
```

Run the existing agent example against Claude:

```bash
make build && make run            # server on :9090
curl -s -X POST localhost:9090/v1/workflows/trigger \
  -H 'content-type: application/json' \
  -d '{ "schemaId": "ai-agent-example", "input": {} }'
curl -s localhost:9090/v1/workflows/<workflowID>/status | jq '.output, .steps'
```

## Fast unit feedback (no key, no server)

```bash
go test ./internal/llm/providers/anthropic/...
```

The stub-server tests verify request mapping (system split, tools, tool_choice) and response
parsing (text, tool_use, usage) deterministically.
