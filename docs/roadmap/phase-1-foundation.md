# Phase 1: Foundation — Make It Reliable

> **Goal:** Establish the durability, error handling, and validation foundations required for production use. Every subsequent phase depends on these primitives.

---

## 1.1 Durable Execution / Workflow Resumption

### Motivation

Today, if the Fuse process crashes or restarts, **all running workflows are lost**. The `Resume()` method in `internal/workflow/workflow.go:94-97` is a stub that returns nil:

```go
func (w *Workflow) Resume() workflowactions.Action {
    // TODO add logic to re-start an already started Workflow that got reloaded from storage
    return nil
}
```

The `WorkflowHandler` actor (`internal/actors/workflow_handler.go:88-96`) already has a code path that calls `Resume()` when loading an existing workflow that isn't in `StateUntriggered`, but since `Resume()` returns nil, this is a no-op that will cause a nil pointer panic when `handleWorkflowAction` tries to call `action.Type()`.

Without durable execution, Fuse cannot guarantee workflow completion — the most fundamental requirement for a workflow engine.

### Prior Art

**Restate** journals every step's result using an append-only execution journal. On failure, the runtime replays the journal deterministically — completed steps return their journaled result instead of re-executing. Non-deterministic operations (API calls, DB writes) are wrapped in `ctx.run()` to ensure idempotency during replay. The journal is the single source of truth for execution state.

**Inngest** treats each step as a durable checkpoint. Steps are individually retryable, and the function resumes from the last completed step on failure. State between steps is serialized and persisted by the platform, making functions stateless from the developer's perspective.

**What Fuse should adopt:** Restate's journaling model is the best fit for Fuse's architecture. The existing `AuditLog` (`internal/workflow/audit_log.go`) already records each step's result in execution order — it's structurally close to an execution journal but lacks persistence and replay capability.

**What Fuse should adapt:** Unlike Restate (which replays code), Fuse should replay at the graph traversal level — reconstruct the workflow's thread states and aggregated output from the journal, then resume `Next()` from where execution left off.

### Design

#### 1.1.1 Execution Journal

Extend the existing `AuditLog` to serve as a persistent execution journal. Each entry records enough state to replay execution.

```go
// internal/workflow/journal.go

// JournalEntryType classifies what happened at this step
type JournalEntryType string

const (
    JournalStepStarted   JournalEntryType = "step:started"
    JournalStepCompleted JournalEntryType = "step:completed"
    JournalStepFailed    JournalEntryType = "step:failed"
    JournalStepRetrying  JournalEntryType = "step:retrying"
    JournalThreadCreated JournalEntryType = "thread:created"
    JournalThreadDone    JournalEntryType = "thread:finished"
    JournalStateChanged  JournalEntryType = "state:changed"
)

// JournalEntry is a single recorded event in the execution journal
type JournalEntry struct {
    Sequence       uint64           `json:"sequence" bson:"sequence"`
    Timestamp      time.Time        `json:"timestamp" bson:"timestamp"`
    Type           JournalEntryType `json:"type" bson:"type"`
    ThreadID       uint16           `json:"threadId" bson:"threadId"`
    FunctionNodeID string           `json:"functionNodeId,omitempty" bson:"functionNodeId,omitempty"`
    ExecID         string           `json:"execId,omitempty" bson:"execId,omitempty"`
    Input          map[string]any   `json:"input,omitempty" bson:"input,omitempty"`
    Result         *FunctionResult  `json:"result,omitempty" bson:"result,omitempty"`
    State          State            `json:"state,omitempty" bson:"state,omitempty"`
    ParentThreads  []uint16         `json:"parentThreads,omitempty" bson:"parentThreads,omitempty"`
}

// Journal is an append-only execution log that enables replay
type Journal struct {
    mu      sync.Mutex
    entries []JournalEntry
    seq     uint64
}

func NewJournal() *Journal {
    return &Journal{entries: make([]JournalEntry, 0, 32)}
}

func (j *Journal) Append(entry JournalEntry) {
    j.mu.Lock()
    defer j.mu.Unlock()
    j.seq++
    entry.Sequence = j.seq
    entry.Timestamp = time.Now()
    j.entries = append(j.entries, entry)
}

func (j *Journal) Entries() []JournalEntry {
    j.mu.Lock()
    defer j.mu.Unlock()
    cp := make([]JournalEntry, len(j.entries))
    copy(cp, j.entries)
    return cp
}

// LoadFrom replaces journal contents with persisted entries (for replay)
func (j *Journal) LoadFrom(entries []JournalEntry) {
    j.mu.Lock()
    defer j.mu.Unlock()
    j.entries = entries
    if len(entries) > 0 {
        j.seq = entries[len(entries)-1].Sequence
    }
}
```

