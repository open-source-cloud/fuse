# Actor Model Patterns Skill

This skill provides expertise in the Actor Model, specifically using ergo.services in Go, including message passing, supervisor strategies, actor lifecycle, state management, and fault tolerance.

## Actor Model Fundamentals

### Core Concepts

- **Actor**: Independent unit of computation with isolated state
- **Message Passing**: Actors communicate only through messages
- **Isolation**: Actors don't share state
- **Supervision**: Parent actors supervise child actors
- **Fault Tolerance**: Actors can fail and be restarted

### Actor Characteristics

- Each actor has a unique address (PID)
- Actors process messages sequentially (one at a time)
- Actors maintain isolated state
- Actors communicate asynchronously via messages
- Actors can create other actors (supervision tree)

## Ergo.services Patterns

### Basic Actor Structure

```go
import (
    "ergo.services/ergo/act"
    "ergo.services/ergo/gen"
)

type WorkflowHandler struct {
    act.Actor  // Embed base actor type
    
    // Actor state
    workflow *workflow.Workflow
    config   *config.Config
}

func (a *WorkflowHandler) Init(args ...any) error {
    a.Log().Debug("starting workflow handler %s", a.PID())
    
    // Initialize actor state
    // Parse initialization arguments if needed
    
    return nil
}

func (a *WorkflowHandler) HandleMessage(from gen.PID, message any) error {
    switch msg := message.(type) {
    case *messaging.ExecuteWorkflow:
        return a.handleExecuteWorkflow(from, msg)
    case *messaging.StopWorkflow:
        return a.handleStopWorkflow(from, msg)
    default:
        a.Log().Warn("received unknown message type: %T", message)
        return fmt.Errorf("unknown message type: %T", message)
    }
}
```

### Factory Pattern

```go
type WorkflowHandlerFactory struct {
    Factory func() gen.ProcessBehavior
}

func NewWorkflowHandlerFactory(
    cfg *config.Config,
    service WorkflowService,
) *WorkflowHandlerFactory {
    return &WorkflowHandlerFactory{
        Factory: func() gen.ProcessBehavior {
            return &WorkflowHandler{
                config:  cfg,
                service: service,
            }
        },
    }
}
```

## Message Passing

### Message Types

```go
// Define typed messages
package messaging

type ExecuteWorkflow struct {
    WorkflowID string
    Input      map[string]any
    Timeout    time.Duration
}

type WorkflowCompleted struct {
    WorkflowID string
    Output     map[string]any
    Duration   time.Duration
}

type StopWorkflow struct {
    WorkflowID string
    Reason     string
}
```

### Sending Messages

```go
// Asynchronous send (fire-and-forget)
err := actor.Send(targetPID, &messaging.ExecuteWorkflow{
    WorkflowID: "wf-123",
    Input:      map[string]any{"key": "value"},
})

// Synchronous call (wait for response)
response, err := actor.Call(targetPID, &messaging.GetStatus{
    WorkflowID: "wf-123",
}, gen.DefaultCallTimeout)

if err != nil {
    a.Log().Error("call failed", "error", err)
    return err
}

// Type assert response
if status, ok := response.(*messaging.StatusResponse); ok {
    a.Log().Info("received status", "state", status.State)
}
```

### Message Handling

```go
func (a *WorkflowHandler) HandleMessage(from gen.PID, message any) error {
    a.Log().Debug("received message", "from", from, "type", fmt.Sprintf("%T", message))
    
    switch msg := message.(type) {
    case *messaging.ExecuteWorkflow:
        return a.handleExecuteWorkflow(from, msg)
        
    case *messaging.StopWorkflow:
        return a.handleStopWorkflow(from, msg)
        
    case *messaging.GetStatus:
        return a.handleGetStatus(from, msg)
        
    default:
        a.Log().Warn("unknown message type", "type", fmt.Sprintf("%T", message))
        return fmt.Errorf("unknown message type: %T", message)
    }
}

func (a *WorkflowHandler) handleExecuteWorkflow(from gen.PID, msg *messaging.ExecuteWorkflow) error {
    a.Log().Info("executing workflow", "workflowID", msg.WorkflowID, "from", from)
    
    // Process workflow execution
    result, err := a.service.ExecuteWorkflow(msg.WorkflowID, msg.Input)
    if err != nil {
        a.Log().Error("workflow execution failed", "error", err)
        return err
    }
    
    // Send response back
    a.Send(from, &messaging.WorkflowCompleted{
        WorkflowID: msg.WorkflowID,
        Output:     result,
    })
    
    return nil
}
```

## Supervisor Strategies

### Supervisor Types

```go
// One-for-One: Restart only failed child
spec := act.SupervisorSpec{
    Type: act.SupervisorTypeOneForOne,
    Restart: act.SupervisorRestartPermanent,
    Children: []act.SupervisorChildSpec{
        {
            Name:    "workflow_handler_1",
            Factory: handlerFactory.Factory,
        },
    },
}

// One-for-All: Restart all children if one fails
spec := act.SupervisorSpec{
    Type: act.SupervisorTypeOneForAll,
    Restart: act.SupervisorRestartTransient,
}

// Rest-for-One: Restart failed child and those started after it
spec := act.SupervisorSpec{
    Type: act.SupervisorTypeRestForOne,
    Restart: act.SupervisorRestartTemporary,
}
```

### Restart Policies

```go
// Permanent: Always restart
Restart: act.SupervisorRestartPermanent

// Temporary: Never restart
Restart: act.SupervisorRestartTemporary

// Transient: Restart only on abnormal termination
Restart: act.SupervisorRestartTransient
```

### Supervisor Implementation

