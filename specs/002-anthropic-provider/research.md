# Phase 0 Research: Native Anthropic Provider

Verified against `github.com/anthropics/anthropic-sdk-go@v1.46.0` via `go doc`.

## R1 — Client & call

- `anthropic.NewClient(opts ...option.RequestOption) anthropic.Client` (value, like `openai`).
  Options: `option.WithAPIKey`, `option.WithBaseURL`, `option.WithHeader`.
- `client.Messages.New(ctx, anthropic.MessageNewParams{...}) (*anthropic.Message, error)`.
- Mirror `openaicompat`: store `name`, `defaultModel`, `client`; resolve `model = req.Model ||
  defaultModel`, error if empty (`anthropic[%s]: no model specified`).

## R2 — Request params (`MessageNewParams`)

- `Model Model` (`Model = string`) → set directly to the resolved model string.
- `MaxTokens int64` is **required** (unlike OpenAI). Supply `anthropicDefaultMaxTokens = 4096` when
  `req.MaxTokens` is nil; else use it. (FR-007)
- `Temperature param.Opt[float64]` → `anthropic.Float(float64(*req.Temperature))` when set.
- `System []TextBlockParam` → collect **all** `RoleSystem` messages into
  `[]anthropic.TextBlockParam{{Text: <content>}}` (Anthropic carries system *outside* `messages`).
- `Messages []MessageParam` → see R3.
- `Tools []ToolUnionParam` → see R4.
- `ToolChoice ToolChoiceUnionParam` → `"auto"` ⇒ `{OfAuto: &anthropic.ToolChoiceAutoParam{}}`;
  `"required"` ⇒ `{OfAny: &anthropic.ToolChoiceAnyParam{}}`; else leave unset.

## R3 — Message mapping (`llm.Message` → `MessageParam`)

Iterate `req.Messages`, skipping `RoleSystem` (handled in R2):

- `RoleUser` → `anthropic.NewUserMessage(anthropic.NewTextBlock(m.Content))`.
- `RoleTool` (a tool result) → `anthropic.NewUserMessage(anthropic.NewToolResultBlock(m.ToolCallID,
  m.Content, false))`. (Anthropic tool results live in a **user** turn.)
- `RoleAssistant` → `anthropic.NewAssistantMessage(blocks...)` where blocks are: an optional
  `NewTextBlock(m.Content)` (when non-empty) followed by one `NewToolUseBlock(tc.ID, tc.Arguments,
  tc.Name)` per `m.ToolCalls` (the SDK marshals `tc.Arguments json.RawMessage` as the input).

Constructors (stable, return `ContentBlockParamUnion`): `NewTextBlock(text)`,
`NewToolUseBlock(id string, input any, name string)`, `NewToolResultBlock(toolUseID, content string,
isError bool)`. `NewUserMessage`/`NewAssistantMessage(blocks ...ContentBlockParamUnion) MessageParam`.

Anthropic auto-combines consecutive same-role turns, so emitting one user message per tool result
is fine.

## R4 — Tools (`llm.Tool` → `ToolUnionParam`)

`llm.Tool.Parameters` is a full JSON-Schema object `{"type":"object","properties":{…},"required":
[…]}`. Anthropic's `ToolInputSchemaParam` takes the **pieces** separately:

```go
anthropic.ToolUnionParam{OfTool: &anthropic.ToolParam{
    Name:        t.Name,
    Description: anthropic.String(t.Description), // when non-empty
    InputSchema: anthropic.ToolInputSchemaParam{
        Properties: t.Parameters["properties"],     // any (the properties object)
        Required:   toStringSlice(t.Parameters["required"]), // []string
    },
}}
```
`Type` defaults to `"object"`. A small `toStringSlice` coerces `[]string` or `[]any` → `[]string`
(our converter emits `[]string`).

## R5 — Response parsing (`*anthropic.Message` → `llm.ChatResponse`)

- `resp.Content []ContentBlockUnion`: switch on `block.Type`:
  - `"text"` → append `block.Text` to the assistant content.
  - `"tool_use"` → `llm.ToolCall{ID: block.ID, Name: block.Name, Arguments: json.RawMessage(block.Input)}`
    (or `block.AsToolUse()`); `block.Input` is already `json.RawMessage`.
- `FinishReason` = `string(resp.StopReason)` (`end_turn`, `tool_use`, `max_tokens`, …).
- `Usage`: `PromptTokens = int(resp.Usage.InputTokens)`, `CompletionTokens =
  int(resp.Usage.OutputTokens)`, `TotalTokens = Input + Output` (Anthropic reports no total).
- Always return `Role: llm.RoleAssistant`.

## R6 — Errors

Wrap SDK errors as `fmt.Errorf("anthropic[%s]: chat completion failed: %w", name, err)`; an empty
`resp.Content` is not itself an error (the agent handles an empty answer).

## R7 — Test strategy (no live key)

Mirror `openaicompat/provider_test.go`: `httptest.NewServer` returning Anthropic-shaped JSON,
`anthropic.New(Config{BaseURL: srv.URL, APIKey: "test", Model: "claude-..."})`, and unmarshal the
captured request body to assert the request shape (system array, messages, tools[].input_schema,
tool_choice). Response fixtures: a text message and a `tool_use` message; assert parsed
`ToolCall`, usage, and stop reason. Pass a dummy `APIKey` so the SDK sends auth headers without
complaint.
