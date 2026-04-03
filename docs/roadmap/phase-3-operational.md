# Phase 3: Operational — Make It Production-Ready

> **Goal:** Add operational primitives that make Fuse safe and manageable in production environments. Depends on Phase 1 (durability) and Phase 2 (cancellation).

---

## 3.1 Event-Driven Triggers

### Motivation

Currently, workflows are triggered **only via HTTP POST** to `/v1/workflows/trigger` (`internal/handlers/trigger_workflow.go:53`). This forces all workflow initiation through a synchronous HTTP call, which limits automation capabilities:

- **No scheduled execution**: Can't run a workflow every hour or at 3am daily
- **No event-bus triggers**: One workflow completing can't automatically trigger another
- **No webhook triggers**: External services (GitHub, Stripe, Slack) can't trigger workflows via webhooks with custom paths
- **No internal event routing**: System events (workflow.completed, workflow.failed) can't trigger reactive workflows

### Prior Art

**Inngest** is fundamentally event-driven. Functions declare which events trigger them (`inngest.createFunction({ triggers: [{ event: "user/signup" }] }, ...)`). Any event can trigger multiple functions. Events carry typed payloads. Inngest also supports cron triggers (`{ cron: "0 * * * *" }`) as first-class trigger types.

**n8n** provides multiple trigger node types:
- **Schedule Trigger**: Cron-like scheduling
- **Webhook Trigger**: Custom webhook URLs per workflow
- **App Triggers**: GitHub, Slack, Stripe, etc. push events to the trigger
- **Manual Trigger**: For testing

**Restate** uses HTTP-based ingress with virtual object addressing. Workflows are invoked by sending messages to specific service/handler combinations. Event-driven patterns are built on top of the messaging primitives.

**What Fuse should adopt:** A `TriggerType` system supporting HTTP (existing), Cron (scheduled), Webhook (custom paths), and Event (internal pub/sub). Each workflow schema declares its trigger type and configuration.

### Design

#### 3.1.1 Trigger Type System

```go
// internal/workflow/trigger.go

// TriggerType classifies how a workflow is initiated
type TriggerType string

const (
    // TriggerHTTP is the current trigger type — explicit API call
    TriggerHTTP TriggerType = "http"
    // TriggerCron triggers on a cron schedule
    TriggerCron TriggerType = "cron"
    // TriggerWebhook triggers when a specific webhook URL is called
    TriggerWebhook TriggerType = "webhook"
    // TriggerEvent triggers when a matching internal event is emitted
    TriggerEvent TriggerType = "event"
)

// TriggerConfig defines the trigger configuration for a workflow schema
type TriggerConfig struct {
    Type    TriggerType    `json:"type" bson:"type" validate:"required,oneof=http cron webhook event"`
    Cron    *CronConfig    `json:"cron,omitempty" bson:"cron,omitempty"`
    Webhook *WebhookConfig `json:"webhook,omitempty" bson:"webhook,omitempty"`
    Event   *EventConfig   `json:"event,omitempty" bson:"event,omitempty"`
}

// CronConfig defines cron trigger parameters
type CronConfig struct {
    // Expression is a cron expression (e.g., "0 */5 * * *" for every 5 minutes)
    Expression string `json:"expression" bson:"expression" validate:"required"`
    // Timezone for cron evaluation (e.g., "America/New_York")
    Timezone string `json:"timezone,omitempty" bson:"timezone,omitempty"`
    // Input is static input data passed to the trigger node on each execution
    Input map[string]any `json:"input,omitempty" bson:"input,omitempty"`
}

// WebhookConfig defines webhook trigger parameters
type WebhookConfig struct {
    // Path is the custom webhook path (e.g., "/hooks/github-push")
    Path string `json:"path" bson:"path" validate:"required"`
    // Method is the HTTP method to listen for (default: POST)
    Method string `json:"method,omitempty" bson:"method,omitempty"`
    // Secret is an optional HMAC secret for webhook signature verification
    Secret string `json:"secret,omitempty" bson:"secret,omitempty"`
}

// EventConfig defines event trigger parameters
type EventConfig struct {
    // EventType is the event name to listen for (e.g., "workflow.completed", "order.created")
    EventType string `json:"eventType" bson:"eventType" validate:"required"`
    // Filter is an optional expr-lang expression to filter matching events
    Filter string `json:"filter,omitempty" bson:"filter,omitempty"`
}
```

#### 3.1.2 GraphSchema Extension

```go
// Extend GraphSchema
type GraphSchema struct {
    ID      string                 `json:"id" bson:"id"`
    Trigger string                 `json:"trigger" bson:"trigger"` // trigger node ID
    Nodes   []*NodeSchema          `json:"nodes" bson:"nodes"`
    Edges   []*EdgeSchema          `json:"edges" bson:"edges"`
    Timeout *WorkflowTimeoutConfig `json:"timeout,omitempty" bson:"timeout,omitempty"`
    TriggerConfig *TriggerConfig   `json:"triggerConfig,omitempty" bson:"triggerConfig,omitempty"` // NEW
}
```

#### 3.1.3 Internal Event Bus

