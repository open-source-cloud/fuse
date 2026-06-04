package ai

import "github.com/open-source-cloud/fuse/pkg/llm"

const (
	// contextStrategyDropOldest drops the oldest middle turns when over budget (deterministic).
	contextStrategyDropOldest = "drop-oldest"
	// contextStrategySummarize replaces dropped turns with a single LLM-generated summary turn
	// (opt-in; introduces an extra model call and nondeterminism — ADR-0028).
	contextStrategySummarize = "summarize"
	// approxCharsPerToken is the heuristic used by estimateTokens. It is a coarse budget guard,
	// not exact billing; centralized so a real tokenizer can replace it later.
	approxCharsPerToken = 4
)

// contextStrategyOrDefault normalizes the configured strategy, defaulting to drop-oldest.
func contextStrategyOrDefault(s string) string {
	if s == contextStrategySummarize {
		return contextStrategySummarize
	}
	return contextStrategyDropOldest
}

// estimateTokens approximates the token count of a message slice (~4 chars/token), counting both
// text content and tool-call arguments.
func estimateTokens(messages []llm.Message) int {
	chars := 0
	for _, m := range messages {
		chars += len(m.Content)
		for _, tc := range m.ToolCalls {
			chars += len(tc.Name) + len(tc.Arguments)
		}
	}
	return chars / approxCharsPerToken
}

// headLen returns the number of leading messages to always preserve: any leading system
// message(s) plus the first user message (the original task).
func headLen(messages []llm.Message) int {
	h := 0
	for h < len(messages) && messages[h].Role == llm.RoleSystem {
		h++
	}
	if h < len(messages) && messages[h].Role == llm.RoleUser {
		h++
	}
	return h
}

// trimContext bounds messages to budget tokens (ADR-0028). It preserves the head (leading system
// turns + first user task) and the most recent turns that fit, returning the oldest middle turns
// as dropped. budget<=0, already-fitting input, or a case where even the recent turns alone exceed
// the budget returns the input unchanged with no dropped turns.
func trimContext(messages []llm.Message, budget int) (kept, dropped []llm.Message) {
	if budget <= 0 || estimateTokens(messages) <= budget {
		return messages, nil
	}
	h := headLen(messages)
	rest := messages[h:]
	start := 0
	for start < len(rest) && estimateTokens(concatMessages(messages[:h], rest[start:])) > budget {
		start++
	}
	if start == 0 {
		return messages, nil
	}
	return concatMessages(messages[:h], rest[start:]), rest[:start]
}

// concatMessages returns a fresh slice of a followed by b.
func concatMessages(a, b []llm.Message) []llm.Message {
	out := make([]llm.Message, 0, len(a)+len(b))
	out = append(out, a...)
	out = append(out, b...)
	return out
}
