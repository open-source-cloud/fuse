# Quickstart: AI Agent Node

Run an `ai/agent` node end-to-end against a local model (no API key) using Ollama.

## Prerequisites
- Ollama running locally with a tool-capable model pulled (e.g. `ollama pull qwen2.5`).
- FUSE configured to use Ollama as the default LLM provider:
  ```bash
  export LLM_DEFAULT_PROVIDER=ollama
  export LLM_OLLAMA_ENABLED=true
  export LLM_OLLAMA_BASE_URL=http://localhost:11434/v1
  export LLM_OLLAMA_MODEL=qwen2.5
  ```

## 1. Build & run
```bash
make build && make run    # server on :9090, debug logging
```

## 2. A minimal agent workflow
A graph with a trigger → `fuse/pkg/ai/agent` node. The agent is told to add two numbers and has a
synchronous arithmetic tool available (e.g. `fuse/pkg/logic/sum`). See
`examples/workflows/ai-agent-example.json`. The agent node inputs:
```json
{ "input": "Use your tools to add 2 and 3, then tell me the result.", "maxIterations": 5 }
```

## 3. Trigger it
```bash
curl -s -X POST localhost:9090/v1/workflows/trigger \
  -H 'content-type: application/json' \
  -d '{ "schemaId": "ai-agent-example", "input": {} }'
```

## 4. Inspect the run
```bash
curl -s localhost:9090/v1/workflows/<workflowID>/status | jq '.output, .steps'
```
Expect: `output` mentions `5`; `steps` shows a call to `fuse/pkg/logic/sum` with `{a:2,b:3}` and
result `{sum:5}`. With debug log level, a `logs` field is also present.

## 5. Verify the non-blocking property
While the agent runs, trigger another simple workflow; it should execute immediately — the agent's
multi-step interaction does not hold a worker slot (FR-006 / SC-004).

## CI note
This example is skipped under `CI=true` (it needs a live provider), consistent with
`make examples-ci` / `scripts/run-example-workflows.sh` and the existing AI/timer-dependent
examples.

## Fast unit feedback (no model, no server)
```bash
go test ./internal/packages/functions/ai/... ./internal/packages/... ./pkg/workflow/...
```
The scripted-provider tests prove the loop, tool threading, and termination deterministically.