#### 1.1.2 Journal Persistence

Add a `JournalRepository` to persist journal entries:

```go
// internal/repositories/journal.go

type JournalRepository interface {
    // Append persists one or more journal entries for a workflow
    Append(workflowID string, entries ...workflow.JournalEntry) error

    // LoadAll retrieves the full journal for a workflow, ordered by sequence
    LoadAll(workflowID string) ([]workflow.JournalEntry, error)

    // LastSequence returns the highest sequence number for a workflow
    LastSequence(workflowID string) (uint64, error)
}
```

Memory and MongoDB implementations follow the existing repository pattern (`_memory.go`, `_mongo.go`).

#### 1.1.3 Recording Journal Entries

Instrument `Workflow` methods to write journal entries at each step boundary:

- `Trigger()` → append `JournalThreadCreated` + `JournalStepStarted`
- `SetResultFor()` → append `JournalStepCompleted` or `JournalStepFailed`
- `Next()` → when creating new threads, append `JournalThreadCreated`; when a thread finishes, append `JournalThreadDone`
- `SetState()` → append `JournalStateChanged`

The `Workflow` struct gains a `journal *Journal` field:

```go
type Workflow struct {
    mu               sync.RWMutex
    debugMu          sync.Mutex
    debugLines       []string
    id               workflow.ID
    graph            *Graph
    journal          *Journal         // NEW
    auditLog         *AuditLog
    threads          *threads
    aggregatedOutput *store.KV
    state            RunningState
}
```

#### 1.1.4 Replay / Resume

Implement `Resume()` by replaying the journal to reconstruct state:

```go
func (w *Workflow) Resume() workflowactions.Action {
    entries := w.journal.Entries()
    if len(entries) == 0 {
        return &workflowactions.NoopAction{}
    }

    // Phase 1: Replay — reconstruct threads, aggregatedOutput, auditLog
    var lastCompletedThreadIDs []uint16
    for _, entry := range entries {
        switch entry.Type {
        case JournalThreadCreated:
            execID := workflow.ExecIDFromString(entry.ExecID)
            w.threads.New(entry.ThreadID, execID)
        case JournalStepStarted:
            w.auditLog.NewEntry(entry.ThreadID, entry.FunctionNodeID, entry.ExecID, entry.Input)
        case JournalStepCompleted:
            w.SetResultFor(workflow.ExecIDFromString(entry.ExecID), entry.Result)
        case JournalThreadDone:
            t := w.threads.Get(entry.ThreadID)
            if t != nil {
                t.SetState(StateFinished)
            }
            lastCompletedThreadIDs = append(lastCompletedThreadIDs, entry.ThreadID)
        case JournalStateChanged:
            w.state.currentState = entry.State
        }
    }

    // Phase 2: Determine next action for threads that were in-progress
    // Find threads that have a StepStarted but no StepCompleted
    // These are the threads that need to be re-executed
    pendingThreads := w.findPendingThreads(entries)
    if len(pendingThreads) == 0 {
        // All threads completed during replay — try Next() on the last finished threads
        for _, threadID := range lastCompletedThreadIDs {
            action := w.Next(threadID)
            if action.Type() != workflowactions.ActionNoop {
                return action
            }
        }
        return &workflowactions.NoopAction{}
    }

    // Re-execute pending steps
    if len(pendingThreads) == 1 {
        return w.replayPendingThread(pendingThreads[0])
    }
    parallel := &workflowactions.RunParallelFunctionsAction{
        Actions: make([]*workflowactions.RunFunctionAction, 0, len(pendingThreads)),
    }
    for _, pt := range pendingThreads {
        action := w.replayPendingThread(pt)
        if runAction, ok := action.(*workflowactions.RunFunctionAction); ok {
            parallel.Actions = append(parallel.Actions, runAction)
        }
    }
    return parallel
}
```

#### 1.1.5 Workflow Handler Integration

The `WorkflowHandler.Init()` already handles the existing/new workflow split. For existing workflows, it calls `Resume()`. The handler also needs to persist journal entries after each mutation:

```go
// In handleMsgFunctionResult, after SetResultFor and Next:
if err := a.journalRepo.Append(a.workflow.ID().String(), newEntries...); err != nil {
    a.Log().Error("failed to persist journal: %s", err)
}
```

