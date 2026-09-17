package actors

// f01_1_verification_test.go — verification of BACKLOG_V2.md F-01, finding 1 ("f01-1-span"):
//
//	"internal/actors/workflow_func.go — node span context discarded (`_ = nodeCtx`), so function
//	 code cannot create child spans or inject `traceparent` outbound. Now critical, because it is
//	 the seam every SDK crosses."
//
// WHAT THESE TESTS PROVE
//   - The defect test and the actor control drive the REAL production entry point,
//     WorkflowFunc.HandleMessage, with a stub
//     gen.Process (Send + Log only) — the same technique the sibling verifier uses
//     (f01_2_verification_test.go). Nothing is mirrored or re-implemented: the ExecutionInfo the
//     node function receives is the one workflow_func.go builds at :139, and the node span is the
//     one it starts at :90.
//   - The defect test proves the observable consequence: a node function handed a live
//     *workflow.ExecutionInfo, during an execution whose inbound message carries a sampled
//     traceparent, has NO way to recover the trace context of its own node span, and therefore
//     cannot continue the run's trace in an outbound call. The probe is deliberately shape-blind
//     (see f01ReachTraceFrom): it accepts a context.Context, a trace.Span, a trace.SpanContext, a
//     map[string]string carrier, a traceparent string, or a "traceparent" key in the function
//     input, reached via any exported field or any zero-argument accessor. Any fix that puts the
//     node span's context within reach of function code flips it green; no particular field name,
//     type or fix shape is assumed.
//   - The two control tests PASS today, and pin down that the red is not an artifact. The probe
//     control feeds the probe a live node span context through five different post-fix shapes
//     (context field, context accessor, carrier field, span field, traceparent string) and asserts
//     it recovers the node span every time, plus a negative case that yields nothing — so "goes
//     green under any legitimate fix" is executed, not merely argued. The actor control proves the
//     harness really executes the registered function through the package/transport path and really
//     replies to the workflow handler, and this same provider + this same inbound carrier DO
//     produce an injectable node-span context when the extract/StartSpan pair of
//     workflow_func.go:89-95 is performed. (With OTEL_ENABLED=false, tracing.noopProvider() has an
//     empty composite propagator and InjectCarrier always returns an empty map — provider.go:82-87
//     — so the enabled path is required for this to mean anything.)
//
// WHAT THEY DO NOT PROVE
//   - Nothing about the HTTP boundary (inbound extraction, A-04 break 1), about async completion
//     spans, or about the persisted ExecutionTrace (a different notion of "trace", ADR-0020).
//   - Nothing about ergo mailbox ordering, supervision or Init: HandleMessage is called directly.
//   - No repository is involved, so there is no memory/Postgres driver dimension here.
//   - They do not assert on source text. Sub-claim "(a) the literal `_ = nodeCtx` is present" is
//     deliberately NOT asserted: any no-op consumption of nodeCtx would satisfy a source-text
//     check while leaving the defect in place, so the behavioural consequence is asserted instead.
//   - One fix shape cannot be absorbed by any test: changing the workflow.Function signature
//     itself (e.g. func(context.Context, *ExecutionInfo)) breaks compilation of every function
//     literal in the repo, this file included. That is a mechanical update, not a weakening.
//
// The defect test asserts the DESIRED behaviour, so it is expected to be RED on main.

import (
	"context"
	"reflect"
	"strings"
	"testing"
	"time"

	"ergo.services/ergo/gen"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"

	"github.com/open-source-cloud/fuse/internal/actors/actornames"
	"github.com/open-source-cloud/fuse/internal/app/config"
	"github.com/open-source-cloud/fuse/internal/messaging"
	"github.com/open-source-cloud/fuse/internal/metrics"
	"github.com/open-source-cloud/fuse/internal/packages"
	"github.com/open-source-cloud/fuse/internal/packages/transport"
	"github.com/open-source-cloud/fuse/internal/tracing"
	"github.com/open-source-cloud/fuse/internal/workflow/workflowactions"
	"github.com/open-source-cloud/fuse/pkg/workflow"
)

