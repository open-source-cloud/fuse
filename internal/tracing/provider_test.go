package tracing_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/open-source-cloud/fuse/internal/app/config"
	"github.com/open-source-cloud/fuse/internal/tracing"
)

func noopConfig() *config.Config {
	cfg := &config.Config{}
	cfg.Otel.Enabled = false
	return cfg
}

func TestNewProvider_noopWhenDisabled(t *testing.T) {
	cfg := noopConfig()
	p, err := tracing.NewProvider(cfg)
	require.NoError(t, err)
	require.NotNil(t, p)

	// Spans on a noop provider must not panic and return valid (noop) spans.
	ctx, span := p.StartSpan(context.Background(), "test.span",
		attribute.String("key", "value"),
	)
	assert.NotNil(t, ctx)
	assert.NotNil(t, span)
	// Noop spans are not recording.
	assert.False(t, span.IsRecording())
	span.End()
}

func TestProvider_shutdownIsIdempotent(t *testing.T) {
	cfg := noopConfig()
	p, err := tracing.NewProvider(cfg)
	require.NoError(t, err)

	// Multiple shutdowns must not panic or error.
	assert.NoError(t, p.Shutdown(context.Background()))
	assert.NoError(t, p.Shutdown(context.Background()))
}

func TestProvider_injectExtractCarrier_roundTrip(t *testing.T) {
	cfg := noopConfig()
	p, err := tracing.NewProvider(cfg)
	require.NoError(t, err)

	ctx := context.Background()
	carrier := p.InjectCarrier(ctx)

	// A noop context produces an empty carrier (no active span to propagate).
	// Extracting it must still return a valid context.
	extracted := p.ExtractCarrier(ctx, carrier)
	assert.NotNil(t, extracted)
}

func TestProvider_injectExtractCarrier_nilCarrier(t *testing.T) {
	cfg := noopConfig()
	p, err := tracing.NewProvider(cfg)
	require.NoError(t, err)

	ctx := context.Background()
	// Extracting a nil carrier must return the original context unchanged.
	// Note: in Go, len(nilMap) == 0, so this follows the same empty-carrier path.
	out := p.ExtractCarrier(ctx, nil)
	assert.Equal(t, ctx, out)
}

func TestProvider_injectExtractCarrier_emptyCarrier(t *testing.T) {
	cfg := noopConfig()
	p, err := tracing.NewProvider(cfg)
	require.NoError(t, err)

	ctx := context.Background()
	carrier := map[string]string{}

	// Extracting an empty (non-nil) carrier must also return the original context.
	out := p.ExtractCarrier(ctx, carrier)
	assert.Equal(t, ctx, out)
}

func TestProvider_startSpanReturnsChildContext(t *testing.T) {
	cfg := noopConfig()
	p, err := tracing.NewProvider(cfg)
	require.NoError(t, err)

	parentCtx := context.Background()
	childCtx, span := p.StartSpan(parentCtx, "child.span")
	defer span.End()

	// The child context must be distinct from the parent.
	assert.NotEqual(t, parentCtx, childCtx)
	// Span from child context must be valid.
	spanFromCtx := trace.SpanFromContext(childCtx)
	assert.NotNil(t, spanFromCtx)
	assert.Equal(t, span, spanFromCtx)
}