#### 1.1.6 Startup Recovery

On application startup, the `WorkflowSupervisor` should query the `WorkflowRepository` for workflows in `StateRunning` or `StateSleeping` and spawn `WorkflowInstanceSupervisor` actors for each, which will trigger `Resume()`.

### Alternatives Considered

1. **Snapshot-based persistence** (persist full workflow state periodically): Simpler but loses granularity. If a crash happens between snapshots, in-flight steps are lost. Journaling is more precise and enables step-level replay.

2. **Event sourcing** (full CQRS with event store): More powerful but significantly more complex. The journal approach gives us 90% of the benefit with a fraction of the complexity. Can migrate to full event sourcing later if needed.

3. **External orchestrator** (delegate persistence to an external system like Temporal): Would solve durability but defeats Fuse's purpose as a self-contained engine.

### Migration Plan

- Journal field added to `Workflow` struct — existing workflows created without a journal get an empty one (backwards compatible)
- `JournalRepository` implementations added alongside existing repos
- `AuditLog` remains for API-facing execution history; `Journal` is the internal replay log
- No schema migrations needed for `GraphSchema` or `NodeSchema`

### Open Questions

1. Should journal entries be batched before persistence (performance) or written synchronously per entry (consistency)?
2. What is the journal retention policy? Keep forever, or prune after workflow completion?
3. Should we support "exactly-once" semantics for function execution during replay, or accept "at-least-once" with idempotency guidance?

---

## 1.2 Retry & Error Recovery

### Motivation

Today, any function failure immediately terminates the workflow. In `workflow_handler.go:167-179`:

```go
if fnResultMsg.Result.Output.Status != workflow.FunctionSuccess {
    a.workflow.SetState(internalworkflow.StateError)
    return
}
```

There is no retry logic, no backoff, no error handling edges. This means a single transient network error (e.g., an HTTP timeout to an external function) permanently fails the entire workflow. Production workflows need resilience against transient failures.

### Prior Art

**Inngest** provides automatic step-level retries with configurable backoff. Each step can specify `retries: { attempts: 3, backoff: { type: "exponential", delay: "1s" } }`. Failed steps retry independently without re-executing prior steps (enabled by durable execution).

**Restate** automatically retries failed steps with exponential backoff. The journal ensures completed steps aren't re-executed. Retries are transparent to the developer — the `ctx.run()` wrapper handles everything.

**n8n** takes a different approach: **Error Trigger** nodes catch workflow failures and route to recovery flows. There's also a "Stop And Error" node for explicit error termination. This enables compensation patterns (e.g., if step 3 fails, run a cleanup flow).

**What Fuse should adopt:**
- Inngest/Restate-style automatic retries with configurable policies per node
- n8n-style error edges for routing to recovery flows when retries are exhausted

### Design

#### 1.2.1 Retry Policy Configuration

Add retry configuration to function metadata and node schema:

```go
// internal/workflow/retry.go

// BackoffType defines the backoff strategy
type BackoffType string

const (
    BackoffFixed       BackoffType = "fixed"
    BackoffExponential BackoffType = "exponential"
    BackoffLinear      BackoffType = "linear"
)

// RetryPolicy defines how a node should handle failures
type RetryPolicy struct {
    // MaxAttempts is the maximum number of retry attempts (0 = no retries)
    MaxAttempts int `json:"maxAttempts" bson:"maxAttempts" validate:"min=0,max=100"`

    // Backoff strategy
    Backoff BackoffConfig `json:"backoff" bson:"backoff"`
}

// BackoffConfig defines the backoff parameters
type BackoffConfig struct {
    // Type of backoff: fixed, exponential, linear
    Type BackoffType `json:"type" bson:"type" validate:"oneof=fixed exponential linear"`

    // InitialInterval is the base delay between retries
    InitialInterval time.Duration `json:"initialInterval" bson:"initialInterval"`

    // MaxInterval caps the delay for exponential/linear backoff
    MaxInterval time.Duration `json:"maxInterval" bson:"maxInterval"`

    // Multiplier for exponential backoff (default: 2.0)
    Multiplier float64 `json:"multiplier,omitempty" bson:"multiplier,omitempty"`
}

// DefaultRetryPolicy returns a sensible default (3 attempts, exponential backoff)
func DefaultRetryPolicy() RetryPolicy {
    return RetryPolicy{
        MaxAttempts: 3,
        Backoff: BackoffConfig{
            Type:            BackoffExponential,
            InitialInterval: 1 * time.Second,
            MaxInterval:     30 * time.Second,
            Multiplier:      2.0,
        },
    }
}

// DelayFor calculates the delay for a given attempt number (0-indexed)
func (p RetryPolicy) DelayFor(attempt int) time.Duration {
    switch p.Backoff.Type {
    case BackoffExponential:
        delay := float64(p.Backoff.InitialInterval) * math.Pow(p.Backoff.Multiplier, float64(attempt))
        if time.Duration(delay) > p.Backoff.MaxInterval {
            return p.Backoff.MaxInterval
        }
        return time.Duration(delay)
    case BackoffLinear:
        delay := p.Backoff.InitialInterval * time.Duration(attempt+1)
        if delay > p.Backoff.MaxInterval {
            return p.Backoff.MaxInterval
        }
        return delay
    default: // fixed
        return p.Backoff.InitialInterval
    }
}
```