const (
	// A fixed, SAMPLED W3C traceparent, as an SDK caller would send it. Sampled matters: with an
	// unsampled parent the SDK returns a non-recording span that REUSES the parent span context,
	// and the "new engine span id" assertion below would be meaningless.
	f01InboundTraceparent = "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01"
	f01InboundTraceID     = "4bf92f3577b34da6a3ce929d0e0e4736"
	f01InboundSpanID      = "00f067aa0ba902b7"

	f01TraceparentKey = "traceparent"
	f01HexDigits      = "0123456789abcdef"

	f01PackageID      = "fuse/pkg/f01"
	f01FunctionName   = "outbound"
	f01FullFunctionID = f01PackageID + "/" + f01FunctionName
)

// --- harness -----------------------------------------------------------------------------------

// f011Log is a no-op gen.Log. Only the levels WorkflowFunc.HandleMessage uses are implemented; any
// other method would panic on the nil embedded interface, which is the intended loud failure.
type f011Log struct {
	gen.Log
}

func (l *f011Log) Trace(_ string, _ ...any)   {}
func (l *f011Log) Debug(_ string, _ ...any)   {}
func (l *f011Log) Info(_ string, _ ...any)    {}
func (l *f011Log) Warning(_ string, _ ...any) {}
func (l *f011Log) Error(_ string, _ ...any)   {}
func (l *f011Log) Panic(_ string, _ ...any)   {}

// f011Sent records one outbound send made by the actor under test.
type f011Sent struct {
	to      any
	message any
}

// f011Process is a stub gen.Process that records outbound sends instead of delivering them.
type f011Process struct {
	gen.Process
	log  *f011Log
	sent []f011Sent
}

func (p *f011Process) Send(to any, message any) error {
	p.sent = append(p.sent, f011Sent{to: to, message: message})
	return nil
}

func (p *f011Process) Log() gen.Log { return p.log }

// f011Registry is an isolated packages.Registry holding exactly one package, so the test never
// touches the process-wide singleton returned by packages.NewPackageRegistry().
type f011Registry struct {
	pkg *packages.LoadedPackage
}

func (r *f011Registry) Register(_ *workflow.Package) {}

func (r *f011Registry) Get(pkgID string) (*packages.LoadedPackage, error) {
	if r.pkg != nil && r.pkg.ID == pkgID {
		return r.pkg, nil
	}
	return nil, packages.ErrLoadedPackageNotFound
}

func (r *f011Registry) Has(pkgID string) bool {
	return r.pkg != nil && r.pkg.ID == pkgID
}

func (r *f011Registry) List() ([]*packages.LoadedPackage, error) {
	if r.pkg == nil {
		return []*packages.LoadedPackage{}, nil
	}
	return []*packages.LoadedPackage{r.pkg}, nil
}

// f01Provider builds a REAL, OTel-enabled tracing.Provider. tracing.Provider is a concrete struct
// with unexported fields and no seam, so an enabled provider can only be obtained from
// tracing.NewProvider — which also installs itself into the OTel globals (provider.go:72-73).
// Those globals are captured and restored so this file (which sorts first in the package) does not
// hand every later test a shut-down provider.
func f01Provider(t *testing.T) *tracing.Provider {
	t.Helper()

	prevTracerProvider := otel.GetTracerProvider()
	prevPropagator := otel.GetTextMapPropagator()

	cfg := &config.Config{}
	cfg.Otel.Enabled = true
	cfg.Otel.Endpoint = "localhost:4317"
	cfg.Otel.Insecure = true
	cfg.Otel.ServiceName = "fuse-f01-verification"
	cfg.Otel.ServiceVersion = "test"

	provider, err := tracing.NewProvider(cfg)
	require.NoError(t, err)
	require.NotNil(t, provider)

	t.Cleanup(func() {
		otel.SetTracerProvider(prevTracerProvider)
		otel.SetTextMapPropagator(prevPropagator)
		// No collector listens on the endpoint; keep the flush short so the suite does not pay the
		// full export timeout. Spans are never read back — every assertion here works off the
		// propagator, not off exported spans.
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()
		_ = provider.Shutdown(ctx)
	})

	return provider
}

