package packages

import (
	"github.com/open-source-cloud/fuse/internal/metrics"
	"github.com/open-source-cloud/fuse/internal/packages/functions/ai"
	"github.com/open-source-cloud/fuse/pkg/llm"
)

// metricsUsageRecorder adapts *metrics.FuseMetrics to ai.UsageRecorder, keeping the ai package
// free of the prometheus dependency (ADR-0029).
type metricsUsageRecorder struct{ m *metrics.FuseMetrics }

// newUsageRecorder returns a metrics-backed recorder, or a no-op when metrics are absent.
func newUsageRecorder(m *metrics.FuseMetrics) ai.UsageRecorder {
	if m == nil {
		return ai.NopUsageRecorder{}
	}
	return metricsUsageRecorder{m: m}
}

// RecordUsage records prompt and completion tokens.
func (r metricsUsageRecorder) RecordUsage(function, provider, model string, u llm.Usage) {
	if u.PromptTokens > 0 {
		r.m.LLMTokens.WithLabelValues(function, provider, model, "prompt").Add(float64(u.PromptTokens))
	}
	if u.CompletionTokens > 0 {
		r.m.LLMTokens.WithLabelValues(function, provider, model, "completion").Add(float64(u.CompletionTokens))
	}
}

// RecordCall records a completion call and its outcome.
func (r metricsUsageRecorder) RecordCall(function, provider, model, status string) {
	r.m.LLMCalls.WithLabelValues(function, provider, model, status).Inc()
}