```go
// internal/events/bus.go

// Event represents an internal system event
type Event struct {
    Type      string         `json:"type"`
    Source    string         `json:"source"`
    Timestamp time.Time     `json:"timestamp"`
    Data      map[string]any `json:"data"`
}

// EventBus manages event subscriptions and publishing
type EventBus interface {
    // Publish emits an event to all matching subscribers
    Publish(event Event) error
    // Subscribe registers a callback for events matching the given type
    Subscribe(eventType string, handler EventHandler) (SubscriptionID, error)
    // Unsubscribe removes a subscription
    Unsubscribe(id SubscriptionID) error
}

type EventHandler func(event Event) error
type SubscriptionID string
```

Memory implementation using channels and goroutines. Future: Redis Pub/Sub or NATS for distributed deployments.

#### 3.1.4 Cron Scheduler Actor

```go
// internal/actors/cron_scheduler.go

type CronScheduler struct {
    act.Actor
    graphRepo     repositories.GraphRepository
    entries       map[string]*cron.Entry // schemaID -> cron entry
    cronEngine    *cron.Cron
}

func (a *CronScheduler) Init(_ ...any) error {
    a.cronEngine = cron.New(cron.WithSeconds())
    // Load all schemas with cron triggers and register them
    schemas := a.loadCronSchemas()
    for _, schema := range schemas {
        a.registerCronTrigger(schema)
    }
    a.cronEngine.Start()
    return nil
}

func (a *CronScheduler) registerCronTrigger(schema *GraphSchema) {
    cfg := schema.TriggerConfig.Cron
    a.cronEngine.AddFunc(cfg.Expression, func() {
        triggerMsg := messaging.NewTriggerWorkflowMessage(schema.ID, workflow.NewID())
        a.Send(gen.Atom(WorkflowSupervisorName), triggerMsg)
    })
}
```

#### 3.1.5 Webhook Router

```go
// internal/actors/webhook_router.go

// WebhookRouter registers and routes incoming webhooks to the correct workflow
type WebhookRouter struct {
    act.Actor
    graphRepo repositories.GraphRepository
    routes    map[string]string // path -> schemaID
}

func (a *WebhookRouter) Init(_ ...any) error {
    // Load all schemas with webhook triggers and register their paths
    schemas := a.loadWebhookSchemas()
    for _, schema := range schemas {
        a.routes[schema.TriggerConfig.Webhook.Path] = schema.ID
    }
    return nil
}
```

Webhook routes registered in the MuxServer alongside existing routes:

```
POST /v1/hooks/{path...} -> WebhookRouter -> TriggerWorkflow
```

#### 3.1.6 Workflow Lifecycle Events

Emit events at key workflow lifecycle points:

```go
// Predefined event types
const (
    EventWorkflowTriggered  = "workflow.triggered"
    EventWorkflowCompleted  = "workflow.completed"
    EventWorkflowFailed     = "workflow.failed"
    EventWorkflowCancelled  = "workflow.cancelled"
    EventFunctionCompleted   = "function.completed"
    EventFunctionFailed      = "function.failed"
)
```

In `WorkflowHandler`, after state transitions:

```go
// After setting StateFinished:
a.eventBus.Publish(events.Event{
    Type:   events.EventWorkflowCompleted,
    Source: a.workflow.ID().String(),
    Data: map[string]any{
        "workflowId": a.workflow.ID().String(),
        "schemaId":   a.workflow.Schema().ID,
    },
})
```

#### 3.1.7 Event Trigger Subscriber

```go
// internal/actors/event_trigger.go

type EventTrigger struct {
    act.Actor
    eventBus    events.EventBus
    graphRepo   repositories.GraphRepository
    subscriptions map[string][]string // eventType -> []schemaID
}

func (a *EventTrigger) Init(_ ...any) error {
    schemas := a.loadEventSchemas()
    for _, schema := range schemas {
        cfg := schema.TriggerConfig.Event
        a.eventBus.Subscribe(cfg.EventType, func(event events.Event) error {
            // Apply optional filter
            if cfg.Filter != "" {
                matches, err := evaluateFilter(cfg.Filter, event.Data)
                if err != nil || !matches {
                    return nil
                }
            }
            triggerMsg := messaging.NewTriggerWorkflowWithInputMessage(
                schema.ID, workflow.NewID(), event.Data,
            )
            return a.Send(gen.Atom(WorkflowSupervisorName), triggerMsg)
        })
    }
    return nil
}
```

### Alternatives Considered

1. **External scheduler (Kubernetes CronJobs, systemd timers)**: Requires external infrastructure. An in-process cron scheduler is simpler and keeps Fuse self-contained. Can always delegate to external schedulers via HTTP triggers.

2. **Separate trigger service**: Microservice for trigger management. Over-engineered for current scope. The actor model naturally supports dedicated scheduler/router actors within the same process.

3. **Generic pub/sub only (no built-in cron/webhook)**: Defers common patterns to users. Cron and webhooks are near-universal requirements and should be first-class.

### Migration Plan

- `TriggerConfig` is optional on `GraphSchema` — existing schemas default to `TriggerHTTP` (current behavior)
- New actors (CronScheduler, WebhookRouter, EventTrigger) added to the supervision tree
- EventBus starts as an in-memory implementation, replaceable with Redis/NATS later
- Existing HTTP trigger endpoint unchanged

### Open Questions