// f01Metadata is the minimum metadata for an internal, code-backed function. Concurrency and
// RateLimit are intentionally absent: MapToRegistryPackage never sets them, which is why
// workflow_func.go:102/:108 never dereference the (nil) concurrency manager and rate limiter.
func f01Metadata() workflow.FunctionMetadata {
	return workflow.FunctionMetadata{
		Transport: transport.Internal,
		Input: workflow.InputMetadata{
			Parameters: make([]workflow.ParameterSchema, 0),
			Edges:      workflow.InputEdgeMetadata{Parameters: make([]workflow.ParameterSchema, 0)},
		},
		Output: workflow.OutputMetadata{
			Parameters: make([]workflow.ParameterSchema, 0),
			Edges:      make([]workflow.OutputEdgeMetadata, 0),
		},
	}
}

// f01NewWorkflowFunc wires a real WorkflowFunc worker around fn, with a real OTel-enabled provider
// and a real metrics registry, backed by a stub gen.Process.
func f01NewWorkflowFunc(t *testing.T, provider *tracing.Provider, fn workflow.Function) (*WorkflowFunc, *f011Process) {
	t.Helper()

	pkg := packages.MapToRegistryPackage(
		workflow.NewPackage(f01PackageID, workflow.NewFunction(f01FunctionName, f01Metadata(), fn)),
	)
	loadedFn, ok := pkg.Functions[f01FullFunctionID]
	require.Truef(t, ok, "package %s must expose %s", f01PackageID, f01FullFunctionID)
	require.NotNil(t, loadedFn.Transport, "the function must be registered as code-backed (executable)")

	worker := &WorkflowFunc{
		packageRegistry: &f011Registry{pkg: pkg},
		fuseMetrics:     metrics.NewFuseMetrics(),
		tracingProvider: provider,
	}
	process := &f011Process{log: &f011Log{}}
	// Promoted from the embedded act.Actor: this is the gen.Process the worker sends and logs through.
	worker.Process = process

	return worker, process
}

// f01ExecuteMessage builds the message WorkflowHandler sends to the func pool
// (messaging.NewExecuteFunctionMessage, as called from workflow_handler.go), carrying an inbound
// trace carrier.
func f01ExecuteMessage(wfID workflow.ID, execID workflow.ExecID) messaging.Message {
	return messaging.NewExecuteFunctionMessage(
		wfID,
		&workflowactions.RunFunctionAction{
			ThreadID:       1,
			FunctionID:     f01FullFunctionID,
			FunctionExecID: execID,
			Args:           map[string]any{},
		},
		"",
		map[string]string{f01TraceparentKey: f01InboundTraceparent},
	)
}

// --- shape-blind trace probe -------------------------------------------------------------------

// f01Reach is what a node function could recover about its own trace context from the only value
// it is handed.
type f01Reach struct {
	Source  string // e.g. `field Ctx`, `method Context()`, `input["traceparent"]`; empty = nothing
	TraceID string
	SpanID  string
	Scanned []string
}

type f01Candidate struct {
	name  string
	value reflect.Value
}

// f01ReachTraceFrom asks the question the backlog finding is about — "can function code continue
// the run's trace?" — without assuming how a fix would answer it. It walks every exported field of
// the struct behind the pointer, every zero-argument accessor that returns a plausible trace
// carrier, and (for a real ExecutionInfo) the function input, and returns the first one that yields
// a usable W3C trace context.
//
// It takes any so TestF01TraceProbe_FindsEveryFixShape can prove the probe is not itself the reason
// the defect test is red.
func f01ReachTraceFrom(provider *tracing.Provider, info any) f01Reach {
	reach := f01Reach{Scanned: make([]string, 0, 8)}
	if info == nil {
		return reach
	}
	if v := reflect.ValueOf(info); v.Kind() != reflect.Ptr || v.IsNil() || v.Elem().Kind() != reflect.Struct {
		return reach
	}

	for _, candidate := range f01TraceCandidates(info) {
		reach.Scanned = append(reach.Scanned, candidate.name)
		traceID, spanID, ok := f01TraceIDsOf(provider, candidate.value)
		if !ok {
			continue
		}
		reach.Source, reach.TraceID, reach.SpanID = candidate.name, traceID, spanID
		return reach
	}

	return reach
}

