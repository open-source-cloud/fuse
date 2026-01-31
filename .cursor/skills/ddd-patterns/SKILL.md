# DDD Patterns Skill

This skill provides expertise in Domain-Driven Design (DDD) patterns, including domain modeling, entities, value objects, aggregates, bounded contexts, repositories, and domain events in Go.

## Domain Modeling in Go

### Ubiquitous Language

- Use domain terminology in code
- Code reflects domain concepts
- Domain experts and developers share language
- Avoid technical jargon in domain layer

```go
// Good: Uses domain language
type Workflow struct {
    ID      WorkflowID
    Graph   *Graph
    State   WorkflowState
    Nodes   []*Node
}

// Bad: Uses technical terms
type WorkflowStruct struct {
    UUID    string
    Data    map[string]interface{}
    Status  int
}
```

### Domain vs Infrastructure

- Domain layer contains business logic
- Infrastructure layer contains technical details
- Domain doesn't depend on infrastructure
- Use interfaces to invert dependencies

## Entities vs Value Objects

### Entities

- Have identity (ID)
- Mutable (can change state)
- Compared by identity
- Lifecycle managed by repository

```go
// Entity: Has identity
type Workflow struct {
    id    WorkflowID  // Identity
    graph *Graph
    state WorkflowState
}

func (w *Workflow) ID() WorkflowID {
    return w.id
}

// Entities are compared by ID
func (w1 *Workflow) Equals(w2 *Workflow) bool {
    return w1.id == w2.id
}
```

### Value Objects

- No identity (compared by value)
- Immutable (create new instance to change)
- Self-validating
- Can be shared

```go
// Value Object: No identity, immutable
type WorkflowID string

func NewWorkflowID() WorkflowID {
    return WorkflowID(uuid.New().String())
}

func (id WorkflowID) String() string {
    return string(id)
}

// Value objects are compared by value
func (id1 WorkflowID) Equals(id2 WorkflowID) bool {
    return id1 == id2
}

// Value Object: Money
type Money struct {
    amount   decimal.Decimal
    currency string
}

func NewMoney(amount decimal.Decimal, currency string) Money {
    if amount.IsNegative() {
        panic("money amount cannot be negative")
    }
    return Money{amount: amount, currency: currency}
}

func (m Money) Add(other Money) Money {
    if m.currency != other.currency {
        panic("cannot add different currencies")
    }
    return NewMoney(m.amount.Add(other.amount), m.currency)
}
```

## Aggregates and Aggregate Roots

### Aggregates

- Cluster of entities and value objects
- Consistency boundary
- One aggregate root (entry point)
- External references only to aggregate root

```go
// Aggregate Root
type Workflow struct {
    id    WorkflowID
    graph *Graph
    nodes []*Node  // Entities within aggregate
    edges []*Edge  // Value objects within aggregate
}

// Aggregate root manages consistency
func (w *Workflow) AddNode(node *Node) error {
    // Validate business rules
    if w.state != WorkflowStateDraft {
        return ErrWorkflowNotEditable
    }
    
    // Maintain consistency
    w.nodes = append(w.nodes, node)
    return nil
}

// External code references aggregate root only
type WorkflowRepository interface {
    Get(id WorkflowID) (*Workflow, error)  // Returns aggregate root
    Save(workflow *Workflow) error
}
```

### Aggregate Design Rules

- Keep aggregates small
- Reference other aggregates by ID, not object
- One transaction = one aggregate
- Use domain events for cross-aggregate communication

```go
// Good: Reference by ID
type Workflow struct {
    id        WorkflowID
    packageID PackageID  // Reference to other aggregate
}

// Bad: Direct reference to other aggregate
type Workflow struct {
    id      WorkflowID
    pkg     *Package  // Tight coupling
}
```

## Domain Events

### Domain Events

- Represent something that happened in domain
- Immutable (created, not modified)
- Published when aggregate changes
- Used for cross-aggregate communication

```go
// Domain Event
type WorkflowStarted struct {
    WorkflowID WorkflowID
    StartedAt  time.Time
    TriggeredBy UserID
}

type WorkflowCompleted struct {
    WorkflowID  WorkflowID
    CompletedAt time.Time
    Result      map[string]any
}

// Aggregate publishes events
type Workflow struct {
    id      WorkflowID
    state   WorkflowState
    events  []DomainEvent  // Domain events to publish
}

func (w *Workflow) Start() {
    if w.state != WorkflowStatePending {
        panic("workflow not in pending state")
    }
    
    w.state = WorkflowStateRunning
    w.events = append(w.events, WorkflowStarted{
        WorkflowID: w.id,
        StartedAt:  time.Now(),
    })
}

func (w *Workflow) GetUncommittedEvents() []DomainEvent {
    return w.events
}

func (w *Workflow) MarkEventsAsCommitted() {
    w.events = nil
}
```

### Event Handlers

- Handle domain events
- Update read models (CQRS)
- Trigger side effects
- Send notifications