1. Should cron schedules support "catch-up" (run missed schedules after downtime) or "skip missed" semantics?
2. Should webhook payloads be validated against the trigger node's input schema?
3. Should the event bus be synchronous (blocking) or asynchronous (fire-and-forget)?
4. Should event triggers support backpressure (skip events when too many workflows are running)?

---

## 3.2 Concurrency Control

### Motivation

The `WorkflowFuncPool` has a fixed pool size of 3 workers (`internal/actors/workflow_func_pool.go:43`):

```go
func (p *WorkflowFuncPool) Init(_ ...any) (act.PoolOptions, error) {
    return act.PoolOptions{
        WorkerFactory: p.workflowFunc.Factory,
        PoolSize:      3,
    }, nil
}
```

This limits concurrent function execution within a single workflow, but there are no controls for:
- How many instances of a specific function can run across all workflows
- How many workflows can run simultaneously
- How many executions a specific external API endpoint handles concurrently

Without concurrency controls, a burst of workflow triggers can overwhelm external systems, cause rate limiting errors, and cascade failures.

### Prior Art

**Inngest** provides rich concurrency control:
- **Per-function concurrency**: `{ concurrency: [{ limit: 5 }] }` — max 5 parallel runs of this function
- **Per-key concurrency**: `{ concurrency: [{ limit: 1, key: "event.data.userId" }] }` — max 1 run per user
- **Concurrency scoping**: Limits can apply at function, account, or environment level
- **Queue behavior**: Excess invocations are queued (not rejected)

**Restate** uses **single-writer concurrency** for virtual objects — requests to the same virtual object key are serialized automatically. This eliminates race conditions without explicit locking.

**What Fuse should adopt:** Configurable concurrency limits at two levels: per-function (across all workflows) and per-workflow (per schema). Excess executions should be queued, not rejected.

### Design

#### 3.2.1 Concurrency Configuration

```go
// internal/workflow/concurrency.go

// ConcurrencyConfig defines concurrency limits
type ConcurrencyConfig struct {
    // Limit is the maximum number of concurrent executions
    Limit int `json:"limit" bson:"limit" validate:"min=1,max=1000"`
    // Key is an optional expression that scopes the limit (e.g., "input.userId")
    // When set, the limit applies per unique key value
    Key string `json:"key,omitempty" bson:"key,omitempty"`
}
```

#### 3.2.2 Schema Extensions

```go
// Function-level concurrency (on PackagedFunction metadata)
type FunctionMetadata struct {
    Transport   transport.Type
    Input       InputMetadata
    Output      OutputMetadata
    Concurrency *ConcurrencyConfig `json:"concurrency,omitempty" bson:"concurrency,omitempty"` // NEW
}

// Workflow-level concurrency (on GraphSchema)
type GraphSchema struct {
    // ... existing fields ...
    Concurrency *ConcurrencyConfig `json:"concurrency,omitempty" bson:"concurrency,omitempty"` // NEW
}
```

#### 3.2.3 Concurrency Semaphore

```go
// internal/concurrency/semaphore.go

// Semaphore provides bounded concurrency control with queueing
type Semaphore struct {
    mu       sync.Mutex
    limit    int
    active   int
    queue    []chan struct{}
}

func NewSemaphore(limit int) *Semaphore {
    return &Semaphore{limit: limit, queue: make([]chan struct{}, 0)}
}

// Acquire blocks until a slot is available. Returns a release function.
func (s *Semaphore) Acquire() func() {
    s.mu.Lock()
    if s.active < s.limit {
        s.active++
        s.mu.Unlock()
        return s.release
    }
    // Queue the request
    ch := make(chan struct{})
    s.queue = append(s.queue, ch)
    s.mu.Unlock()
    <-ch // Block until released
    return s.release
}

// TryAcquire returns true if a slot is immediately available
func (s *Semaphore) TryAcquire() (func(), bool) {
    s.mu.Lock()
    defer s.mu.Unlock()
    if s.active < s.limit {
        s.active++
        return s.release, true
    }
    return nil, false
}

func (s *Semaphore) release() {
    s.mu.Lock()
    defer s.mu.Unlock()
    if len(s.queue) > 0 {
        // Wake up the next queued request
        ch := s.queue[0]
        s.queue = s.queue[1:]
        close(ch)
    } else {
        s.active--
    }
}

// Active returns the current number of active acquisitions
func (s *Semaphore) Active() int {
    s.mu.Lock()
    defer s.mu.Unlock()
    return s.active
}

// Queued returns the number of waiting acquisitions
func (s *Semaphore) Queued() int {
    s.mu.Lock()
    defer s.mu.Unlock()
    return len(s.queue)
}
```

#### 3.2.4 Concurrency Manager

