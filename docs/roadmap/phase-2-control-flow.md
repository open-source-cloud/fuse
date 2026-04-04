# Phase 2: Control Flow — Make It Powerful

> **Goal:** Add advanced control flow patterns that enable complex, real-world automation scenarios. Depends on Phase 1 (durable execution for sleep/wait, retries for sub-workflows).

---

## 2.1 Wait / Sleep / Delayed Execution

### Motivation

The `StateSleeping` state already exists in `internal/workflow/workflow.go:36` but is **never used** anywhere in the codebase. There is no mechanism for a workflow to pause execution for a duration or until an external event occurs.

Use cases that require this:

- **Rate limiting**: Wait 1 second between API calls to respect rate limits
- **Polling**: Check external status every 30 seconds until complete
- **Scheduled actions**: Send a reminder email 24 hours after signup
- **Human-in-the-loop**: Wait for a manager's approval before proceeding
- **External orchestration**: Pause until a payment processor confirms a charge

The current async function pattern (`internal/actors/workflow_handler.go:202-252`) is a partial solution — it pauses execution until an external callback arrives — but it lacks timeout support, has no persistence across restarts, and requires the external system to know the exact callback URL format.

### Prior Art

**Inngest** provides two primitives:

- `step.sleep("wait-period", "1h")` — Durable sleep that survives restarts. The function execution is paused and resumed by the platform after the duration.
- `step.waitForEvent("wait-approval", { event: "approval/received", timeout: "24h", match: "data.userId" })` — Waits for a matching event with timeout. If the event arrives, execution continues with the event data. If timeout fires, the step fails.

**Restate** provides:

- `ctx.sleep(duration)` — Journaled sleep. Timer is persisted; if the process restarts, the sleep resumes from the correct time.
- Awakeables (`ctx.awakeable()`) — Returns a durable promise + external handle. External systems resolve the awakeable by ID, which unblocks the waiting handler. Supports timeout.

**n8n** provides:

- **Wait node** — Pauses workflow execution for a specified time, until a specific date, or until a webhook is received.

**What Fuse should adopt:** Two distinct action types — `SleepAction` for durable timers and `WaitForEventAction` (Awakeable) for external signal waiting. Both should integrate with the journal (Phase 1.1) for durability.

### Design

#### 2.1.1 New Action Types

```go
// internal/workflow/workflowactions/action.go

const (
    ActionSleep        ActionType = "workflow:sleep"
    ActionWaitForEvent ActionType = "workflow:wait-for-event"
)

// SleepAction pauses workflow execution for a duration
type SleepAction struct {
    ThreadID uint16
    ExecID   workflow.ExecID
    Duration time.Duration
    Reason   string // human-readable reason for the sleep
}

func (a *SleepAction) Type() ActionType { return ActionSleep }

// WaitForEventAction pauses workflow execution until an external event arrives
type WaitForEventAction struct {
    ThreadID    uint16
    ExecID      workflow.ExecID
    AwakeableID string        // unique ID for external systems to resolve
    Timeout     time.Duration // max wait time (0 = no timeout)
    Filter      string        // optional: expression to match incoming events
}

func (a *WaitForEventAction) Type() ActionType { return ActionWaitForEvent }
```

#### 2.1.2 Sleep Function (Internal Package)

Create a built-in `sleep` function in the internal packages:

```go
// internal/packages/internal/sleep/sleep.go

package sleep

import (
    "github.com/open-source-cloud/fuse/pkg/workflow"
)

const (
    PackageID  = "system"
    FunctionID = "sleep"
)

// Metadata describes the sleep function's interface
func Metadata() workflow.FunctionMetadata {
    return workflow.FunctionMetadata{
        Transport: transport.Internal,
        Input: workflow.InputMetadata{
            Parameters: []workflow.ParameterSchema{
                {Name: "duration", Type: "string", Required: true, Description: "Duration to sleep (e.g., '5s', '1h30m')"},
                {Name: "reason", Type: "string", Required: false, Description: "Human-readable reason for the delay"},
            },
        },
        Output: workflow.OutputMetadata{
            Parameters: []workflow.ParameterSchema{
                {Name: "sleptFor", Type: "string", Description: "Actual duration slept"},
            },
        },
    }
}
```

The `sleep` function doesn't execute like a normal function. When the `WorkflowFunc` worker encounters a `system/sleep` function, it returns a special result indicating a sleep is needed. The `WorkflowHandler` then:

1. Sets the workflow state to `StateSleeping`
2. Journals the sleep entry with deadline
3. Schedules a wake-up timer via `SendAfter`
4. On wake-up, sets state back to `StateRunning`, records sleep completion, calls `Next()`

#### 2.1.3 Awakeable / Wait-for-Event