#### 1.2.2 Node Schema Extension

Add `RetryPolicy` and `ErrorEdge` to the node/edge schemas:

```go
// Extend NodeSchema (internal/workflow/graph_schema.go or node_schema.go)
type NodeSchema struct {
    ID         string       `json:"id" bson:"id"`
    FunctionID string       `json:"functionId" bson:"functionId"`
    Config     NodeConfig   `json:"config,omitempty" bson:"config,omitempty"`
    Retry      *RetryPolicy `json:"retry,omitempty" bson:"retry,omitempty"` // NEW
}

// Extend EdgeSchema
type EdgeSchema struct {
    ID          string         `json:"id" bson:"id"`
    From        string         `json:"from" bson:"from"`
    To          string         `json:"to" bson:"to"`
    Conditional *EdgeCondition `json:"conditional,omitempty" bson:"conditional,omitempty"`
    Input       []InputMapping `json:"input,omitempty" bson:"input,omitempty"`
    OnError     bool           `json:"onError,omitempty" bson:"onError,omitempty"` // NEW
}
```

#### 1.2.3 Retry State Tracking

Track retry attempts per execution step:

```go
// internal/workflow/retry_tracker.go

type RetryTracker struct {
    mu       sync.Mutex
    attempts map[string]int // execID -> attempt count
}

func NewRetryTracker() *RetryTracker {
    return &RetryTracker{attempts: make(map[string]int)}
}

func (rt *RetryTracker) Increment(execID string) int {
    rt.mu.Lock()
    defer rt.mu.Unlock()
    rt.attempts[execID]++
    return rt.attempts[execID]
}

func (rt *RetryTracker) GetAttempts(execID string) int {
    rt.mu.Lock()
    defer rt.mu.Unlock()
    return rt.attempts[execID]
}

func (rt *RetryTracker) Clear(execID string) {
    rt.mu.Lock()
    defer rt.mu.Unlock()
    delete(rt.attempts, execID)
}
```

#### 1.2.4 Error Edge Routing

When retries are exhausted and the node has error edges, route to the error handling path:

```go
// In workflow.go, new method:

func (w *Workflow) HandleNodeFailure(threadID uint16, execID workflow.ExecID, result *workflow.FunctionResult) workflowactions.Action {
    entry, _ := w.auditLog.Get(execID.String())
    node, _ := w.graph.FindNode(entry.FunctionNodeID)

    // Check retry policy
    retryPolicy := w.getRetryPolicy(node)
    attempts := w.retryTracker.Increment(execID.String())

    if attempts <= retryPolicy.MaxAttempts {
        delay := retryPolicy.DelayFor(attempts - 1)
        return &workflowactions.RetryFunctionAction{
            RunFunctionAction: workflowactions.RunFunctionAction{
                ThreadID:       threadID,
                FunctionID:     node.FunctionID(),
                FunctionExecID: execID,
                Args:           entry.Input,
            },
            Delay:   delay,
            Attempt: attempts,
        }
    }

    // Retries exhausted — check for error edges
    errorEdges := w.findErrorEdges(node)
    if len(errorEdges) > 0 {
        w.retryTracker.Clear(execID.String())
        // Route to error handling subgraph
        return w.newRunFunctionAction(w.threads.Get(threadID), errorEdges[0])
    }

    // No error edges — fail the workflow
    w.retryTracker.Clear(execID.String())
    return nil // caller sets StateError
}
```

#### 1.2.5 New Action Type

```go
// internal/workflow/workflowactions/action.go

const ActionRetryFunction ActionType = "function:retry"

type RetryFunctionAction struct {
    RunFunctionAction
    Delay   time.Duration
    Attempt int
}

func (a *RetryFunctionAction) Type() ActionType { return ActionRetryFunction }
```