```go
// internal/concurrency/manager.go

// Manager tracks and enforces concurrency limits across the system
type Manager struct {
    mu         sync.RWMutex
    functions  map[string]*Semaphore // functionID -> semaphore
    workflows  map[string]*Semaphore // schemaID -> semaphore
    keyed      map[string]*Semaphore // "functionID:keyValue" -> semaphore
}

func NewManager() *Manager {
    return &Manager{
        functions: make(map[string]*Semaphore),
        workflows: make(map[string]*Semaphore),
        keyed:     make(map[string]*Semaphore),
    }
}

// AcquireFunction acquires a concurrency slot for a function execution
func (m *Manager) AcquireFunction(functionID string, limit int) func() {
    m.mu.Lock()
    sem, exists := m.functions[functionID]
    if !exists {
        sem = NewSemaphore(limit)
        m.functions[functionID] = sem
    }
    m.mu.Unlock()
    return sem.Acquire()
}

// AcquireWorkflow acquires a concurrency slot for a workflow execution
func (m *Manager) AcquireWorkflow(schemaID string, limit int) func() {
    m.mu.Lock()
    sem, exists := m.workflows[schemaID]
    if !exists {
        sem = NewSemaphore(limit)
        m.workflows[schemaID] = sem
    }
    m.mu.Unlock()
    return sem.Acquire()
}
```

#### 3.2.5 Integration with WorkflowFunc

In `WorkflowFunc.HandleMessage` (`internal/actors/workflow_func.go`), wrap function execution with concurrency control:

```go
func (a *WorkflowFunc) HandleMessage(from gen.PID, message any) error {
    // ... parse ExecuteFunctionMessage ...

    // Acquire concurrency slot (blocks if limit reached)
    functionID := fmt.Sprintf("%s/%s", execMsg.PackageID, execMsg.FunctionID)
    metadata := a.getMetadata(execMsg.PackageID, execMsg.FunctionID)
    if metadata != nil && metadata.Concurrency != nil {
        release := a.concurrencyManager.AcquireFunction(functionID, metadata.Concurrency.Limit)
        defer release()
    }

    // ... execute function ...
}
```

#### 3.2.6 Integration with WorkflowSupervisor

For workflow-level concurrency, check before spawning:

```go
func (s *WorkflowSupervisor) HandleMessage(from gen.PID, message any) error {
    // ... parse TriggerWorkflowMessage ...

    schema := s.getSchema(triggerMsg.SchemaID)
    if schema.Concurrency != nil {
        release := s.concurrencyManager.AcquireWorkflow(triggerMsg.SchemaID, schema.Concurrency.Limit)
        // Store release function for cleanup when workflow completes
        s.releaseMap[triggerMsg.WorkflowID] = release
    }

    return s.spawnWorkflowActor(triggerMsg.SchemaID, triggerMsg.WorkflowID)
}
```

### Alternatives Considered

1. **Actor mailbox backpressure**: Ergo's actor mailboxes provide natural backpressure, but they don't provide cross-workflow limits. A function called by 100 workflows simultaneously needs a global limit.

2. **External rate limiter (Redis-based)**: More suitable for distributed deployments but adds infrastructure dependency. Start with in-memory semaphores, add Redis backend later.

3. **Reject instead of queue**: Simpler but worse user experience. Queuing ensures all workflows eventually execute.

### Migration Plan

- `ConcurrencyConfig` is optional on both `FunctionMetadata` and `GraphSchema`
- `ConcurrencyManager` registered as a singleton via DI (uber-go/fx)
- No changes to existing schemas — unconfigured functions/workflows have no limits (current behavior)

### Open Questions

1. Should the concurrency queue have a maximum depth (to prevent unbounded memory growth)?
2. How should concurrency interact with retries — does a retry consume a new slot or reuse the existing one?
3. Should concurrency metrics be exposed via a health/metrics endpoint?
4. For distributed deployments, should we support Redis-backed semaphores from the start?

---

## 3.3 Throttling & Rate Limiting

### Motivation

External HTTP function providers (registered via the package API) often impose rate limits. For example, the Stripe API allows 100 requests/second, GitHub allows 5000 requests/hour. When workflows call these APIs without rate limiting, they can:

- Exceed API quotas, causing `429 Too Many Requests` errors
- Get temporarily or permanently banned
- Cause cascading failures across all workflows using the same API

Concurrency control (3.2) limits parallel execution but doesn't control the **rate** of execution. 10 fast functions completing in 100ms each could fire 100 requests/second even with a concurrency limit of 10.

### Prior Art

**Inngest** provides:
- **Throttling**: `{ throttle: { limit: 10, period: "1m" } }` — max 10 runs per minute, excess are queued
- **Rate limiting**: `{ rateLimit: { limit: 100, period: "1h", key: "event.data.apiKey" } }` — hard limit per key
- **Debouncing**: `{ debounce: { period: "5s", key: "event.data.userId" } }` — coalesce rapid events, only process the last one

**What Fuse should adopt:** Token-bucket rate limiting at the function level, configurable per external package/function. This controls the rate without blocking the event loop.

### Design

#### 3.3.1 Rate Limit Configuration

```go
// internal/workflow/ratelimit.go

// RateLimitConfig defines rate limiting parameters for a function
type RateLimitConfig struct {
    // Limit is the maximum number of executions allowed per period
    Limit int `json:"limit" bson:"limit" validate:"min=1"`
    // Period is the time window for the rate limit
    Period time.Duration `json:"period" bson:"period"`
    // Key is an optional expression to scope the rate limit (e.g., "input.apiKey")
    Key string `json:"key,omitempty" bson:"key,omitempty"`
    // Strategy defines behavior when limit is exceeded
    Strategy RateLimitStrategy `json:"strategy,omitempty" bson:"strategy,omitempty"`
}

type RateLimitStrategy string

const (
    // RateLimitQueue queues excess requests until a token is available (default)
    RateLimitQueue RateLimitStrategy = "queue"
    // RateLimitReject rejects excess requests immediately
    RateLimitReject RateLimitStrategy = "reject"
)
```