func f01TraceCandidates(info any) []f01Candidate {
	value := reflect.ValueOf(info).Elem()
	structType := value.Type()
	ptrType := reflect.TypeOf(info)
	ptrValue := reflect.ValueOf(info)

	candidates := make([]f01Candidate, 0, structType.NumField()+ptrType.NumMethod()+1)
	for i := range structType.NumField() {
		field := structType.Field(i)
		if !field.IsExported() {
			continue
		}
		candidates = append(candidates, f01Candidate{name: "field " + field.Name, value: value.Field(i)})
	}
	for i := range ptrType.NumMethod() {
		method := ptrType.Method(i)
		if method.Type.NumIn() != 1 || method.Type.NumOut() != 1 || !f01IsTraceCarrierType(method.Type.Out(0)) {
			continue
		}
		candidates = append(candidates, f01Candidate{
			name:  "method " + method.Name + "()",
			value: ptrValue.Method(i).Call(nil)[0],
		})
	}
	// A carrier smuggled through the function input is a legitimate fix shape too.
	if exec, isExecInfo := info.(*workflow.ExecutionInfo); isExecInfo && exec.Input != nil {
		candidates = append(candidates, f01Candidate{
			name:  `input["` + f01TraceparentKey + `"]`,
			value: reflect.ValueOf(exec.Input.GetStr(f01TraceparentKey)),
		})
	}

	return candidates
}

func f01IsTraceCarrierType(t reflect.Type) bool {
	ctxType := reflect.TypeOf((*context.Context)(nil)).Elem()
	spanType := reflect.TypeOf((*trace.Span)(nil)).Elem()

	switch {
	case t == ctxType || t.Implements(ctxType):
		return true
	case t == spanType || t.Implements(spanType):
		return true
	case t == reflect.TypeOf(trace.SpanContext{}):
		return true
	case t == reflect.TypeOf(map[string]string{}):
		return true
	case t.Kind() == reflect.String:
		return true
	default:
		return false
	}
}

// f01TraceIDsOf turns one candidate into (traceID, spanID) if it carries a usable trace context.
func f01TraceIDsOf(provider *tracing.Provider, value reflect.Value) (string, string, bool) {
	if !value.IsValid() {
		return "", "", false
	}
	switch value.Kind() {
	case reflect.Interface, reflect.Map, reflect.Ptr, reflect.Func, reflect.Slice:
		if value.IsNil() {
			return "", "", false
		}
	case reflect.String:
		return f01ParseTraceparent(value.String())
	default:
	}

	switch carrier := value.Interface().(type) {
	case context.Context:
		return f01ParseTraceparent(provider.InjectCarrier(carrier)[f01TraceparentKey])
	case trace.Span:
		ctx := trace.ContextWithSpan(context.Background(), carrier)
		return f01ParseTraceparent(provider.InjectCarrier(ctx)[f01TraceparentKey])
	case trace.SpanContext:
		ctx := trace.ContextWithSpanContext(context.Background(), carrier)
		return f01ParseTraceparent(provider.InjectCarrier(ctx)[f01TraceparentKey])
	case map[string]string:
		return f01ParseTraceparent(carrier[f01TraceparentKey])
	default:
		return "", "", false
	}
}

// f01ParseTraceparent extracts (traceID, spanID) from a W3C traceparent header value.
func f01ParseTraceparent(traceparent string) (string, string, bool) {
	parts := strings.Split(traceparent, "-")
	if len(parts) != 4 {
		return "", "", false
	}
	traceID, spanID := parts[1], parts[2]
	if len(traceID) != 32 || len(spanID) != 16 {
		return "", "", false
	}
	if strings.Trim(traceID, f01HexDigits) != "" || strings.Trim(spanID, f01HexDigits) != "" {
		return "", "", false
	}
	if strings.Trim(traceID, "0") == "" || strings.Trim(spanID, "0") == "" {
		return "", "", false
	}
	return traceID, spanID, true
}