#### 1.2.6 WorkflowHandler Integration

In `handleMsgFunctionResult`, replace the immediate `StateError` transition:

```go
// Before (current):
if fnResultMsg.Result.Output.Status != workflow.FunctionSuccess {
    a.workflow.SetState(internalworkflow.StateError)
    return
}

// After (new):
if fnResultMsg.Result.Output.Status != workflow.FunctionSuccess {
    action := a.workflow.HandleNodeFailure(fnResultMsg.ThreadID, fnResultMsg.ExecID, &fnResultMsg.Result)
    if action == nil {
        a.workflow.SetState(internalworkflow.StateError)
        return
    }
    a.handleWorkflowAction(action)
    return
}
```

The handler dispatches `RetryFunctionAction` by scheduling a delayed re-send using ergo's timer facilities:

```go
case workflowactions.ActionRetryFunction:
    retryAction := action.(*workflowactions.RetryFunctionAction)
    // Schedule retry after delay using ergo timer
    a.SendAfter(a.PID(), retryMsg, retryAction.Delay)
```

### Alternatives Considered

1. **Global retry policy only**: Simpler but too coarse. Different functions have different failure characteristics — an HTTP call to an unreliable API needs more retries than an in-memory computation.

2. **External retry orchestrator** (e.g., separate retry queue): Adds architectural complexity. Since Fuse already has actors and timers via ergo, using `SendAfter` for retry delays is natural and doesn't require additional infrastructure.

3. **No error edges, only retries**: Insufficient for compensation patterns. Sometimes you need to run cleanup logic when a step fails permanently, not just retry the same step.

### Migration Plan

- `RetryPolicy` is optional on `NodeSchema` — existing schemas work unchanged (nil = no retries, matches current behavior)
- `OnError` flag on `EdgeSchema` defaults to false — existing edges are unaffected
- `RetryTracker` is workflow-internal state, no persistence changes needed for existing workflows

### Open Questions

1. Should retry delay use ergo's `SendAfter` or a separate timer mechanism? `SendAfter` is natural but ties retry scheduling to the actor's lifecycle.
2. Should error edges receive the original input + the error details, or just the error?
3. Should there be a global max-retries cap to prevent runaway retry storms?

---

## 1.3 Timeouts

### Motivation

Currently, there is **no timeout mechanism** anywhere in the execution path. If an external HTTP function hangs, the workflow waits indefinitely. If an async function's callback never arrives, the workflow stays in `StateRunning` forever.

The `WorkflowFuncPool` has a fixed pool size of 3 workers (`internal/actors/workflow_func_pool.go:43`), meaning one hung function can starve other functions waiting for execution within the same workflow.

### Prior Art

**Inngest** supports both step-level timeouts and function-level timeouts. If a step exceeds its timeout, it's automatically failed and retried (or the function fails if retries are exhausted).

**Restate** provides configurable call timeouts for service interactions. The framework itself tracks execution duration and can fail hung operations.

**n8n** has execution timeout settings at the workflow level and environment level, preventing runaway workflows from consuming resources indefinitely.

**What Fuse should adopt:** Two-level timeouts — per-node (function execution timeout) and per-workflow (total execution timeout). Per-node is most critical for external/HTTP functions; per-workflow provides a safety net for complex graphs.

### Design

#### 1.3.1 Timeout Configuration

```go
// internal/workflow/timeout.go

// TimeoutConfig defines timeout settings for a node or workflow
type TimeoutConfig struct {
    // Execution timeout for this node's function
    // Zero means no timeout (not recommended for external functions)
    Execution time.Duration `json:"execution,omitempty" bson:"execution,omitempty"`
}

// WorkflowTimeoutConfig defines timeout settings at the workflow level
type WorkflowTimeoutConfig struct {
    // Total maximum duration for the entire workflow execution
    // Zero means no timeout
    Total time.Duration `json:"total,omitempty" bson:"total,omitempty"`
}
```

#### 1.3.2 Schema Extensions