```go
type WorkflowEventHandler interface {
    HandleWorkflowStarted(event WorkflowStarted) error
    HandleWorkflowCompleted(event WorkflowCompleted) error
}

// Event handler updates read model
type WorkflowReadModelUpdater struct {
    readRepo WorkflowReadRepository
}

func (h *WorkflowReadModelUpdater) HandleWorkflowStarted(event WorkflowStarted) error {
    return h.readRepo.UpdateStatus(event.WorkflowID, "running")
}
```

## Bounded Contexts

### Context Boundaries

- Each bounded context has own domain model
- Different contexts can have same concept with different meaning
- Clear boundaries prevent coupling
- Context maps show relationships

### Context Mapping Patterns

- **Shared Kernel**: Shared code between contexts
- **Customer-Supplier**: Upstream/downstream relationship
- **Conformist**: Downstream conforms to upstream
- **Anticorruption Layer**: Translates between contexts
- **Separate Ways**: Independent contexts
- **Open Host Service**: Published language for integration
- **Published Language**: Well-documented integration language

```go
// Bounded Context: Workflow Management
package workflow

type Workflow struct {
    // Workflow domain model
}

// Bounded Context: Package Management
package packages

type Package struct {
    // Package domain model (different from workflow context)
}
```

## Repository Pattern Implementation

### Repository Interface

- Define in domain layer (or application layer)
- Abstracts data access
- Returns aggregates
- Methods reflect domain language

```go
// Repository interface (domain/application layer)
type WorkflowRepository interface {
    Get(id WorkflowID) (*Workflow, error)
    Save(workflow *Workflow) error
    FindByState(state WorkflowState) ([]*Workflow, error)
    Exists(id WorkflowID) bool
}

// Implementation (infrastructure layer)
type MongoWorkflowRepository struct {
    collection *mongo.Collection
}

func (r *MongoWorkflowRepository) Get(id WorkflowID) (*Workflow, error) {
    // MongoDB implementation
}
```

### Repository Best Practices

- One repository per aggregate root
- Return aggregates, not entities
- Use domain language in method names
- Handle persistence concerns in implementation

```go
// Good: Domain language
type WorkflowRepository interface {
    FindActiveWorkflows() ([]*Workflow, error)
    FindByOwner(ownerID UserID) ([]*Workflow, error)
}

// Bad: Technical language
type WorkflowRepository interface {
    SelectWhereStatusEquals(status string) ([]*Workflow, error)
    QueryByOwnerID(id string) ([]*Workflow, error)
}
```

## Domain Services vs Application Services

### Domain Services

- Contain domain logic that doesn't belong to entities
- Stateless operations
- Part of domain layer
- Use domain language

```go
// Domain Service: Complex domain logic
type WorkflowValidator struct {
    packageRegistry PackageRegistry
}

func (v *WorkflowValidator) ValidateWorkflow(workflow *Workflow) error {
    // Complex validation logic that involves multiple aggregates
    for _, node := range workflow.Nodes() {
        if !v.packageRegistry.Exists(node.FunctionID()) {
            return ErrFunctionNotFound
        }
    }
    return nil
}
```

### Application Services

- Orchestrate domain objects
- Coordinate use cases
- Transaction boundaries
- Part of application layer

```go
// Application Service: Orchestrates use case
type WorkflowService struct {
    workflowRepo WorkflowRepository
    graphRepo    GraphRepository
    validator    *WorkflowValidator
}

func (s *WorkflowService) CreateWorkflow(schema *GraphSchema) (*Workflow, error) {
    // 1. Validate schema
    if err := s.validator.ValidateSchema(schema); err != nil {
        return nil, err
    }
    
    // 2. Create graph
    graph, err := workflow.NewGraph(schema)
    if err != nil {
        return nil, err
    }
    
    // 3. Save graph
    if err := s.graphRepo.Save(graph); err != nil {
        return nil, err
    }
    
    // 4. Create workflow
    wf := workflow.New(graph)
    
    // 5. Save workflow
    if err := s.workflowRepo.Save(wf); err != nil {
        return nil, err
    }
    
    return wf, nil
}
```

## Best Practices

1. **Use ubiquitous language** - Code reflects domain
2. **Keep aggregates small** - Easier to maintain consistency
3. **Reference aggregates by ID** - Avoid tight coupling
4. **Use domain events** - Cross-aggregate communication
5. **Repository per aggregate** - Clear data access boundaries
6. **Separate domain and application services** - Different responsibilities
7. **Value objects are immutable** - Create new instances
8. **Entities have identity** - Compared by ID
9. **Domain layer is independent** - No infrastructure dependencies
10. **Test domain logic** - Unit tests for business rules

## References

- [Domain-Driven Design](https://www.domainlanguage.com/ddd/)
- [Implementing Domain-Driven Design](https://www.domainlanguage.com/ddd/)
- [DDD in Go](https://threedots.tech/tags/domain-driven-design/)