#### 3.3.2 Token Bucket Implementation

```go
// internal/concurrency/token_bucket.go

// TokenBucket implements a token bucket rate limiter
type TokenBucket struct {
    mu         sync.Mutex
    tokens     float64
    maxTokens  float64
    refillRate float64    // tokens per second
    lastRefill time.Time
}

func NewTokenBucket(limit int, period time.Duration) *TokenBucket {
    rate := float64(limit) / period.Seconds()
    return &TokenBucket{
        tokens:     float64(limit),
        maxTokens:  float64(limit),
        refillRate: rate,
        lastRefill: time.Now(),
    }
}

// Take attempts to consume one token. Returns the wait duration if no token is available.
func (tb *TokenBucket) Take() time.Duration {
    tb.mu.Lock()
    defer tb.mu.Unlock()

    tb.refill()

    if tb.tokens >= 1 {
        tb.tokens--
        return 0
    }

    // Calculate wait time until next token
    deficit := 1 - tb.tokens
    waitTime := time.Duration(deficit / tb.refillRate * float64(time.Second))
    return waitTime
}

// Wait blocks until a token is available and consumes it
func (tb *TokenBucket) Wait() {
    for {
        wait := tb.Take()
        if wait == 0 {
            return
        }
        time.Sleep(wait)
    }
}

func (tb *TokenBucket) refill() {
    now := time.Now()
    elapsed := now.Sub(tb.lastRefill).Seconds()
    tb.tokens = math.Min(tb.maxTokens, tb.tokens+elapsed*tb.refillRate)
    tb.lastRefill = now
}
```

#### 3.3.3 Rate Limit Manager

```go
// internal/concurrency/rate_limiter.go

type RateLimiter struct {
    mu      sync.RWMutex
    buckets map[string]*TokenBucket // "functionID" or "functionID:key" -> bucket
}

func NewRateLimiter() *RateLimiter {
    return &RateLimiter{buckets: make(map[string]*TokenBucket)}
}

// Acquire waits until a token is available for the given function
func (rl *RateLimiter) Acquire(functionID string, config RateLimitConfig, keyValue string) error {
    bucketKey := functionID
    if config.Key != "" && keyValue != "" {
        bucketKey = fmt.Sprintf("%s:%s", functionID, keyValue)
    }

    rl.mu.Lock()
    bucket, exists := rl.buckets[bucketKey]
    if !exists {
        bucket = NewTokenBucket(config.Limit, config.Period)
        rl.buckets[bucketKey] = bucket
    }
    rl.mu.Unlock()

    if config.Strategy == RateLimitReject {
        wait := bucket.Take()
        if wait > 0 {
            return fmt.Errorf("rate limit exceeded for %s, retry after %v", functionID, wait)
        }
        return nil
    }

    // Default: queue (block until token available)
    bucket.Wait()
    return nil
}
```

#### 3.3.4 Integration with WorkflowFunc

```go
// In WorkflowFunc.HandleMessage, before function execution:
metadata := a.getMetadata(execMsg.PackageID, execMsg.FunctionID)
if metadata != nil && metadata.RateLimit != nil {
    functionID := fmt.Sprintf("%s/%s", execMsg.PackageID, execMsg.FunctionID)
    if err := a.rateLimiter.Acquire(functionID, *metadata.RateLimit, ""); err != nil {
        // Rate limit exceeded with reject strategy
        result := workflow.NewFunctionResultError(err)
        a.Send(from, messaging.NewFunctionResultMessage(..., result))
        return nil
    }
}
```

#### 3.3.5 Metadata Extension

```go
// Extend FunctionMetadata (internal/packages/function_metadata.go)
type FunctionMetadata struct {
    Transport   transport.Type
    Input       FunctionInputMetadata
    Output      FunctionOutputMetadata
    Concurrency *ConcurrencyConfig `json:"concurrency,omitempty"`
    RateLimit   *RateLimitConfig   `json:"rateLimit,omitempty"` // NEW
}
```

### Alternatives Considered

1. **Sliding window rate limiter**: More accurate than token bucket for bursty traffic but more complex. Token bucket is the industry standard for API rate limiting and handles bursts naturally.

2. **Rate limiting at the HTTP transport level**: Would only apply to HTTP functions. A generic rate limiter at the workflow func level covers all transport types.

3. **Delegating to external rate limiters (nginx, Envoy)**: Doesn't work for outbound calls to external APIs. The rate limiter needs to be at the function execution level.

### Migration Plan

- `RateLimitConfig` is optional on `FunctionMetadata` — no limits by default
- `RateLimiter` registered as a singleton via DI
- Token buckets are in-memory and reset on restart (acceptable for rate limiting — starts fresh)

### Open Questions

1. Should rate limit state be persisted across restarts?
2. Should the rate limiter expose its state via an API (current tokens, queue depth)?
3. How should rate limiting interact with retries — should a retried execution go to the back of the queue?
4. Should there be global rate limits in addition to per-function limits?

---