```go
// Extend NodeSchema
type NodeSchema struct {
    ID         string         `json:"id" bson:"id"`
    FunctionID string         `json:"functionId" bson:"functionId"`
    Config     NodeConfig     `json:"config,omitempty" bson:"config,omitempty"`
    Retry      *RetryPolicy   `json:"retry,omitempty" bson:"retry,omitempty"`
    Timeout    *TimeoutConfig `json:"timeout,omitempty" bson:"timeout,omitempty"` // NEW
}

// Extend GraphSchema
type GraphSchema struct {
    ID      string                 `json:"id" bson:"id"`
    Trigger string                 `json:"trigger" bson:"trigger"`
    Nodes   []*NodeSchema          `json:"nodes" bson:"nodes"`
    Edges   []*EdgeSchema          `json:"edges" bson:"edges"`
    Timeout *WorkflowTimeoutConfig `json:"timeout,omitempty" bson:"timeout,omitempty"` // NEW
}
```

#### 1.3.3 Node Execution Timeout

The `WorkflowHandler` tracks pending executions with deadlines:

```go
// internal/actors/execution_timer.go

type ExecutionTimer struct {
    mu       sync.Mutex
    timers   map[string]gen.CancelFunc // execID -> cancel function
}

func NewExecutionTimer() *ExecutionTimer {
    return &ExecutionTimer{timers: make(map[string]gen.CancelFunc)}
}

// StartTimer begins a timeout countdown for an execution.
// When the timeout fires, it sends a TimeoutMessage to the handler.
func (et *ExecutionTimer) Start(actor interface{ SendAfter(gen.PID, any, time.Duration) gen.CancelFunc },
    self gen.PID, execID string, timeout time.Duration) {
    et.mu.Lock()
    defer et.mu.Unlock()
    cancel := actor.SendAfter(self, messaging.NewTimeoutMessage(execID), timeout)
    et.timers[execID] = cancel
}

// Cancel stops a pending timeout (called when the function completes in time)
func (et *ExecutionTimer) Cancel(execID string) {
    et.mu.Lock()
    defer et.mu.Unlock()
    if cancel, exists := et.timers[execID]; exists {
        cancel()
        delete(et.timers, execID)
    }
}
```

#### 1.3.4 Timeout Message Type

```go
// internal/messaging/timeout.go

const Timeout MessageType = "execution:timeout"

type TimeoutMessage struct {
    ExecID string
}

func NewTimeoutMessage(execID string) Message {
    return Message{
        Type: Timeout,
        Args: TimeoutMessage{ExecID: execID},
    }
}
```

#### 1.3.5 WorkflowHandler Integration

```go
// In WorkflowHandler.HandleMessage, add timeout case:
case messaging.Timeout:
    return a.handleMsgTimeout(msg)

// Handler method:
func (a *WorkflowHandler) handleMsgTimeout(msg messaging.Message) error {
    timeoutMsg, ok := msg.Args.(messaging.TimeoutMessage)
    if !ok {
        return nil
    }

    a.workflow.RunExclusive(func() {
        // Create a timeout error result
        result := &workflow.FunctionResult{
            Output: workflow.FunctionOutput{
                Status: workflow.FunctionError,
                Data:   map[string]any{"error": "execution timeout exceeded"},
            },
        }
        // Feed through the same error handling path (retry/error edge)
        a.workflow.SetResultFor(workflow.ExecIDFromString(timeoutMsg.ExecID), result)
        action := a.workflow.HandleNodeFailure(
            workflow.ExecIDFromString(timeoutMsg.ExecID).Thread(),
            workflow.ExecIDFromString(timeoutMsg.ExecID),
            result,
        )
        if action == nil {
            a.workflow.SetState(internalworkflow.StateError)
            return
        }
        a.handleWorkflowAction(action)
    })
    return nil
}
```

#### 1.3.6 Workflow-Level Timeout

On `WorkflowHandler.Init()`, if the graph schema specifies a total timeout, set a single timer:

```go
if schema.Timeout != nil && schema.Timeout.Total > 0 {
    a.SendAfter(a.PID(), messaging.NewWorkflowTimeoutMessage(a.workflow.ID()), schema.Timeout.Total)
}
```

When fired, the workflow transitions to `StateError` with a "workflow timeout" reason.

### Alternatives Considered

1. **Context-based timeouts** (Go `context.WithTimeout`): Natural for Go code, but Fuse's actor model doesn't pass contexts between actors. Ergo's `SendAfter` is the idiomatic approach for actor-based timeout management.

2. **HTTP client-level timeouts only**: Would handle HTTP function timeouts but not internal functions or async functions. A generic solution at the workflow layer covers all cases.

3. **External watchdog process**: Over-engineered for this use case. Actor-based timers are sufficient and don't require additional infrastructure.

### Migration Plan

- `TimeoutConfig` is optional on `NodeSchema` — existing schemas have no timeout (matches current behavior)
- `WorkflowTimeoutConfig` is optional on `GraphSchema` — existing schemas are unaffected
- Default timeout values could be set via environment configuration for external functions

