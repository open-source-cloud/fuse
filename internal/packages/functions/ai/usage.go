package ai

import "github.com/open-source-cloud/fuse/pkg/llm"

// UsageRecorder records LLM token usage and call outcomes for observability (ADR-0029). It is a
// narrow port so this package stays free of the metrics/prometheus dependency; the engine injects
// a metrics-backed implementation, tests a no-op or a fake.
type UsageRecorder interface {
	// RecordUsage records the tokens consumed by one completion for a given ai function.
	RecordUsage(function, provider, model string, u llm.Usage)
	// RecordCall records that a completion call was made and its outcome (success|error).
	RecordCall(function, provider, model, status string)
}

// NopUsageRecorder is a UsageRecorder that does nothing (default for tests / when metrics are off).
type NopUsageRecorder struct{}

// RecordUsage does nothing.
func (NopUsageRecorder) RecordUsage(string, string, string, llm.Usage) {}

// RecordCall does nothing.
func (NopUsageRecorder) RecordCall(string, string, string, string) {}