## 3.4 Idempotency

### Motivation

Currently, calling `POST /v1/workflows/trigger` with the same schema ID creates a **new workflow execution every time**. There is no deduplication. This causes problems when:

- Network retries cause duplicate triggers
- Webhook providers send the same event multiple times (at-least-once delivery)
- Cron triggers fire at startup for missed schedules
- Users accidentally double-click a trigger button

### Prior Art

**Inngest** supports idempotency keys on events: `inngest.send({ name: "order/created", data: {...}, id: "order-123" })`. If an event with the same `id` is already being processed, the duplicate is discarded.

**Restate** provides exactly-once semantics through its invocation model. Each invocation can include an idempotency key, and the system deduplicates requests with the same key.

**What Fuse should adopt:** Optional idempotency keys on workflow triggers. The system tracks recent idempotency keys and returns the existing workflow ID for duplicates.

### Design

#### 3.4.1 Idempotency Key on Trigger

Extend the trigger API to accept an optional idempotency key:

```go
// Trigger request body (internal/dtos/trigger_workflow.go or equivalent)
type TriggerWorkflowRequest struct {
    SchemaID       string `json:"schemaId" validate:"required"`
    IdempotencyKey string `json:"idempotencyKey,omitempty"`
}
```

#### 3.4.2 Idempotency Store

```go
// internal/idempotency/store.go

// Store tracks idempotency keys and their associated workflow IDs
type Store interface {
    // Check returns the workflow ID if the key has been seen, or empty string if new
    Check(key string) (workflowID string, exists bool)
    // Set records an idempotency key with its associated workflow ID and TTL
    Set(key string, workflowID string, ttl time.Duration) error
    // Delete removes an idempotency key (for cleanup)
    Delete(key string) error
}

// MemoryStore is an in-memory implementation with TTL-based expiration
type MemoryStore struct {
    mu      sync.RWMutex
    entries map[string]idempotencyEntry
}

type idempotencyEntry struct {
    workflowID string
    expiresAt  time.Time
}

func NewMemoryStore() Store {
    store := &MemoryStore{entries: make(map[string]idempotencyEntry)}
    go store.cleanup() // Background goroutine for TTL expiration
    return store
}

func (s *MemoryStore) Check(key string) (string, bool) {
    s.mu.RLock()
    defer s.mu.RUnlock()
    entry, exists := s.entries[key]
    if !exists || time.Now().After(entry.expiresAt) {
        return "", false
    }
    return entry.workflowID, true
}

func (s *MemoryStore) Set(key string, workflowID string, ttl time.Duration) error {
    s.mu.Lock()
    defer s.mu.Unlock()
    s.entries[key] = idempotencyEntry{
        workflowID: workflowID,
        expiresAt:  time.Now().Add(ttl),
    }
    return nil
}

func (s *MemoryStore) Delete(key string) error {
    s.mu.Lock()
    defer s.mu.Unlock()
    delete(s.entries, key)
    return nil
}

func (s *MemoryStore) cleanup() {
    ticker := time.NewTicker(1 * time.Minute)
    for range ticker.C {
        s.mu.Lock()
        now := time.Now()
        for key, entry := range s.entries {
            if now.After(entry.expiresAt) {
                delete(s.entries, key)
            }
        }
        s.mu.Unlock()
    }
}
```

#### 3.4.3 Trigger Handler Integration

```go
// In TriggerWorkflowHandler.HandlePost:
func (h *TriggerWorkflowHandler) HandlePost(from gen.PID, w http.ResponseWriter, r *http.Request) error {
    var body TriggerWorkflowRequest
    if err := h.BindJSON(w, r, &body); err != nil {
        return h.SendBadRequest(w, err, EmptyFields)
    }

    // Check idempotency
    if body.IdempotencyKey != "" {
        existingID, exists := h.idempotencyStore.Check(body.IdempotencyKey)
        if exists {
            return h.SendJSON(w, http.StatusOK, map[string]any{
                "workflowId": existingID,
                "status":     "deduplicated",
            })
        }
    }

    workflowID := workflow.NewID()

    // Record idempotency key
    if body.IdempotencyKey != "" {
        h.idempotencyStore.Set(body.IdempotencyKey, workflowID.String(), 24*time.Hour)
    }

    // ... existing trigger logic with workflowID ...
}
```

#### 3.4.4 Extend TriggerWorkflow Message

```go
// Extend TriggerWorkflowMessage
type TriggerWorkflowMessage struct {
    SchemaID       string
    WorkflowID     workflow.ID
    IdempotencyKey string // NEW
    Input          map[string]any // NEW (for event/webhook triggers)
}
```

### Alternatives Considered

1. **Database-level unique constraint**: Use a persistent store with a unique index on idempotency key. More durable but slower. In-memory store is faster for hot-path deduplication; a database provides durability if added later.

2. **Content-hash based deduplication**: Hash the trigger payload and schema ID. Wouldn't work for intentional re-triggers with the same data.

3. **Bloom filter**: Space-efficient but probabilistic (false positives). Not acceptable for idempotency guarantees.

### Migration Plan

- `IdempotencyKey` is optional in the trigger request — existing integrations unaffected
- `IdempotencyStore` registered via DI with in-memory implementation
- Default TTL of 24 hours — configurable via environment variable

