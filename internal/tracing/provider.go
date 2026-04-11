// Package tracing provides OpenTelemetry distributed tracing for the FUSE workflow engine.
package tracing

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.30.0"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/open-source-cloud/fuse/internal/app/config"
)

const instrumentationName = "github.com/open-source-cloud/fuse"

// Provider wraps an OTel TracerProvider and exposes helpers for span creation.
type Provider struct {
	tracer trace.Tracer
	tp     *sdktrace.TracerProvider
	prop   propagation.TextMapPropagator
}

// NewProvider creates a tracing Provider from config.
// When OTel is disabled it returns a no-op provider so callers never need to nil-check.
func NewProvider(cfg *config.Config) (*Provider, error) {
	if !cfg.Otel.Enabled {
		return noopProvider(), nil
	}

	opts := []grpc.DialOption{}
	if cfg.Otel.Insecure {
		opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}

	exporter, err := otlptracegrpc.New(
		context.Background(),
		otlptracegrpc.WithEndpoint(cfg.Otel.Endpoint),
		otlptracegrpc.WithDialOption(opts...),
	)
	if err != nil {
		return nil, fmt.Errorf("tracing: create OTLP exporter: %w", err)
	}

	res, err := resource.New(context.Background(),
		resource.WithAttributes(
			semconv.ServiceName(cfg.Otel.ServiceName),
			semconv.ServiceVersion(cfg.Otel.ServiceVersion),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("tracing: create resource: %w", err)
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
	)

	prop := propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	)
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(prop)

	return &Provider{
		tracer: tp.Tracer(instrumentationName),
		tp:     tp,
		prop:   prop,
	}, nil
}

func noopProvider() *Provider {
	return &Provider{
		tracer: noop.NewTracerProvider().Tracer(instrumentationName),
		prop:   propagation.NewCompositeTextMapPropagator(),
	}
}

// StartSpan creates a new span as a child of the context's current span.
func (p *Provider) StartSpan(ctx context.Context, name string, attrs ...attribute.KeyValue) (context.Context, trace.Span) {
	return p.tracer.Start(ctx, name, trace.WithAttributes(attrs...))
}

// InjectCarrier serialises the current span context from ctx into a map for actor message propagation.
func (p *Provider) InjectCarrier(ctx context.Context) map[string]string {
	carrier := make(map[string]string)
	p.prop.Inject(ctx, propagation.MapCarrier(carrier))
	return carrier
}

// ExtractCarrier deserialises a trace carrier map back into a context with the parent span set.
func (p *Provider) ExtractCarrier(ctx context.Context, carrier map[string]string) context.Context {
	if len(carrier) == 0 {
		return ctx
	}
	return p.prop.Extract(ctx, propagation.MapCarrier(carrier))
}

// Shutdown flushes and stops the underlying TracerProvider. Safe to call on no-op providers.
func (p *Provider) Shutdown(ctx context.Context) error {
	if p.tp == nil {
		return nil
	}
	return p.tp.Shutdown(ctx)
}