```go
type WorkflowSupervisor struct {
    act.Supervisor
    
    config       *config.Config
    childFactory *WorkflowHandlerFactory
    workflows    map[string]gen.PID
}

func (a *WorkflowSupervisor) Init(_ ...any) (act.SupervisorSpec, error) {
    a.Log().Debug("starting workflow supervisor %s", a.PID())
    
    spec := act.SupervisorSpec{
        Type:    act.SupervisorTypeOneForOne,
        Restart: act.SupervisorRestartPermanent,
        Children: []act.SupervisorChildSpec{
            {
                Name:    "workflow_handler_1",
                Factory: a.childFactory.Factory,
                Args:    []any{},
            },
        },
    }
    
    return spec, nil
}
```

## Actor Lifecycle

### Lifecycle Hooks

```go
func (a *WorkflowHandler) Init(args ...any) error {
    // Called when actor is spawned
    a.Log().Debug("actor initialized", "pid", a.PID())
    
    // Initialize state
    // Load resources
    // Setup connections
    
    return nil
}

func (a *WorkflowHandler) HandleInfo(message any) error {
    // Handle system messages
    return a.HandleMessage(gen.PID{}, message)
}

func (a *WorkflowHandler) Terminate(reason error) {
    // Called when actor terminates
    a.Log().Info("actor terminating", "reason", reason)
    
    // Cleanup resources
    // Close connections
    // Save state
}
```

### Actor State Management

```go
type WorkflowHandler struct {
    act.Actor
    
    // Actor state (isolated, not shared)
    workflow     *workflow.Workflow
    currentNode  *workflow.Node
    context      map[string]any
    state        WorkflowState
}

func (a *WorkflowHandler) handleExecuteWorkflow(from gen.PID, msg *messaging.ExecuteWorkflow) error {
    // Update actor state
    a.workflow = loadWorkflow(msg.WorkflowID)
    a.state = WorkflowStateRunning
    a.context = msg.Input
    
    // Process workflow
    // State changes are isolated to this actor
}
```

## Fault Tolerance

### Let It Crash Philosophy

- Don't try to catch all errors
- Let supervisors handle failures
- Actors should fail fast on unexpected errors
- Supervisors decide restart strategy

```go
func (a *WorkflowHandler) handleExecuteWorkflow(from gen.PID, msg *messaging.ExecuteWorkflow) error {
    // Don't catch all errors - let supervisor handle
    result, err := a.service.ExecuteWorkflow(msg.WorkflowID, msg.Input)
    if err != nil {
        // Return error - supervisor will handle
        return fmt.Errorf("workflow execution failed: %w", err)
    }
    
    // Only handle expected errors
    if errors.Is(err, ErrWorkflowNotFound) {
        a.Send(from, &messaging.ErrorResponse{
            Error: "workflow not found",
        })
        return nil  // Don't crash on expected error
    }
    
    return err  // Unexpected error - let supervisor handle
}
```

### Error Handling

```go
// Expected errors: Handle gracefully
if errors.Is(err, ErrWorkflowNotFound) {
    a.Log().Warn("workflow not found", "workflowID", msg.WorkflowID)
    a.Send(from, &messaging.ErrorResponse{Error: "not found"})
    return nil
}

// Unexpected errors: Let supervisor handle
if err != nil {
    a.Log().Error("unexpected error", "error", err)
    return err  // Supervisor will restart actor
}
```

## Actor Pools and Workers

### Pool Pattern

```go
type WorkflowFuncPool struct {
    act.Pool
    
    packageRegistry packages.Registry
}

func (a *WorkflowFuncPool) Init(args ...any) (act.PoolSpec, error) {
    a.Log().Debug("starting workflow function pool %s", a.PID())
    
    spec := act.PoolSpec{
        Size:    10,  // Number of worker actors
        Factory: workflowFuncFactory.Factory,
    }
    
    return spec, nil
}

// Worker actor
type WorkflowFunc struct {
    act.Actor
    
    packageRegistry packages.Registry
}

func (a *WorkflowFunc) HandleMessage(from gen.PID, message any) error {
    switch msg := message.(type) {
    case *messaging.ExecuteFunction:
        return a.executeFunction(from, msg)
    default:
        return fmt.Errorf("unknown message: %T", message)
    }
}
```

### WebWorker Pattern (HTTP Handlers)

```go
type HealthCheckHandler struct {
    act.WebWorker
    
    service SomeService
}

func (h *HealthCheckHandler) HandleGet(from gen.PID, w http.ResponseWriter, r *http.Request) error {
    h.Log().Info("received health check request", "from", from)
    
    response := map[string]any{
        "status": "healthy",
        "time":   time.Now(),
    }
    
    return h.SendJSON(w, http.StatusOK, response)
}
```

## Best Practices

1. **Isolate state** - No shared mutable state between actors
2. **Use message passing** - All communication via messages
3. **Let supervisors handle failures** - Don't catch everything
4. **Keep Init lightweight** - Heavy initialization should be async
5. **Use actor's logger** - `a.Log()`, not global logger
6. **Define clear message types** - Avoid using `any` in messages
7. **Use typed init arguments** - Structs, not positional args
8. **Clean up in Terminate** - Release resources properly
9. **Don't block actor mailbox** - Delegate long operations to pools
10. **Supervisor strategy matters** - Choose appropriate restart policy

## References

- [Ergo.services Documentation](https://docs.ergo.services/)
- [Actor Model Basics](https://docs.ergo.services/basics/actor-model)
- [Supervisor Pattern](https://docs.ergo.services/actors/supervisor)
- [Pool Pattern](https://docs.ergo.services/actors/pool)