### Open Questions

1. What should the default TTL be? 24 hours is common but may be too short for some use cases.
2. Should the idempotency check return the full workflow status (not just ID) so the caller knows if it's still running?
3. Should idempotency keys be schema-scoped or globally unique?
4. Should there be a way to "force" a trigger even with a duplicate idempotency key?

---

## 3.5 Audit Tracing Persistence

### Motivation

This addresses open issue [#34](https://github.com/open-source-cloud/fuse/issues/34) — "Audit tracing (for devmode)."

Currently, execution traces exist in two forms:
1. `AuditLog` (`internal/workflow/audit_log.go`) — in-memory ordered map of execution entries
2. `AppendDebugTraceLine` (`internal/workflow/workflow_status.go:55-62`) — bounded in-memory debug trace (max 256 lines)

Neither is persisted. When a workflow completes and its actor tree is cleaned up (Phase 1.4), the traces are lost. There's no way to inspect historical workflow executions.

### Prior Art

**Inngest** provides a web dashboard with full execution timeline — every step's input, output, duration, retries, and errors are visible. The timeline is stored persistently and queryable.

**n8n** stores execution data per workflow with detailed node-by-node input/output snapshots. Users can view past executions, compare runs, and debug failures.

**Restate** exposes the execution journal through a CLI and SQL-like query interface, allowing operators to inspect and debug invocations.

**What Fuse should adopt:** Persist execution traces as structured data, queryable via API. Integrate with the execution journal (Phase 1.1) so traces include timing, inputs, outputs, and errors.

### Design

#### 3.5.1 Execution Trace Model

```go
// internal/workflow/trace.go

// ExecutionTrace is the complete, persistable trace of a workflow execution
type ExecutionTrace struct {
    WorkflowID  string                  `json:"workflowId" bson:"workflowId"`
    SchemaID    string                  `json:"schemaId" bson:"schemaId"`
    Status      State                   `json:"status" bson:"status"`
    TriggeredAt time.Time               `json:"triggeredAt" bson:"triggeredAt"`
    CompletedAt *time.Time              `json:"completedAt,omitempty" bson:"completedAt,omitempty"`
    Duration    *time.Duration          `json:"duration,omitempty" bson:"duration,omitempty"`
    Steps       []ExecutionStepTrace    `json:"steps" bson:"steps"`
    Error       *string                 `json:"error,omitempty" bson:"error,omitempty"`
}

// ExecutionStepTrace is the trace for a single step (node execution)
type ExecutionStepTrace struct {
    ExecID         string              `json:"execId" bson:"execId"`
    ThreadID       uint16              `json:"threadId" bson:"threadId"`
    FunctionNodeID string              `json:"functionNodeId" bson:"functionNodeId"`
    FunctionID     string              `json:"functionId" bson:"functionId"`
    StartedAt      time.Time           `json:"startedAt" bson:"startedAt"`
    CompletedAt    *time.Time          `json:"completedAt,omitempty" bson:"completedAt,omitempty"`
    Duration       *time.Duration      `json:"duration,omitempty" bson:"duration,omitempty"`
    Input          map[string]any      `json:"input,omitempty" bson:"input,omitempty"`
    Output         *workflow.FunctionOutput `json:"output,omitempty" bson:"output,omitempty"`
    Status         string              `json:"status" bson:"status"` // "running", "completed", "failed", "retrying"
    Attempt        int                 `json:"attempt" bson:"attempt"`
    Error          *string             `json:"error,omitempty" bson:"error,omitempty"`
}
```

#### 3.5.2 Trace Repository

```go
// internal/repositories/trace.go

type TraceRepository interface {
    // Save persists or updates a workflow execution trace
    Save(trace *workflow.ExecutionTrace) error
    // FindByWorkflowID retrieves the trace for a specific workflow execution
    FindByWorkflowID(workflowID string) (*workflow.ExecutionTrace, error)
    // FindBySchemaID retrieves traces for all executions of a schema
    FindBySchemaID(schemaID string, opts TraceQueryOpts) ([]*workflow.ExecutionTrace, error)
    // Delete removes a trace (for retention policy)
    Delete(workflowID string) error
}

type TraceQueryOpts struct {
    Limit  int       // Max results (default 50)
    Offset int       // Pagination offset
    Status *string   // Filter by status
    Since  *time.Time // Only traces after this time
}
```

#### 3.5.3 Trace Builder

Build traces from journal entries:

```go
// internal/workflow/trace_builder.go

// BuildTrace constructs an ExecutionTrace from journal entries
func BuildTrace(workflowID, schemaID string, entries []JournalEntry) *ExecutionTrace {
    trace := &ExecutionTrace{
        WorkflowID: workflowID,
        SchemaID:   schemaID,
        Steps:      make([]ExecutionStepTrace, 0),
    }

    stepMap := make(map[string]*ExecutionStepTrace) // execID -> step

    for _, entry := range entries {
        switch entry.Type {
        case JournalStepStarted:
            step := &ExecutionStepTrace{
                ExecID:         entry.ExecID,
                ThreadID:       entry.ThreadID,
                FunctionNodeID: entry.FunctionNodeID,
                StartedAt:      entry.Timestamp,
                Input:          entry.Input,
                Status:         "running",
                Attempt:        1,
            }
            stepMap[entry.ExecID] = step
            trace.Steps = append(trace.Steps, *step)

        case JournalStepCompleted:
            if step, ok := stepMap[entry.ExecID]; ok {
                step.CompletedAt = &entry.Timestamp
                dur := entry.Timestamp.Sub(step.StartedAt)
                step.Duration = &dur
                step.Output = &entry.Result.Output
                step.Status = "completed"
                updateStep(trace, step)
            }

        case JournalStepFailed:
            if step, ok := stepMap[entry.ExecID]; ok {
                step.CompletedAt = &entry.Timestamp
                dur := entry.Timestamp.Sub(step.StartedAt)
                step.Duration = &dur
                step.Status = "failed"
                errStr := fmt.Sprintf("%v", entry.Result.Output.Data["error"])
                step.Error = &errStr
                updateStep(trace, step)
            }

        case JournalStateChanged:
            if entry.State == StateRunning && trace.TriggeredAt.IsZero() {
                trace.TriggeredAt = entry.Timestamp
            }
            trace.Status = entry.State
            if isTerminalState(entry.State) {
                trace.CompletedAt = &entry.Timestamp
                dur := entry.Timestamp.Sub(trace.TriggeredAt)
                trace.Duration = &dur
            }
        }
    }

    return trace
}
```

#### 3.5.4 HTTP API

```
GET /v1/workflows/{workflowID}/trace
Response 200:
{
    "workflowId": "...",
    "schemaId": "...",
    "status": "finished",
    "triggeredAt": "2025-07-28T10:00:00Z",
    "completedAt": "2025-07-28T10:00:05Z",
    "duration": "5s",
    "steps": [
        {
            "execId": "...",
            "threadId": 0,
            "functionNodeId": "node1",
            "functionId": "http/get",
            "startedAt": "2025-07-28T10:00:00Z",
            "completedAt": "2025-07-28T10:00:02Z",
            "duration": "2s",
            "input": { "url": "https://api.example.com" },
            "output": { "status": "success", "data": { ... } },
            "status": "completed",
            "attempt": 1
        }
    ]
}

GET /v1/schemas/{schemaID}/traces?limit=10&status=error
Response 200:
{
    "traces": [ ... ],
    "total": 42,
    "limit": 10,
    "offset": 0
}
```

#### 3.5.5 Trace Handler

```go
// internal/handlers/workflow_trace.go

const (
    WorkflowTraceHandlerName     = "workflow_trace_handler"
    WorkflowTraceHandlerPoolName = "workflow_trace_handler_pool"
)

type WorkflowTraceHandlerFactory HandlerFactory[*WorkflowTraceHandler]

type WorkflowTraceHandler struct {
    Handler
    traceRepo repositories.TraceRepository
}

func (h *WorkflowTraceHandler) HandleGet(from gen.PID, w http.ResponseWriter, r *http.Request) error {
    workflowID, err := h.GetPathParam(r, "workflowID")
    if err != nil {
        return h.SendBadRequest(w, err, EmptyFields)
    }

    trace, err := h.traceRepo.FindByWorkflowID(workflowID)
    if err != nil {
        return h.SendNotFound(w, "trace not found", EmptyFields)
    }

    return h.SendJSON(w, http.StatusOK, trace)
}
```

#### 3.5.6 Trace Persistence Integration

In `WorkflowHandler`, persist trace on completion:

```go
// After setting terminal state (Finished/Error/Cancelled):
trace := workflow.BuildTrace(
    a.workflow.ID().String(),
    a.workflow.Schema().ID,
    a.workflow.Journal().Entries(),
)
if err := a.traceRepo.Save(trace); err != nil {
    a.Log().Error("failed to persist execution trace: %s", err)
}
```

#### 3.5.7 Retention Policy

```go
// internal/workflow/trace.go

type TraceRetentionConfig struct {
    // MaxAge is the maximum age of traces before they're eligible for deletion
    MaxAge time.Duration `json:"maxAge,omitempty"`
    // MaxCount is the maximum number of traces per schema to retain
    MaxCount int `json:"maxCount,omitempty"`
}
```

A background actor periodically prunes old traces based on the retention policy.

### Alternatives Considered

1. **OpenTelemetry integration**: Emit traces as OpenTelemetry spans to be collected by Jaeger/Zipkin. Powerful for distributed tracing but requires external infrastructure. Can be added as an exporter on top of the internal trace model.

2. **Log-only traces**: Write traces to structured logs (zerolog). Simpler but not queryable without a log aggregation system.

3. **Event store (event sourcing)**: Store all events and reconstruct traces on demand. More flexible but slower for queries. Pre-computed traces are faster to serve.

### Migration Plan

- New `TraceRepository` implementations (memory first; persistent adapter optional)
- New HTTP endpoints added to router
- Trace building from journal entries means this depends on Phase 1.1
- For workflows that predate journaling, traces will only be available after the journal is implemented

### Open Questions

1. Should traces include the full input/output data, or redact sensitive fields?
2. What's the default retention policy (time-based, count-based, or both)?
3. Should there be a "live trace" endpoint (streaming/polling) for running workflows?
4. Should traces be exportable in OpenTelemetry format for integration with existing observability tools?