```go
// internal/workflow/awakeable.go

// Awakeable represents a durable promise that can be resolved externally
type Awakeable struct {
    ID         string        `json:"id" bson:"id"`
    WorkflowID workflow.ID   `json:"workflowId" bson:"workflowId"`
    ExecID     workflow.ExecID `json:"execId" bson:"execId"`
    ThreadID   uint16        `json:"threadId" bson:"threadId"`
    CreatedAt  time.Time     `json:"createdAt" bson:"createdAt"`
    Timeout    time.Duration `json:"timeout" bson:"timeout"`
    DeadlineAt time.Time     `json:"deadlineAt" bson:"deadlineAt"`
    Status     AwakeableStatus `json:"status" bson:"status"`
    Result     map[string]any `json:"result,omitempty" bson:"result,omitempty"`
}

type AwakeableStatus string

const (
    AwakeablePending  AwakeableStatus = "pending"
    AwakeableResolved AwakeableStatus = "resolved"
    AwakeableTimedOut AwakeableStatus = "timed_out"
    AwakeableCancelled AwakeableStatus = "cancelled"
)
```

#### 2.1.4 Awakeable Repository

```go
// internal/repositories/awakeable.go

type AwakeableRepository interface {
    Save(awakeable *workflow.Awakeable) error
    FindByID(id string) (*workflow.Awakeable, error)
    FindPending(workflowID string) ([]*workflow.Awakeable, error)
    Resolve(id string, result map[string]any) error
}
```

#### 2.1.5 Awakeable HTTP API

```
POST /v1/awakeables/{awakeableID}/resolve
{
    "data": { ... }
}

Response 200:
{
    "workflowId": "...",
    "awakeableId": "...",
    "status": "resolved"
}
```

This endpoint is similar to the existing async function result endpoint (`POST /v1/workflows/{workflowID}/execs/{execID}`) but decoupled from the internal exec ID structure.

#### 2.1.6 New Message Types

```go
// internal/messaging/sleep.go

const SleepWakeUp MessageType = "workflow:sleep:wakeup"

type SleepWakeUpMessage struct {
    WorkflowID workflow.ID
    ExecID     workflow.ExecID
    ThreadID   uint16
}

// internal/messaging/awakeable.go

const AwakeableResolved MessageType = "workflow:awakeable:resolved"

type AwakeableResolvedMessage struct {
    WorkflowID  workflow.ID
    AwakeableID string
    ExecID      workflow.ExecID
    ThreadID    uint16
    Data        map[string]any
}
```

#### 2.1.7 WorkflowHandler Sleep Handling

```go
func (a *WorkflowHandler) handleWorkflowAction(action workflowactions.Action) {
    switch action.Type() {
    case workflowactions.ActionRunFunction:
        a.handleWorkflowRunFunctionAction(action)
    case workflowactions.ActionRunParallelFunctions:
        for _, runFuncAction := range action.(*workflowactions.RunParallelFunctionsAction).Actions {
            a.handleWorkflowRunFunctionAction(runFuncAction)
        }
    case workflowactions.ActionSleep:
        a.handleSleepAction(action.(*workflowactions.SleepAction))
    case workflowactions.ActionWaitForEvent:
        a.handleWaitForEventAction(action.(*workflowactions.WaitForEventAction))
    }
}

func (a *WorkflowHandler) handleSleepAction(action *workflowactions.SleepAction) {
    a.workflow.SetState(internalworkflow.StateSleeping)
    // Journal the sleep entry (for durable replay)
    a.workflow.Journal().Append(workflow.JournalEntry{
        Type:     workflow.JournalSleepStarted,
        ThreadID: action.ThreadID,
        ExecID:   action.ExecID.String(),
        Data:     map[string]any{"duration": action.Duration.String()},
    })
    // Schedule wake-up
    msg := messaging.NewSleepWakeUpMessage(a.workflow.ID(), action.ExecID, action.ThreadID)
    a.SendAfter(a.PID(), msg, action.Duration)
}
```

#### 2.1.8 Durability (Journal Integration)

On restart/resume (Phase 1.1), sleep entries in the journal are replayed:

- If the sleep deadline is in the past → immediately continue (fire wake-up)
- If the sleep deadline is in the future → schedule a new timer for the remaining duration

```go
// During Resume() journal replay:
case JournalSleepStarted:
    deadline := entry.Timestamp.Add(parseDuration(entry.Data["duration"]))
    remaining := time.Until(deadline)
    if remaining <= 0 {
        // Sleep already elapsed — continue immediately
        immediateWakeUps = append(immediateWakeUps, entry)
    } else {
        // Re-schedule the timer
        a.SendAfter(a.PID(), wakeUpMsg, remaining)
    }
```

### Alternatives Considered

1. **Implement sleep as a regular async function**: The function would start a goroutine with `time.Sleep`, then call the async callback. This works but isn't durable — if the process restarts, the timer is lost.
2. **External scheduler (cron, Redis delayed queues)**: Adds infrastructure dependency. Ergo's `SendAfter` combined with journaling provides the same functionality without external systems.
3. **Combined sleep + wait-for-event into a single primitive**: Conceptually simpler but less flexible. Sleep and wait-for-event have different use cases and different timeout semantics.

### Migration Plan

- New action types added to `workflowactions` — existing actions unaffected
- `StateSleeping` already exists — this feature gives it meaning
- New `system/sleep` and `system/wait` internal packages registered via `InternalPackages`
- New `/v1/awakeables/` endpoint added to router — no impact on existing endpoints
- Awakeable repository added alongside existing repositories

### Open Questions