// --- post-fix ExecutionInfo shapes, used only by the probe control ------------------------------

type f01CtxFieldShape struct {
	WorkflowID workflow.ID
	Ctx        context.Context //nolint:containedctx // models a post-fix ExecutionInfo field
}

type f01AccessorShape struct {
	WorkflowID workflow.ID
	ctx        context.Context //nolint:containedctx // models a post-fix ExecutionInfo field
}

func (i *f01AccessorShape) Context() context.Context { return i.ctx }

type f01CarrierFieldShape struct {
	WorkflowID   workflow.ID
	TraceCarrier map[string]string
}

type f01SpanFieldShape struct {
	WorkflowID workflow.ID
	Span       trace.Span
}

type f01TraceparentFieldShape struct {
	WorkflowID  workflow.ID
	Traceparent string
}

type f01NoTraceShape struct {
	WorkflowID  workflow.ID
	Environment string
	Ctx         context.Context //nolint:containedctx // deliberately nil: the pre-fix situation
	Tags        map[string]string
}

// --- tests -------------------------------------------------------------------------------------

// Control. Expected GREEN: it proves the harness drives the production path end to end AND that
// this provider can continue the inbound trace, so the red below is the missing seam and not a
// dead propagator or a function that never ran.
func TestWorkflowFunc_HandleMessage_RunsNodeFunctionAndProviderContinuesInboundTrace(t *testing.T) {
	// Arrange
	provider := f01Provider(t)
	wfID := workflow.NewID()
	execID := workflow.NewExecID(1)

	seen := make([]*workflow.ExecutionInfo, 0, 1)
	worker, process := f01NewWorkflowFunc(t, provider, func(info *workflow.ExecutionInfo) (workflow.FunctionResult, error) {
		seen = append(seen, info)
		return workflow.NewFunctionResultSuccess(), nil
	})

	// Act
	err := worker.HandleMessage(gen.PID{}, f01ExecuteMessage(wfID, execID))

	// Assert — the real production path ran the registered function and answered the handler.
	require.NoError(t, err)
	require.Len(t, seen, 1, "control: HandleMessage must have executed the registered node function exactly once")
	assert.Equal(t, wfID, seen[0].WorkflowID, "control: the function receives the message's workflow id")
	assert.Equal(t, execID, seen[0].ExecID, "control: the function receives the message's exec id")

	require.Len(t, process.sent, 1, "control: exactly one message must go back to the workflow handler")
	assert.Equal(t, actornames.WorkflowHandlerName(wfID), process.sent[0].to)
	resultMsg, isMessage := process.sent[0].message.(messaging.Message)
	require.True(t, isMessage, "control: the handler reply must be a messaging.Message")
	assert.Equal(t, messaging.FunctionResult, resultMsg.Type)

	// Assert — the provider is capable: the extract/StartSpan pair of workflow_func.go:89-95, run
	// with this provider and this carrier, yields an injectable node-span context.
	parentCtx := provider.ExtractCarrier(context.Background(), map[string]string{f01TraceparentKey: f01InboundTraceparent})
	nodeCtx, nodeSpan := provider.StartSpan(parentCtx, "node.execute")
	defer nodeSpan.End()

	traceID, spanID, ok := f01ParseTraceparent(provider.InjectCarrier(nodeCtx)[f01TraceparentKey])
	require.True(t, ok, "control: an enabled provider must inject a traceparent from a node span context")
	assert.Equal(t, f01InboundTraceID, traceID, "control: the node span must continue the inbound trace")
	assert.NotEqual(t, f01InboundSpanID, spanID, "control: the node span must have its own span id")
}