### Open Questions

1. Should there be a global default timeout for all external (HTTP) functions to prevent forgotten timeout configurations?
2. How should timeouts interact with async functions? Should the timeout fire from initial send or from when the async callback is expected?
3. Should workflow-level timeout be configurable per-trigger (different SLAs for different callers)?

---

## 1.4 Graceful Workflow Completion

### Motivation

When all threads finish, the workflow transitions to `StateFinished` or `StateError`, but the actor tree (WorkflowInstanceSupervisor, WorkflowHandler, WorkflowFuncPool) is **never cleaned up**. This is noted as a TODO in `workflow.go:109`:

```go
case 0:
    currentThread.SetState(StateFinished)
    // TODO: if ALL threads are finished, finish actor-tree for this workflow
    return &workflowactions.NoopAction{}
```

And the handler detects completion in `workflow_handler.go:183-192`:

```go
if a.workflow.AllThreadsFinished() {
    a.workflow.SetState(internalworkflow.StateFinished)
} else {
    a.Log().Warning("got noop action from workflow")
}
```

But no cleanup follows. Over time, completed workflow actors accumulate, consuming memory and actor system resources.

### Prior Art

**n8n** cleans up execution state after workflow completion, maintaining execution history separately from the runtime state.

**Inngest** automatically cleans up function execution state after completion, retaining only the execution history for observability.

**Restate** manages virtual object lifecycle — workflow-type services are cleaned up after completion with configurable retention periods for their state.

**What Fuse should adopt:** Persist final state, then shut down the actor tree. The `WorkflowSupervisor` should remove the workflow from its active actors map.

### Design

#### 1.4.1 Completion Flow

```
WorkflowHandler detects AllThreadsFinished()
  |
  v
Set StateFinished/StateError
  |
  v
Persist final journal entry (JournalStateChanged)
  |
  v
Persist final workflow state to WorkflowRepository
  |
  v
Send WorkflowCompleted message to WorkflowInstanceSupervisor
  |
  v
WorkflowInstanceSupervisor stops its children and itself
  |
  v
WorkflowSupervisor receives child termination event
  |
  v
Remove workflow from workflowActors map
```

#### 1.4.2 Completion Message

```go
// internal/messaging/workflow_completed.go

const WorkflowCompleted MessageType = "workflow:completed"

type WorkflowCompletedMessage struct {
    WorkflowID workflow.ID
    FinalState workflow.State
}

func NewWorkflowCompletedMessage(workflowID workflow.ID, state workflow.State) Message {
    return Message{
        Type: WorkflowCompleted,
        Args: WorkflowCompletedMessage{WorkflowID: workflowID, FinalState: state},
    }
}
```

#### 1.4.3 WorkflowHandler Changes

```go
// In handleMsgFunctionResult, after detecting completion:
if a.workflow.AllThreadsFinished() {
    a.workflow.SetState(internalworkflow.StateFinished)
    a.persistFinalState()
    // Notify supervisor that this workflow is done
    supName := WorkflowInstanceSupervisorName(a.workflow.ID())
    a.Send(gen.Atom(supName), messaging.NewWorkflowCompletedMessage(
        a.workflow.ID(), internalworkflow.StateFinished,
    ))
}
```

#### 1.4.4 WorkflowInstanceSupervisor Handling

```go
func (s *WorkflowInstanceSupervisor) HandleMessage(from gen.PID, message any) error {
    msg, ok := message.(messaging.Message)
    if !ok {
        return nil
    }
    if msg.Type == messaging.WorkflowCompleted {
        s.Log().Info("workflow completed, shutting down actor tree")
        return gen.TerminateReasonNormal
    }
    return nil
}
```

#### 1.4.5 WorkflowSupervisor Cleanup

The `WorkflowSupervisor` already receives termination events via `HandleEvent`. Add cleanup of the `workflowActors` map:

```go
func (s *WorkflowSupervisor) HandleEvent(event gen.MessageEvent) error {
    // When a child workflow instance supervisor terminates
    for wfID, pid := range s.workflowActors {
        if pid == event.PID {
            delete(s.workflowActors, wfID)
            s.Log().Info("cleaned up workflow actor %s", wfID)
            break
        }
    }
    return nil
}
```

### Alternatives Considered

1. **TTL-based cleanup** (keep actors alive for N minutes after completion for late queries): Adds complexity and still consumes resources. Better to persist state and serve queries from the repository.

