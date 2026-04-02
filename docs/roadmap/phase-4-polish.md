# Phase 4: Polish — Make It Great

> **Goal:** Add higher-level patterns and operational polish that make Fuse a delight to use for complex automation scenarios. Depends on Phases 1-3.

---

## 4.1 Loop Over Items / Batch Processing

### Motivation

The current loop support ([#22](https://github.com/open-source-cloud/fuse/issues/22)) enables graph-level cycles — a node's output edge can point back to a previous node, creating a loop in the graph. However, there is no built-in pattern for the most common loop use case: **iterating over a collection of items**.

Use cases:
- Process each item in an order: validate, charge, ship
- Send notifications to a list of users
- Fetch data from a paginated API, processing each page
- Transform each row in a dataset
- Batch-process items (e.g., 10 at a time for bulk API calls)

Currently, implementing iteration requires building a manual loop in the graph with a counter node, a condition node, and an indexer — unnecessarily complex for a common pattern.

### Prior Art

**n8n** provides two iteration primitives:
- **Loop Over Items**: Automatically iterates over all items in the input, executing the connected nodes for each item. Handles batching internally.
- **Split In Batches**: Splits input items into configurable batch sizes, executing the loop body once per batch. Supports "done" and "loop" outputs for explicit flow control.

Key n8n design decisions:
- Items are the fundamental data unit — every node receives and outputs items
- Looping happens at the "item level" automatically in most nodes
- Explicit loop nodes are for when you need to control batch size or iteration order

**Inngest** doesn't have a built-in loop primitive. Instead, loops are expressed as code:
```typescript
for (const item of items) {
    await step.run(`process-${item.id}`, () => processItem(item));
}
```
Each step inside the loop is independently durable and retryable.

**What Fuse should adopt:** A `system/foreach` control node that takes an array input, iterates over items (or batches), and executes a subgraph per item/batch. This should leverage the parallel threading model for concurrent batch processing.

### Design

#### 4.1.1 ForEach Function (Internal Package)

```go
// internal/packages/internal/foreach/foreach.go

const (
    PackageID  = "system"
    FunctionID = "foreach"
)

func Metadata() workflow.FunctionMetadata {
    return workflow.FunctionMetadata{
        Transport: transport.Internal,
        Input: workflow.InputMetadata{
            Parameters: []workflow.ParameterSchema{
                {Name: "items", Type: "[]any", Required: true, Description: "Array of items to iterate over"},
                {Name: "batchSize", Type: "int", Required: false, Default: 1, Description: "Number of items per batch (1 = item-by-item)"},
                {Name: "concurrency", Type: "int", Required: false, Default: 1, Description: "Max concurrent batches (1 = sequential)"},
            },
        },
        Output: workflow.OutputMetadata{
            Parameters: []workflow.ParameterSchema{
                {Name: "item", Type: "any", Description: "Current item (when batchSize=1)"},
                {Name: "batch", Type: "[]any", Description: "Current batch (when batchSize>1)"},
                {Name: "index", Type: "int", Description: "Current index/batch number"},
                {Name: "total", Type: "int", Description: "Total number of items"},
                {Name: "isLast", Type: "bool", Description: "True if this is the last item/batch"},
            },
            Edges: []workflow.OutputEdgeMetadata{
                {Name: "each", Count: 1, Description: "Executed for each item/batch"},
                {Name: "done", Count: 1, Description: "Executed after all items are processed"},
            },
        },
    }
}
```

#### 4.1.2 New Action Type

```go
// internal/workflow/workflowactions/action.go

const ActionForEach ActionType = "workflow:foreach"

// ForEachAction instructs the handler to iterate over items
type ForEachAction struct {
    ThreadID    uint16
    ExecID      workflow.ExecID
    Items       []any
    BatchSize   int
    Concurrency int
    // LoopBody describes the subgraph to execute per item/batch
    LoopBodyEdge *Edge // edge to the first node in the loop body
    // DoneEdge describes the edge to follow after all items are processed
    DoneEdge *Edge
}

func (a *ForEachAction) Type() ActionType { return ActionForEach }
```

#### 4.1.3 Loop Execution Model

The ForEach node orchestrates iteration through the workflow's existing threading model:

```
ForEach Node (Thread 0)
  |
  v  items = [A, B, C, D], batchSize=1, concurrency=2
  |
  +-- Thread 1: process(A) -> Thread 1 done
  +-- Thread 2: process(B) -> Thread 2 done
  |   (wait for slot)
  +-- Thread 3: process(C) -> Thread 3 done
  +-- Thread 4: process(D) -> Thread 4 done
  |
  v  All iterations complete
  |
  ForEach Node collects results
  |
  v  "done" edge -> next node
```

#### 4.1.4 ForEach State Tracking

```go
// internal/workflow/foreach_state.go

// ForEachState tracks the progress of a ForEach iteration
type ForEachState struct {
    mu            sync.Mutex
    ExecID        string         `json:"execId"`
    Items         []any          `json:"items"`
    BatchSize     int            `json:"batchSize"`
    Concurrency   int            `json:"concurrency"`
    TotalBatches  int            `json:"totalBatches"`
    Completed     int            `json:"completed"`
    Results       []any          `json:"results"` // collected results from each iteration
    ActiveThreads map[uint16]int `json:"activeThreads"` // threadID -> batch index
}

func NewForEachState(execID string, items []any, batchSize, concurrency int) *ForEachState {
    totalBatches := len(items) / batchSize
    if len(items)%batchSize != 0 {
        totalBatches++
    }
    return &ForEachState{
        ExecID:        execID,
        Items:         items,
        BatchSize:     batchSize,
        Concurrency:   concurrency,
        TotalBatches:  totalBatches,
        Results:       make([]any, totalBatches),
        ActiveThreads: make(map[uint16]int),
    }
}

// RecordCompletion marks a batch as complete and returns the next batch index to process (-1 if done)
func (s *ForEachState) RecordCompletion(threadID uint16, result any) (nextBatchIndex int, allDone bool) {
    s.mu.Lock()
    defer s.mu.Unlock()

    batchIndex := s.ActiveThreads[threadID]
    s.Results[batchIndex] = result
    s.Completed++
    delete(s.ActiveThreads, threadID)

    if s.Completed >= s.TotalBatches {
        return -1, true
    }

    // Find next unstarted batch
    nextIndex := s.findNextBatch()
    return nextIndex, false
}

func (s *ForEachState) findNextBatch() int {
    started := make(map[int]bool)
    for _, idx := range s.ActiveThreads {
        started[idx] = true
    }
    for i := 0; i < s.TotalBatches; i++ {
        if !started[i] && s.Results[i] == nil {
            return i
        }
    }
    return -1
}

// GetBatch returns the items for a given batch index
func (s *ForEachState) GetBatch(batchIndex int) []any {
    start := batchIndex * s.BatchSize
    end := start + s.BatchSize
    if end > len(s.Items) {
        end = len(s.Items)
    }
    return s.Items[start:end]
}
```

#### 4.1.5 WorkflowHandler ForEach Handling

```go
func (a *WorkflowHandler) handleForEachAction(action *workflowactions.ForEachAction) {
    state := NewForEachState(
        action.ExecID.String(),
        action.Items,
        action.BatchSize,
        action.Concurrency,
    )
    a.forEachStates[action.ExecID.String()] = state

    // Start initial concurrent batches
    for i := 0; i < min(action.Concurrency, state.TotalBatches); i++ {
        batch := state.GetBatch(i)
        a.spawnForEachIteration(action, state, i, batch)
    }
}

func (a *WorkflowHandler) spawnForEachIteration(
    action *workflowactions.ForEachAction,
    state *ForEachState,
    batchIndex int,
    batch []any,
) {
    // Create iteration-specific input
    var input map[string]any
    if action.BatchSize == 1 && len(batch) == 1 {
        input = map[string]any{
            "item":   batch[0],
            "index":  batchIndex,
            "total":  len(action.Items),
            "isLast": batchIndex == state.TotalBatches-1,
        }
    } else {
        input = map[string]any{
            "batch":  batch,
            "index":  batchIndex,
            "total":  len(action.Items),
            "isLast": batchIndex == state.TotalBatches-1,
        }
    }

    // Spawn a new thread for this iteration
    threadID := a.workflow.AllocateThread()
    state.ActiveThreads[threadID] = batchIndex

    execID := workflow.NewExecID(threadID)
    execMsg := messaging.NewExecuteFunctionMessage(a.workflow.ID(), &workflowactions.RunFunctionAction{
        ThreadID:       threadID,
        FunctionID:     action.LoopBodyEdge.To().FunctionID(),
        FunctionExecID: execID,
        Args:           input,
    })

    pool := WorkflowFuncPoolName(a.workflow.ID())
    a.Send(pool, execMsg)
}
```

#### 4.1.6 Iteration Completion Handling

When a ForEach iteration thread completes, the handler checks if more batches need processing:

```go
// In handleMsgFunctionResult, after Next() returns NoopAction and thread is finished:
if forEachState, isForEach := a.findForEachForThread(threadID); isForEach {
    result := fnResultMsg.Result.Output.Data
    nextBatch, allDone := forEachState.RecordCompletion(threadID, result)

    if allDone {
        // All iterations complete — aggregate results and continue to "done" edge
        aggregated := map[string]any{"results": forEachState.Results}
        a.workflow.SetForEachResult(forEachState.ExecID, aggregated)
        // Follow the "done" edge
        doneAction := a.workflow.NextForEachDone(forEachState.ExecID)
        a.handleWorkflowAction(doneAction)
        delete(a.forEachStates, forEachState.ExecID)
    } else if nextBatch >= 0 {
        // More batches — start next iteration
        batch := forEachState.GetBatch(nextBatch)
        a.spawnForEachIteration(originalAction, forEachState, nextBatch, batch)
    }
}
```

#### 4.1.7 Graph Integration

In the graph schema, a ForEach node has two named output edges:
- `each` edge: connects to the first node of the loop body
- `done` edge: connects to the node that runs after all iterations complete

```json
{
    "nodes": [
        {"id": "foreach1", "functionId": "system/foreach"},
        {"id": "process", "functionId": "mypackage/processItem"},
        {"id": "aggregate", "functionId": "mypackage/aggregate"}
    ],
    "edges": [
        {"from": "foreach1", "to": "process", "conditional": {"name": "each"}},
        {"from": "foreach1", "to": "aggregate", "conditional": {"name": "done"}}
    ]
}
```

### Alternatives Considered

1. **Unrolling loops into parallel branches**: Expand the ForEach into N parallel branches at graph creation time. Doesn't work because the number of items is only known at runtime.

2. **Recursive sub-workflows**: Use sub-workflows (Phase 2.3) to process each item. Works but heavy — each iteration spawns an entire workflow instance. ForEach as a built-in primitive is more efficient.

3. **Code-level loops (like Inngest)**: Fuse is graph-based, not code-based. A graph-level ForEach node is the natural equivalent of Inngest's code-level loops.

### Migration Plan

- New `system/foreach` internal package registered alongside existing internal packages
- New action type and state tracking added to WorkflowHandler
- No changes to existing schemas — ForEach is opt-in via the new function ID
- Loop body subgraph is defined using existing edge/node primitives

### Open Questions

1. Should ForEach support early exit (break) when a condition is met?
2. Should iteration order be guaranteed (sequential) or undefined (parallel) when concurrency > 1?
3. How should errors in individual iterations be handled — fail the entire ForEach, or collect errors and continue?
4. Should there be a `system/map` variant that transforms items without side effects?
5. Should ForEach support nested loops (ForEach inside ForEach)?

---

## 4.2 Workflow Versioning

### Motivation

Currently, workflow schemas are mutable. Calling `PUT /v1/schemas/{schemaID}` (`internal/handlers/workflow_schema.go:59`) replaces the schema in-place via `graphService.Upsert()`:

```go
func (h *WorkflowSchemaHandler) HandlePut(from gen.PID, w http.ResponseWriter, r *http.Request) error {
    // ...
    graph, err := h.graphService.Upsert(schemaID, &schema)
    // ...
}
```

The `GraphService.Upsert` method (`internal/services/graph_service.go:54-75`) either creates or updates the graph:

```go
func (s *DefaultGraphService) Upsert(schemaID string, schema *GraphSchema) (*Graph, error) {
    existingGraph, err := s.graphRepo.FindByID(schemaID)
    if err != nil {
        return s.create(schema)
    }
    return s.update(existingGraph, schema)
}
```

Problems with mutable schemas:
- **Running workflows affected**: If a schema is updated while a workflow is running, the running workflow uses the old in-memory graph but new workflows get the updated schema. This is accidental versioning with no guarantees.
- **No rollback**: If a schema update introduces a bug, there's no way to revert to the previous version.
- **No audit trail**: No record of what changed between schema versions.
- **No safe deployment**: Can't do canary deployments (run 10% of traffic on new version).

### Prior Art

**n8n** provides:
- **Workflow versions**: Each save creates a new version. Users can view version history and restore previous versions.
- **Version comparison**: Visual diff between workflow versions.
- **Active version**: Only one version is "active" at a time; new executions use the active version.

**Inngest** uses:
- **Function versioning via code deployment**: Since functions are defined in code, versioning happens through the deployment process. Inngest tracks which version is deployed and can route events to the correct version.

**Restate** doesn't version workflows per se, but its deployment model supports rolling updates — new invocations use the new code while existing invocations complete on the old code.

**What Fuse should adopt:** Schema versioning where each update creates a new version. Running workflows are pinned to the version they started with. New workflows use the latest (or explicitly specified) version. Support rollback to any previous version.

### Design

#### 4.2.1 Versioned Schema Model

```go
// internal/workflow/versioned_schema.go

// SchemaVersion represents a specific version of a workflow schema
type SchemaVersion struct {
    SchemaID  string       `json:"schemaId" bson:"schemaId"`
    Version   int          `json:"version" bson:"version"`
    Schema    GraphSchema  `json:"schema" bson:"schema"`
    CreatedAt time.Time    `json:"createdAt" bson:"createdAt"`
    CreatedBy string       `json:"createdBy,omitempty" bson:"createdBy,omitempty"`
    Comment   string       `json:"comment,omitempty" bson:"comment,omitempty"`
    IsActive  bool         `json:"isActive" bson:"isActive"`
}

// SchemaVersionHistory tracks all versions of a schema
type SchemaVersionHistory struct {
    SchemaID       string `json:"schemaId" bson:"schemaId"`
    ActiveVersion  int    `json:"activeVersion" bson:"activeVersion"`
    LatestVersion  int    `json:"latestVersion" bson:"latestVersion"`
    TotalVersions  int    `json:"totalVersions" bson:"totalVersions"`
}
```

#### 4.2.2 Graph Repository Extension

```go
// Extend GraphRepository
type GraphRepository interface {
    // Existing methods (unchanged behavior — operate on active version)
    FindByID(id string) (*workflow.Graph, error)
    Save(graph *workflow.Graph) error

    // Version-aware methods (NEW)
    FindByIDAndVersion(id string, version int) (*workflow.Graph, error)
    SaveVersion(schemaVersion *workflow.SchemaVersion) error
    ListVersions(schemaID string) ([]workflow.SchemaVersion, error)
    SetActiveVersion(schemaID string, version int) error
    GetVersionHistory(schemaID string) (*workflow.SchemaVersionHistory, error)
}
```

#### 4.2.3 GraphService Changes

```go
// internal/services/graph_service.go

type GraphService interface {
    FindByID(schemaID string) (*workflow.Graph, error)
    FindByIDAndVersion(schemaID string, version int) (*workflow.Graph, error) // NEW
    Upsert(schemaID string, schema *workflow.GraphSchema) (*workflow.Graph, error)
    ListVersions(schemaID string) ([]workflow.SchemaVersion, error) // NEW
    SetActiveVersion(schemaID string, version int) error // NEW
    Rollback(schemaID string, version int) error // NEW
}

func (s *DefaultGraphService) Upsert(schemaID string, schema *GraphSchema) (*Graph, error) {
    // Populate metadata
    if err := s.populateNodeMetadata(graph, schema.Nodes); err != nil {
        return nil, err
    }

    existingGraph, err := s.graphRepo.FindByID(schemaID)
    if err != nil {
        // New schema — version 1
        return s.createVersioned(schema, 1)
    }

    // Existing schema — increment version
    history, _ := s.graphRepo.GetVersionHistory(schemaID)
    newVersion := history.LatestVersion + 1

    // Save as new version
    schemaVersion := &workflow.SchemaVersion{
        SchemaID:  schemaID,
        Version:   newVersion,
        Schema:    *schema,
        CreatedAt: time.Now(),
        IsActive:  true, // New version becomes active by default
    }

    // Deactivate previous active version
    s.graphRepo.SetActiveVersion(schemaID, newVersion)
    s.graphRepo.SaveVersion(schemaVersion)

    // Also save to main graph store (for FindByID compatibility)
    return s.update(existingGraph, schema)
}

func (s *DefaultGraphService) Rollback(schemaID string, version int) error {
    oldVersion, err := s.graphRepo.FindByIDAndVersion(schemaID, version)
    if err != nil {
        return fmt.Errorf("version %d not found for schema %s: %w", version, schemaID, err)
    }

    // Create a new version with the old schema content
    history, _ := s.graphRepo.GetVersionHistory(schemaID)
    newVersion := history.LatestVersion + 1

    schemaVersion := &workflow.SchemaVersion{
        SchemaID:  schemaID,
        Version:   newVersion,
        Schema:    oldVersion.Schema(),
        CreatedAt: time.Now(),
        Comment:   fmt.Sprintf("Rollback to version %d", version),
        IsActive:  true,
    }

    s.graphRepo.SetActiveVersion(schemaID, newVersion)
    return s.graphRepo.SaveVersion(schemaVersion)
}
```

#### 4.2.4 Workflow Pinning

When a workflow is triggered, it pins to the current active version:

```go
// In WorkflowHandler.Init(), when creating a new workflow:
graphRef, err := a.graphService.FindByID(initArgs.schemaID)
if err != nil {
    return gen.TerminateReasonPanic
}

// Record the version this workflow is running on
versionHistory, _ := a.graphService.GetVersionHistory(initArgs.schemaID)
a.workflow = internalworkflow.New(initArgs.workflowID, graphRef)
a.workflow.SetSchemaVersion(versionHistory.ActiveVersion) // NEW
```

On resume (Phase 1.1), load the specific version:

```go
// In Resume path:
version := a.workflow.SchemaVersion()
graphRef, err := a.graphService.FindByIDAndVersion(initArgs.schemaID, version)
```

#### 4.2.5 HTTP API Extensions

```
# List versions
GET /v1/schemas/{schemaID}/versions
Response 200:
{
    "schemaId": "my-workflow",
    "activeVersion": 3,
    "latestVersion": 3,
    "versions": [
        {"version": 1, "createdAt": "2025-06-01T...", "isActive": false},
        {"version": 2, "createdAt": "2025-06-15T...", "isActive": false},
        {"version": 3, "createdAt": "2025-07-01T...", "isActive": true}
    ]
}

# Get specific version
GET /v1/schemas/{schemaID}/versions/{version}
Response 200:
{
    "schemaId": "my-workflow",
    "version": 2,
    "schema": { ... },
    "createdAt": "2025-06-15T...",
    "isActive": false
}

# Set active version (rollback or promote)
POST /v1/schemas/{schemaID}/versions/{version}/activate
Response 200:
{
    "schemaId": "my-workflow",
    "activeVersion": 2,
    "previousVersion": 3
}

# Rollback (creates new version with old content)
POST /v1/schemas/{schemaID}/rollback
{
    "version": 1,
    "comment": "Rolling back due to bug in v3"
}
Response 200:
{
    "schemaId": "my-workflow",
    "newVersion": 4,
    "restoredFrom": 1
}
```

#### 4.2.6 Version Handler

```go
// internal/handlers/schema_versions.go

const (
    SchemaVersionsHandlerName     = "schema_versions_handler"
    SchemaVersionsHandlerPoolName = "schema_versions_handler_pool"
)

type SchemaVersionsHandlerFactory HandlerFactory[*SchemaVersionsHandler]

type SchemaVersionsHandler struct {
    Handler
    graphService services.GraphService
}

func (h *SchemaVersionsHandler) HandleGet(from gen.PID, w http.ResponseWriter, r *http.Request) error {
    schemaID, err := h.GetPathParam(r, "schemaID")
    if err != nil {
        return h.SendBadRequest(w, err, EmptyFields)
    }

    // Check if specific version requested
    versionStr, verr := h.GetPathParam(r, "version")
    if verr == nil && versionStr != "" {
        version, _ := strconv.Atoi(versionStr)
        graph, err := h.graphService.FindByIDAndVersion(schemaID, version)
        if err != nil {
            return h.SendNotFound(w, "version not found", EmptyFields)
        }
        return h.SendJSON(w, http.StatusOK, graph.Schema())
    }

    // List all versions
    versions, err := h.graphService.ListVersions(schemaID)
    if err != nil {
        return h.SendNotFound(w, "schema not found", EmptyFields)
    }
    return h.SendJSON(w, http.StatusOK, versions)
}
```

#### 4.2.7 Trigger with Specific Version

Extend the trigger API to optionally specify a schema version:

```go
type TriggerWorkflowRequest struct {
    SchemaID       string `json:"schemaId" validate:"required"`
    IdempotencyKey string `json:"idempotencyKey,omitempty"`
    SchemaVersion  *int   `json:"schemaVersion,omitempty"` // NEW: pin to specific version
}
```

### Alternatives Considered

1. **Git-style branching**: Full branch/merge model for schemas. Over-engineered — workflow schemas are single documents, not code trees. Linear versioning is sufficient.

2. **Copy-on-write schemas**: Each workflow execution gets a full copy of the schema. Wasteful storage-wise. Version pinning (just storing the version number) is more efficient.

3. **Immutable schemas with new IDs**: Each "update" creates a new schema with a new ID. Breaks references and doesn't provide a clear upgrade path. Versioned schemas under the same ID are more intuitive.

4. **No rollback, only create-new**: Users can only move forward by creating new versions. Rollback via re-creating is tedious. Explicit rollback is a better UX.

### Migration Plan

- Existing schemas become version 1 when the migration runs
- `GraphRepository.FindByID()` continues to return the active version (backwards compatible)
- New version-aware methods are additive
- Workflow struct gains a `schemaVersion int` field — existing workflows default to version 1
- The PUT endpoint (`/v1/schemas/{schemaID}`) now creates versions instead of replacing — same API, versioned backend

#### MongoDB Migration

```go
// Create version collection, migrate existing schemas:
// For each schema in "graphs" collection:
//   1. Copy to "graph_versions" with version=1, isActive=true
//   2. Add "activeVersion: 1" field to the original document
```

### Open Questions

1. Should there be a maximum number of versions to retain per schema?
2. Should schema updates require a "comment" describing what changed?
3. Should there be a diff API that shows what changed between two versions?
4. Should canary deployment be supported (route % of traffic to a specific version)?
5. Should version pinning in triggers require explicit opt-in, or should all triggers specify a version?