1. Should `WaitForEvent` support event matching/filtering (like Inngest's `match` parameter), or should filtering be the responsibility of the triggering system?
2. Should there be a maximum sleep duration to prevent workflows from sleeping for years?
3. Should awakeables be reusable (resolve multiple times) or single-use?
4. How should sleep interact with workflow-level timeout (Phase 1.3)? Should sleep time count against the total workflow timeout?

---

## 2.2 Workflow Cancellation

### Motivation

There is currently **no way to cancel a running workflow**. Once triggered, a workflow runs until completion or error. There is no cancellation API, no `StateCancelled` state, and no mechanism to interrupt in-flight function executions.

Use cases:

- User triggers a workflow by mistake and wants to stop it
- A long-running workflow becomes irrelevant (e.g., an order is cancelled)
- Operational need to stop a misbehaving workflow
- Cascading cancellation when a parent workflow is cancelled (for sub-workflows, Phase 2.3)

### Prior Art

**Inngest** supports cancellation via events or API. A function can specify cancellation events: when a matching event arrives, the function is automatically cancelled. There's also an API to cancel individual function runs.

**Restate** supports cancellation through the invocation API. Cancelling an invocation terminates the handler and cleans up resources. Child invocations can be configured to cancel automatically.

**n8n** allows stopping running executions from the UI or API, which terminates the workflow at whatever node is currently executing.

**What Fuse should adopt:** Cancellation via HTTP API, with a `StateCancelled` terminal state. Cancellation should interrupt sleeping/waiting workflows immediately and prevent new function executions from starting.

### Design

#### 2.2.1 New State

```go
// internal/workflow/workflow.go

const (
    // StateCancelled Workflow cancelled state (terminated by user/system)
    StateCancelled State = "cancelled"
)
```

#### 2.2.2 Cancellation Message

```go
// internal/messaging/cancel_workflow.go

const CancelWorkflow MessageType = "workflow:cancel"

type CancelWorkflowMessage struct {
    WorkflowID workflow.ID
    Reason     string
}

func NewCancelWorkflowMessage(workflowID workflow.ID, reason string) Message {
    return Message{
        Type: CancelWorkflow,
        Args: CancelWorkflowMessage{WorkflowID: workflowID, Reason: reason},
    }
}
```

#### 2.2.3 HTTP API

```
POST /v1/workflows/{workflowID}/cancel
{
    "reason": "User requested cancellation"  // optional
}

Response 200:
{
    "workflowId": "...",
    "status": "cancelled",
    "cancelledAt": "2025-07-28T10:00:00Z"
}

Response 404: Workflow not found
Response 409: Workflow already in terminal state (finished/error/cancelled)
```

#### 2.2.4 Cancel Handler

```go
// internal/handlers/cancel_workflow.go

const (
    CancelWorkflowHandlerName     = "cancel_workflow_handler"
    CancelWorkflowHandlerPoolName = "cancel_workflow_handler_pool"
)

type CancelWorkflowHandlerFactory HandlerFactory[*CancelWorkflowHandler]

type CancelWorkflowHandler struct {
    Handler
}

func NewCancelWorkflowHandlerFactory() *CancelWorkflowHandlerFactory {
    return &CancelWorkflowHandlerFactory{
        Factory: func() gen.ProcessBehavior {
            return &CancelWorkflowHandler{}
        },
    }
}

func (h *CancelWorkflowHandler) HandlePost(from gen.PID, w http.ResponseWriter, r *http.Request) error {
    workflowID, err := h.GetPathParam(r, "workflowID")
    if err != nil {
        return h.SendBadRequest(w, err, EmptyFields)
    }

    var body struct {
        Reason string `json:"reason"`
    }
    _ = h.BindJSON(w, r, &body) // reason is optional

    // Send cancel message to WorkflowSupervisor which routes to the correct handler
    cancelMsg := messaging.NewCancelWorkflowMessage(workflow.ID(workflowID), body.Reason)
    if err := h.Send(gen.Atom(WorkflowSupervisorName), cancelMsg); err != nil {
        return h.SendNotFound(w, "workflow not found or not running", EmptyFields)
    }

    return h.SendJSON(w, http.StatusOK, map[string]any{
        "workflowId":  workflowID,
        "status":      "cancelled",
        "cancelledAt": time.Now().Format(time.RFC3339),
    })
}
```

#### 2.2.5 WorkflowSupervisor Routing

The `WorkflowSupervisor` (`internal/actors/workflow_sup.go`) needs to route cancel messages to the correct `WorkflowHandler`:

```go
func (s *WorkflowSupervisor) HandleMessage(from gen.PID, message any) error {
    msg, ok := message.(messaging.Message)
    if !ok { return nil }

    switch msg.Type {
    case messaging.TriggerWorkflow:
        // ... existing trigger logic
    case messaging.CancelWorkflow:
        cancelMsg, _ := msg.CancelWorkflowMessage()
        pid, exists := s.workflowActors[cancelMsg.WorkflowID]
        if !exists {
            s.Log().Warning("cancel requested for unknown workflow %s", cancelMsg.WorkflowID)
            return nil
        }
        return s.Send(pid, message)
    }
    return nil
}
```

#### 2.2.6 WorkflowHandler Cancellation

```go
// In HandleMessage:
case messaging.CancelWorkflow:
    return a.handleMsgCancelWorkflow(msg)

func (a *WorkflowHandler) handleMsgCancelWorkflow(msg messaging.Message) error {
    cancelMsg, ok := msg.Args.(messaging.CancelWorkflowMessage)
    if !ok {
        return nil
    }

    a.workflow.RunExclusive(func() {
        currentState := a.workflow.State()
        // Only cancel if in a non-terminal state
        if currentState == internalworkflow.StateFinished ||
            currentState == internalworkflow.StateError ||
            currentState == internalworkflow.StateCancelled {
            a.Log().Warning("cannot cancel workflow %s in state %s", cancelMsg.WorkflowID, currentState)
            return
        }

        a.workflow.SetState(internalworkflow.StateCancelled)
        a.workflow.Journal().Append(workflow.JournalEntry{
            Type:  workflow.JournalStateChanged,
            State: internalworkflow.StateCancelled,
            Data:  map[string]any{"reason": cancelMsg.Reason},
        })

        // Cancel any pending execution timers
        a.executionTimer.CancelAll()

        // Persist final state
        a.persistFinalState()

        if a.workflowDebugTrace() {
            a.workflow.AppendDebugTracef("state=cancelled reason=%s", cancelMsg.Reason)
        }
    })

    // Notify supervisor for cleanup (same as graceful completion)
    supName := WorkflowInstanceSupervisorName(a.workflow.ID())
    return a.Send(gen.Atom(supName), messaging.NewWorkflowCompletedMessage(
        a.workflow.ID(), internalworkflow.StateCancelled,
    ))
}
```

#### 2.2.7 Guard Against Post-Cancellation Messages

After cancellation, the handler may still receive in-flight function results. Guard against this:

```go
func (a *WorkflowHandler) handleMsgFunctionResult(msg messaging.Message) error {
    // ... existing parsing ...

    a.workflow.RunExclusive(func() {
        // Guard: ignore results after cancellation
        if a.workflow.State() == internalworkflow.StateCancelled {
            a.Log().Warning("ignoring function result for cancelled workflow %s", a.workflow.ID())
            return
        }
        // ... rest of existing logic
    })
    return nil
}
```

### Alternatives Considered

1. **Kill actor immediately** (terminate actor without cleanup): Fast but loses state. Better to transition through a clean `StateCancelled` state that persists the reason.
2. **Cancel via event system** (Inngest-style event-triggered cancellation): More flexible but requires the event system (Phase 3.1) to be built first. HTTP API is the simplest initial approach.
3. **Soft cancel (just prevent new steps from starting)**: Simpler but doesn't interrupt sleeping/waiting workflows. Need the ability to wake up sleeping workflows immediately for responsive cancellation.

### Migration Plan

- New `StateCancelled` constant added — no impact on existing states
- New HTTP endpoint added to router — no impact on existing endpoints
- New message type added — WorkflowHandler/Supervisor gain a new case in their switch
- Existing workflows in `StateRunning` can be cancelled immediately after this feature ships

### Open Questions

1. Should cancellation trigger error edges (compensation logic), or bypass them entirely?
2. Should there be a "force cancel" that terminates the actor immediately vs. "graceful cancel" that waits for in-flight functions?
3. Should cancelled workflows be resumable (re-trigger from where they were cancelled)?

---

## 2.3 Sub-workflows

### Motivation

The project vision ([#1](https://github.com/open-source-cloud/fuse/issues/1)) explicitly calls for "multi-level, nodes can also be other workflows (Reusable workflows)." This is not yet implemented.

Sub-workflows enable:

- **Reusability**: Common patterns (e.g., "send notification + wait for acknowledgement") defined once, used in many workflows
- **Modularity**: Complex workflows decomposed into manageable sub-graphs
- **Isolation**: Sub-workflow failures contained without affecting the parent
- **Dynamic composition**: Choose which sub-workflow to run based on runtime data

### Prior Art

**n8n** provides an "Execute Sub-workflow" node that:

- Triggers another workflow by ID
- Passes input data to the sub-workflow's trigger node
- Waits for the sub-workflow to complete
- Receives the sub-workflow's output as the node's output
- Supports async execution (fire-and-forget) mode

**Inngest** supports `step.invoke()` which calls another Inngest function and waits for its result. The invoked function runs independently with its own retries and steps, but the caller blocks until completion.

**Restate** uses standard service calls — workflows invoke other workflows/services like normal function calls. The calling workflow is durably suspended until the callee returns.

**What Fuse should adopt:** n8n-style sub-workflow execution via a built-in `system/subworkflow` function. The parent workflow enters `StateSleeping` on the thread that called the sub-workflow, while the sub-workflow runs independently. On completion, the sub-workflow's output is propagated back to the parent.

### Design

#### 2.3.1 Sub-workflow Function (Internal Package)

```go
// internal/packages/internal/subworkflow/subworkflow.go

const (
    PackageID  = "system"
    FunctionID = "subworkflow"
)

func Metadata() workflow.FunctionMetadata {
    return workflow.FunctionMetadata{
        Transport: transport.Internal,
        Input: workflow.InputMetadata{
            Parameters: []workflow.ParameterSchema{
                {Name: "schemaId", Type: "string", Required: true, Description: "Schema ID of the workflow to execute"},
                {Name: "input", Type: "map", Required: false, Description: "Input data to pass to the sub-workflow trigger"},
                {Name: "async", Type: "bool", Required: false, Default: false, Description: "If true, don't wait for completion"},
            },
        },
        Output: workflow.OutputMetadata{
            Parameters: []workflow.ParameterSchema{
                {Name: "workflowId", Type: "string", Description: "ID of the spawned sub-workflow"},
                {Name: "status", Type: "string", Description: "Final status of the sub-workflow"},
                {Name: "output", Type: "map", Description: "Output data from the sub-workflow"},
            },
        },
    }
}
```

#### 2.3.2 New Action Type

```go
// internal/workflow/workflowactions/action.go

const ActionRunSubWorkflow ActionType = "workflow:subworkflow:run"

type RunSubWorkflowAction struct {
    ParentWorkflowID workflow.ID
    ParentThreadID   uint16
    ParentExecID     workflow.ExecID
    SchemaID         string
    Input            map[string]any
    Async            bool
}

func (a *RunSubWorkflowAction) Type() ActionType { return ActionRunSubWorkflow }
```

#### 2.3.3 Parent-Child Relationship Tracking

```go
// internal/workflow/subworkflow.go

// SubWorkflowRef tracks a parent-child workflow relationship
type SubWorkflowRef struct {
    ParentWorkflowID workflow.ID    `json:"parentWorkflowId" bson:"parentWorkflowId"`
    ParentThreadID   uint16         `json:"parentThreadId" bson:"parentThreadId"`
    ParentExecID     workflow.ExecID `json:"parentExecId" bson:"parentExecId"`
    ChildWorkflowID  workflow.ID    `json:"childWorkflowId" bson:"childWorkflowId"`
    ChildSchemaID    string         `json:"childSchemaId" bson:"childSchemaId"`
    Async            bool           `json:"async" bson:"async"`
}
```

Add to WorkflowRepository:

```go
type WorkflowRepository interface {
    Exists(id string) bool
    Get(id string) (*workflow.Workflow, error)
    Save(workflow *workflow.Workflow) error
    SaveSubWorkflowRef(ref *workflow.SubWorkflowRef) error           // NEW
    FindSubWorkflowRef(childID string) (*workflow.SubWorkflowRef, error) // NEW
}
```

#### 2.3.4 Execution Flow

```
Parent WorkflowHandler
  |
  v  (encounters system/subworkflow node)
WorkflowFunc detects subworkflow function
  |
  v  (returns SubWorkflowAction)
Parent WorkflowHandler.handleSubWorkflowAction()
  |
  v
  1. Parent thread enters StateSleeping
  2. Save SubWorkflowRef (parent -> child mapping)
  3. Send TriggerWorkflow to WorkflowSupervisor (spawns child)
  4. Journal: JournalSubWorkflowStarted
  |
  ... child executes independently ...
  |
  v  (child completes)
Child WorkflowHandler completes
  |
  v
WorkflowInstanceSupervisor.HandleMessage(WorkflowCompleted)
  |
  v
  1. Check SubWorkflowRef — is this a child workflow?
  2. If yes, send SubWorkflowCompleted message to parent's WorkflowHandler
  |
  v
Parent WorkflowHandler.handleMsgSubWorkflowCompleted()
  |
  v
  1. Set result on parent's audit log (child output = parent node output)
  2. Parent thread resumes: Next(parentThreadID)
```

#### 2.3.5 Sub-Workflow Completion Message

```go
// internal/messaging/subworkflow.go

const SubWorkflowCompleted MessageType = "workflow:subworkflow:completed"

type SubWorkflowCompletedMessage struct {
    ParentWorkflowID workflow.ID
    ParentThreadID   uint16
    ParentExecID     workflow.ExecID
    ChildWorkflowID  workflow.ID
    ChildStatus      workflow.FunctionOutputStatus
    ChildOutput      map[string]any
}
```

#### 2.3.6 Cancellation Cascade

When a parent workflow is cancelled (Phase 2.2), also cancel active child workflows:

```go
// In handleMsgCancelWorkflow, after setting StateCancelled:
activeChildren := a.workflowRepository.FindActiveSubWorkflows(a.workflow.ID().String())
for _, child := range activeChildren {
    cancelMsg := messaging.NewCancelWorkflowMessage(child.ChildWorkflowID, "parent cancelled")
    a.Send(gen.Atom(WorkflowSupervisorName), cancelMsg)
}
```

### Alternatives Considered

1. **Inline sub-graph expansion** (copy sub-workflow nodes into parent graph): Simpler execution model but loses isolation. A failure in the sub-graph directly affects the parent. Also makes the graph unwieldy for complex compositions.
2. **Event-based decoupling** (parent emits event, sub-workflow triggered by event, result emitted as event): More loosely coupled but harder to implement synchronous waiting. Better suited for fire-and-forget patterns (which we support via the `async` parameter).
3. **Dedicated SubWorkflowSupervisor**: Unnecessary complexity. The existing `WorkflowSupervisor` can manage both parent and child workflows; the only addition is the parent-child relationship tracking.

### Migration Plan

- New `system/subworkflow` function added to internal packages
- New action type, message type, and repository method added
- No changes to existing graph schemas — sub-workflows are used by referencing `system/subworkflow` as a node's function
- Sub-workflow references stored separately from workflow data

### Open Questions

1. Should sub-workflows inherit the parent's store/private store, or start with a clean slate?
2. What is the maximum nesting depth for sub-workflows to prevent infinite recursion?
3. Should async (fire-and-forget) sub-workflows have their own timeout, or rely on the child's workflow-level timeout?
4. Should sub-workflow input mapping use the same `InputMapping` schema as edges, or a simplified key-value format?

---

## 2.4 Enhanced Conditionals / Switch

### Motivation

The current conditional branching (`internal/workflow/workflow.go:154-176`) only supports **exact value equality matching**:

```go
for _, edge := range currentNode.OutputEdges() {
    edgeCondition := edge.Condition()
    if edgeCondition.Value == conditionalValue {
        // ...
        edges = append(edges, edge)
    }
}
```

This limits branching to simple scenarios. Real-world workflows need:

- Range checks: `amount > 1000`
- Pattern matching: `email matches "*@company.com"`
- Multiple conditions: `status == "active" && tier == "premium"`
- Default/fallback: "if none of the above match, go here"
- Type-based routing: different paths for different data shapes

The project already depends on `expr-lang/expr` (noted in the codebase's tech stack), which is a powerful expression evaluation library — but it's not used for conditional branching.

### Prior Art

**n8n** provides:

- **IF node**: Two outputs (true/false) with expression-based conditions supporting comparison, string, number, boolean, and date operators
- **Switch node**: Multiple named outputs, each with its own condition rule. Supports fallback output when no conditions match.

**Inngest** doesn't have built-in conditional routing (functions handle branching in code), but its expression engine supports complex matching on event data.

**What Fuse should adopt:** Expression-based conditions using `expr-lang/expr`, with support for multiple named outputs (switch pattern) and a default/fallback edge.

### Design

#### 2.4.1 Enhanced Edge Condition Schema

```go
// internal/workflow/edge_schema.go

// EdgeConditionType classifies how the condition is evaluated
type EdgeConditionType string

const (
    // ConditionExact matches edge value exactly (current behavior)
    ConditionExact EdgeConditionType = "exact"
    // ConditionExpression evaluates an expr-lang expression
    ConditionExpression EdgeConditionType = "expression"
    // ConditionDefault matches when no other condition on the same node matches
    ConditionDefault EdgeConditionType = "default"
)

// EdgeCondition defines when an edge should be followed
type EdgeCondition struct {
    Name       string            `json:"name" bson:"name"`
    Type       EdgeConditionType `json:"type,omitempty" bson:"type,omitempty"` // NEW: defaults to "exact"
    Value      any               `json:"value,omitempty" bson:"value,omitempty"`
    Expression string            `json:"expression,omitempty" bson:"expression,omitempty"` // NEW: expr-lang expression
}
```

#### 2.4.2 Expression Evaluation

```go
// internal/workflow/expression.go

import "github.com/expr-lang/expr"

// EvaluateCondition evaluates an edge condition against the current workflow state
func (w *Workflow) EvaluateCondition(condition *EdgeCondition, currentNode *Node) (bool, error) {
    switch condition.Type {
    case ConditionDefault, "":
        if condition.Type == "" {
            // Legacy: exact match (backwards compatible)
            return w.evaluateExactCondition(condition, currentNode), nil
        }
        return true, nil // Default always matches (used as fallback)

    case ConditionExact:
        return w.evaluateExactCondition(condition, currentNode), nil

    case ConditionExpression:
        return w.evaluateExpression(condition.Expression, currentNode)
    }
    return false, fmt.Errorf("unknown condition type: %s", condition.Type)
}

func (w *Workflow) evaluateExactCondition(condition *EdgeCondition, currentNode *Node) bool {
    conditionalSource := currentNode.FunctionMetadata().Output.ConditionalOutputField
    conditionalValue := w.aggregatedOutput.Get(fmt.Sprintf("%s.%s", currentNode.ID(), conditionalSource))
    return condition.Value == conditionalValue
}

func (w *Workflow) evaluateExpression(expression string, currentNode *Node) (bool, error) {
    // Build environment with node outputs and workflow store
    env := make(map[string]any)

    // Add current node's output
    for key, value := range w.aggregatedOutput.Raw() {
        env[key] = value
    }

    // Add node-scoped shorthand: output.fieldName
    nodePrefix := currentNode.ID() + "."
    nodeOutput := make(map[string]any)
    for key, value := range w.aggregatedOutput.Raw() {
        if strings.HasPrefix(key, nodePrefix) {
            shortKey := strings.TrimPrefix(key, nodePrefix)
            nodeOutput[shortKey] = value
        }
    }
    env["output"] = nodeOutput

    // Compile and evaluate
    program, err := expr.Compile(expression, expr.Env(env), expr.AsBool())
    if err != nil {
        return false, fmt.Errorf("failed to compile expression %q: %w", expression, err)
    }

    result, err := expr.Run(program, env)
    if err != nil {
        return false, fmt.Errorf("failed to evaluate expression %q: %w", expression, err)
    }

    boolResult, ok := result.(bool)
    if !ok {
        return false, fmt.Errorf("expression %q did not return bool, got %T", expression, result)
    }
    return boolResult, nil
}
```

#### 2.4.3 Updated Edge Filtering

Replace `filterOutputEdgesByConditionals` in `workflow.go`:

```go
func (w *Workflow) filterOutputEdgesByConditionals(currentNode *Node) []*Edge {
    if !currentNode.IsConditional() {
        return currentNode.OutputEdges()
    }

    var matchedEdges []*Edge
    var defaultEdge *Edge

    for _, edge := range currentNode.OutputEdges() {
        condition := edge.Condition()
        if condition == nil {
            matchedEdges = append(matchedEdges, edge)
            continue
        }

        if condition.Type == ConditionDefault {
            defaultEdge = edge
            continue
        }

        matches, err := w.EvaluateCondition(condition, currentNode)
        if err != nil {
            log.Error().Err(err).Str("edge", edge.ID()).Msg("condition evaluation failed")
            continue
        }
        if matches {
            matchedEdges = append(matchedEdges, edge)
        }
    }

    // If no conditions matched and there's a default edge, use it
    if len(matchedEdges) == 0 && defaultEdge != nil {
        matchedEdges = append(matchedEdges, defaultEdge)
    }

    return matchedEdges
}
```

#### 2.4.4 Expression Security

Expressions are user-provided and must be sandboxed:

```go
// Compile with safety constraints
program, err := expr.Compile(expression,
    expr.Env(env),
    expr.AsBool(),
    expr.DisableAllBuiltins(),  // Disable potentially dangerous builtins
    // Whitelist safe functions:
    expr.Function("len", ..., new(func([]any) int)),
    expr.Function("contains", ..., new(func(string, string) bool)),
    expr.Function("startsWith", ..., new(func(string, string) bool)),
    expr.Function("endsWith", ..., new(func(string, string) bool)),
    expr.Function("lower", ..., new(func(string) string)),
    expr.Function("upper", ..., new(func(string) string)),
)
```

### Alternatives Considered

1. **JavaScript/Lua embedded expressions**: More powerful but heavier dependency and larger attack surface. `expr-lang/expr` is Go-native, fast, and designed for exactly this use case.
2. **DSL for conditions**: Building a custom condition DSL is redundant when `expr-lang/expr` already exists and is a Go standard for expression evaluation.
3. **Keep exact matching only, with more comparison operators**: Would require building a custom comparison engine. Using `expr` is more flexible and handles complex conditions naturally.

### Migration Plan

- `EdgeCondition.Type` defaults to empty string — existing conditions work unchanged via the legacy exact-match path
- New `Expression` field is optional — only used when `Type` is `"expression"`
- `ConditionDefault` is new — existing schemas don't have default edges, which is fine (no match = no edges, same as current behavior)

### Open Questions

1. Should expression compilation be cached per edge (for performance in loops)?
2. What functions should be whitelisted in the expression sandbox?
3. Should expressions have access to the workflow store, or only node outputs?
4. Should there be a maximum expression complexity / length to prevent abuse?

---

## 2.5 Merge Strategies at Join Nodes

### Motivation

When parallel branches converge at a join node (a node with multiple input edges from different threads), the current behavior collects all input mappings from all parent edges but has **no configurable strategy** for combining them. The `newRunFunctionAction` method in `workflow.go:233-258` simply iterates over input edges and collects mappings:

```go
if currentThread.ID() != node.thread {
    newOrCurrentThread = w.threads.New(node.thread, execID)
    for _, inputEdge := range node.InputEdges() {
        if inputEdge.To() == node {
            mappings = append(mappings, inputEdge.Input()...)
        }
    }
}
```

The `inputMapping` method handles array accumulation (`strings.HasPrefix(inputParamSchema.Type, "[]")` at line 325) but this is the only "merge strategy" — append to arrays. There's no way to:

- Merge maps from parallel branches
- Pick the first/last result
- Combine results into a structured object
- Apply a custom merge function

### Prior Art

**n8n** provides a **Merge node** with multiple strategies:

- **Append**: Combine all items from all inputs into one list
- **Combine by Position**: Pair items from each input by index
- **Combine by Field**: Join items from different inputs by a matching field (like SQL JOIN)
- **Multiplex**: Create all possible combinations (cross product)
- **Choose Branch**: Use only items from a specific input
- **SQL Query**: Custom SQL-like merge logic

**What Fuse should adopt:** A configurable `MergeStrategy` on join nodes, starting with the most common strategies: append (current behavior), merge-objects, first-wins, and last-wins.

### Design

#### 2.5.1 Merge Strategy Configuration

```go
// internal/workflow/merge.go

// MergeStrategyType defines how to combine outputs from parallel branches at a join node
type MergeStrategyType string

const (
    // MergeAppend appends all values into arrays (current default behavior)
    MergeAppend MergeStrategyType = "append"
    // MergeObject merges all outputs into a single map (later values override earlier)
    MergeObject MergeStrategyType = "merge"
    // MergeFirstWins uses the first branch's output that completes
    MergeFirstWins MergeStrategyType = "first"
    // MergeLastWins uses the last branch's output that completes
    MergeLastWins MergeStrategyType = "last"
    // MergeKeyed groups outputs by branch name into a keyed map
    MergeKeyed MergeStrategyType = "keyed"
)

// MergeConfig defines the merge strategy for a join node
type MergeConfig struct {
    Strategy MergeStrategyType `json:"strategy" bson:"strategy" validate:"oneof=append merge first last keyed"`
}

// DefaultMergeConfig returns the default merge strategy (append, matches current behavior)
func DefaultMergeConfig() MergeConfig {
    return MergeConfig{Strategy: MergeAppend}
}
```

#### 2.5.2 Node Schema Extension

```go
// Extend NodeSchema
type NodeSchema struct {
    ID         string         `json:"id" bson:"id"`
    FunctionID string         `json:"functionId" bson:"functionId"`
    Retry      *RetryPolicy   `json:"retry,omitempty" bson:"retry,omitempty"`
    Timeout    *TimeoutConfig `json:"timeout,omitempty" bson:"timeout,omitempty"`
    Merge      *MergeConfig   `json:"merge,omitempty" bson:"merge,omitempty"` // NEW
}
```

#### 2.5.3 Merge Strategy Implementations

```go
// internal/workflow/merge.go

// ApplyMergeStrategy combines inputs from multiple parent edges using the specified strategy
func ApplyMergeStrategy(config MergeConfig, inputs []BranchInput) map[string]any {
    switch config.Strategy {
    case MergeObject:
        return mergeObjects(inputs)
    case MergeFirstWins:
        return mergeFirstWins(inputs)
    case MergeLastWins:
        return mergeLastWins(inputs)
    case MergeKeyed:
        return mergeKeyed(inputs)
    default: // MergeAppend
        return mergeAppend(inputs)
    }
}

// BranchInput represents the output from one branch arriving at a join node
type BranchInput struct {
    EdgeID   string
    ThreadID uint16
    Data     map[string]any
}

func mergeAppend(inputs []BranchInput) map[string]any {
    result := make(map[string]any)
    for _, input := range inputs {
        for key, value := range input.Data {
            if existing, ok := result[key]; ok {
                // Append to existing slice
                switch v := existing.(type) {
                case []any:
                    result[key] = append(v, value)
                default:
                    result[key] = []any{v, value}
                }
            } else {
                result[key] = value
            }
        }
    }
    return result
}

func mergeObjects(inputs []BranchInput) map[string]any {
    result := make(map[string]any)
    for _, input := range inputs {
        for key, value := range input.Data {
            result[key] = value // later values override
        }
    }
    return result
}

func mergeFirstWins(inputs []BranchInput) map[string]any {
    if len(inputs) == 0 {
        return make(map[string]any)
    }
    return inputs[0].Data
}

func mergeLastWins(inputs []BranchInput) map[string]any {
    if len(inputs) == 0 {
        return make(map[string]any)
    }
    return inputs[len(inputs)-1].Data
}

func mergeKeyed(inputs []BranchInput) map[string]any {
    result := make(map[string]any)
    for _, input := range inputs {
        result[input.EdgeID] = input.Data
    }
    return result
}
```

#### 2.5.4 Integration with newRunFunctionAction

Modify the join-point logic in `workflow.go:233-258` to use the merge strategy:

```go
if currentThread.ID() != node.thread {
    newOrCurrentThread = w.threads.New(node.thread, execID)

    // Collect branch inputs
    var branchInputs []BranchInput
    for _, inputEdge := range node.InputEdges() {
        if inputEdge.To() == node {
            branchData := w.resolveMappings(inputEdge.Input())
            branchInputs = append(branchInputs, BranchInput{
                EdgeID:   inputEdge.ID(),
                ThreadID: inputEdge.From().Thread(),
                Data:     branchData,
            })
        }
    }

    // Apply merge strategy
    mergeConfig := DefaultMergeConfig()
    if node.schema.Merge != nil {
        mergeConfig = *node.schema.Merge
    }
    args = ApplyMergeStrategy(mergeConfig, branchInputs)
} else {
    currentThread.SetCurrentExecID(execID)
    args = w.inputMapping(edge, edge.Input())
}
```

### Alternatives Considered

1. **Custom merge function (user-provided code)**: Maximum flexibility but introduces code execution security concerns. Expression-based merging (using `expr`) could be a future extension.
2. **Merge as a separate node type**: Like n8n's Merge node. This would require inserting a synthetic merge node at every join point, complicating the graph. Configuring merge strategy on the join node itself is simpler.
3. **No merge configuration — always append**: Simplest approach and matches current behavior, but insufficient for workflows that need structured output from parallel branches.

### Migration Plan

- `MergeConfig` is optional on `NodeSchema` — nil defaults to `MergeAppend` (current behavior)
- The `inputMapping` function's existing array accumulation logic is superseded by the merge strategy, but only at join nodes (nodes with multiple parent threads). Single-thread edges continue to use `inputMapping` directly.

### Open Questions

1. Should merge strategies be extensible (pluggable merge functions)?
2. Should `MergeKeyed` use edge IDs or edge names as keys?
3. How should merge interact with conditional edges — should only the "taken" branches contribute to the merge?
4. Should there be a `MergeCustomExpression` strategy that uses `expr-lang/expr` for custom merge logic?