2. **Actor pool recycling** (reuse the same actors for new workflows): Complicates actor state management. The ergo model of spawn-per-workflow is clean; just need to add the cleanup side.

### Migration Plan

- No schema changes needed
- Existing running workflows won't be cleaned up (they'll still accumulate), but new workflows will clean up properly
- A startup reconciliation (Phase 1.1 recovery) can identify orphaned completed workflows and clean them up

### Open Questions

1. Should there be a configurable grace period before cleanup (to handle late async results)?
2. Should the `WorkflowSupervisor.workflowActors` map be persisted for crash recovery, or reconstructed from the `WorkflowRepository`?

---

## 1.5 Input Mapping Validation

### Motivation

The `validateInputMapping` method in `workflow.go:378-381` is a stub:

```go
func (w *Workflow) validateInputMapping(_ *workflow.ParameterSchema, _ any) bool {
    // TODO implement input mapping validations
    return true
}
```

This means invalid data (wrong types, missing required fields) passes silently between nodes. A function expecting an integer could receive a string, causing a runtime panic inside the function rather than a clean validation error at the workflow level.

### Prior Art

**n8n** validates node inputs against expected schemas and shows validation errors in the UI.

**Inngest** validates event schemas using JSON Schema / Zod schemas at the event ingestion layer.

**Restate** relies on language-level type safety (TypeScript, Java) plus serialization validation.

**What Fuse should adopt:** Validate input mappings against the `ParameterSchema` type definitions that already exist in the codebase (`pkg/workflow/metadata.go:68-74`).

### Design

#### 1.5.1 Validation Implementation

```go
// internal/workflow/validation.go

// ValidateInputMapping validates a value against a ParameterSchema
func ValidateInputMapping(schema *workflow.ParameterSchema, value any) error {
    if schema == nil {
        return nil // no schema = no validation
    }

    // Required field check
    if schema.Required && value == nil {
        return fmt.Errorf("required parameter %q is nil", schema.Name)
    }

    // Nil values that are not required pass validation
    if value == nil {
        return nil
    }

    // Type checking
    if schema.Type != "" {
        if err := validateType(schema.Type, value); err != nil {
            return fmt.Errorf("parameter %q: %w", schema.Name, err)
        }
    }

    return nil
}

func validateType(expected string, value any) error {
    // Handle array types
    if strings.HasPrefix(expected, "[]") {
        rv := reflect.ValueOf(value)
        if rv.Kind() != reflect.Slice && rv.Kind() != reflect.Array {
            return fmt.Errorf("expected %s, got %T", expected, value)
        }
        return nil
    }

    switch expected {
    case "string":
        if _, ok := value.(string); !ok {
            return fmt.Errorf("expected string, got %T", value)
        }
    case "int":
        if _, ok := toInt(value); !ok {
            return fmt.Errorf("expected int, got %T", value)
        }
    case "float64":
        if _, ok := toFloat64(value); !ok {
            return fmt.Errorf("expected float64, got %T", value)
        }
    case "bool":
        if _, ok := value.(bool); !ok {
            return fmt.Errorf("expected bool, got %T", value)
        }
    case "map":
        if _, ok := value.(map[string]any); !ok {
            return fmt.Errorf("expected map, got %T", value)
        }
    case "any":
        // Any type is always valid
    default:
        // Unknown type — skip validation
    }
    return nil
}
```

#### 1.5.2 Integration

Replace the stub in `workflow.go`:

```go
func (w *Workflow) validateInputMapping(schema *workflow.ParameterSchema, value any) bool {
    if err := ValidateInputMapping(schema, value); err != nil {
        return false
    }
    return true
}
```

Optionally, change the signature to return `error` for better error messages in the logging that already exists around the call sites.

### Alternatives Considered

1. **JSON Schema validation**: More standard but heavier dependency. The existing `ParameterSchema` type is simpler and sufficient for the current type system.

2. **Compile-time validation only** (validate schema at registration, not at runtime): Wouldn't catch data flow issues where a node produces unexpected output types.

### Migration Plan

- No schema changes — uses existing `ParameterSchema` type
- Existing workflows that pass invalid data will start getting validation errors — this is the desired behavior, but may surface latent bugs
- Validation failures should be warnings initially, with a configuration flag to make them hard errors

### Open Questions

1. Should validation failures be hard errors (stop execution) or soft warnings (log and continue)?
2. Should there be a "strict mode" vs "permissive mode" configuration?
3. Should validation support custom validation rules via the existing `Validations []string` field in `ParameterSchema`?