// Control. Expected GREEN: the probe the defect test uses recognises every fix shape anyone might
// reasonably choose for threading the node span context to function code, and produces no false
// positive from a struct that carries none. Without this, a red below would be ambiguous between
// "the seam is missing" and "the probe only knows one fix shape".
func TestF01TraceProbe_FindsEveryFixShape(t *testing.T) {
	// Arrange — a live node span context, exactly as workflow_func.go:90 produces one.
	provider := f01Provider(t)
	parentCtx := provider.ExtractCarrier(context.Background(), map[string]string{f01TraceparentKey: f01InboundTraceparent})
	nodeCtx, nodeSpan := provider.StartSpan(parentCtx, "node.execute")
	defer nodeSpan.End()
	nodeSpanID := nodeSpan.SpanContext().SpanID().String()
	nodeCarrier := provider.InjectCarrier(nodeCtx)

	testCases := []struct {
		name       string
		info       any
		wantSource string
	}{
		{"context field", &f01CtxFieldShape{WorkflowID: workflow.NewID(), Ctx: nodeCtx}, "field Ctx"},
		{"context accessor", &f01AccessorShape{WorkflowID: workflow.NewID(), ctx: nodeCtx}, "method Context()"},
		{"carrier field", &f01CarrierFieldShape{WorkflowID: workflow.NewID(), TraceCarrier: nodeCarrier}, "field TraceCarrier"},
		{"span field", &f01SpanFieldShape{WorkflowID: workflow.NewID(), Span: nodeSpan}, "field Span"},
		{
			"traceparent string field",
			&f01TraceparentFieldShape{WorkflowID: workflow.NewID(), Traceparent: nodeCarrier[f01TraceparentKey]},
			"field Traceparent",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			// Act
			reach := f01ReachTraceFrom(provider, testCase.info)

			// Assert
			assert.Equal(t, testCase.wantSource, reach.Source, "scanned: %v", reach.Scanned)
			assert.Equal(t, f01InboundTraceID, reach.TraceID, "the fix shape must expose the run's trace id")
			assert.Equal(t, nodeSpanID, reach.SpanID, "the fix shape must expose the node span, not the caller's remote span")
			assert.NotEqual(t, f01InboundSpanID, reach.SpanID)
		})
	}

	t.Run("no trace anywhere", func(t *testing.T) {
		// Act — a nil context field, an unrelated map and a uuid-shaped string must not be mistaken
		// for a trace context.
		reach := f01ReachTraceFrom(provider, &f01NoTraceShape{
			WorkflowID:  workflow.NewID(),
			Environment: "default",
			Tags:        map[string]string{"team": "core"},
		})

		// Assert
		assert.Empty(t, reach.Source)
		assert.Empty(t, reach.TraceID)
		assert.NotEmpty(t, reach.Scanned, "the probe must actually have walked the struct")
	})
}

// The defect. Expected RED on main: workflow_func.go throws the node span context away, so the
// function it then invokes cannot parent anything on the node span nor continue the run's trace.
func TestWorkflowFunc_HandleMessage_GivesNodeFunctionTheNodeSpanContext(t *testing.T) {
	// Arrange
	provider := f01Provider(t)
	calls := 0
	var reach f01Reach
	worker, _ := f01NewWorkflowFunc(t, provider, func(info *workflow.ExecutionInfo) (workflow.FunctionResult, error) {
		// What every outbound-HTTP / SDK node must be able to do: continue the run's trace.
		calls++
		reach = f01ReachTraceFrom(provider, info)
		return workflow.NewFunctionResultSuccess(), nil
	})

	// Act
	err := worker.HandleMessage(gen.PID{}, f01ExecuteMessage(workflow.NewID(), workflow.NewExecID(1)))

	// Assert
	require.NoError(t, err)
	require.Equal(t, 1, calls, "the node function must have been executed exactly once")

	assert.NotEmptyf(t, reach.Source,
		"node function has no reachable trace context on *workflow.ExecutionInfo (scanned: %v): the node span context created at workflow_func.go:90 is discarded, so function code can neither create a child span nor inject traceparent outbound",
		reach.Scanned)
	assert.Equalf(t, f01InboundTraceID, reach.TraceID,
		"an outbound call from the node function must continue the run's trace %s (reached via %q, scanned %v)",
		f01InboundTraceID, reach.Source, reach.Scanned)
	assert.NotEqualf(t, f01InboundSpanID, reach.SpanID,
		"the node function's context must be parented on the engine's node span, not on the caller's remote span (reached via %q)",
		reach.Source)
}
