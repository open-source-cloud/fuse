# Clean & Hexagonal Architecture Skill

This skill provides expertise in Clean Architecture and Hexagonal Architecture (Ports and Adapters) patterns in Go, including dependency inversion, layer separation, and testable design.

## Dependency Inversion Principle

### Core Principle

- High-level modules don't depend on low-level modules
- Both depend on abstractions (interfaces)
- Abstractions don't depend on details
- Details depend on abstractions

```go
// Bad: High-level depends on low-level
type WorkflowService struct {
    repo *MongoWorkflowRepository  // Depends on concrete implementation
}

// Good: Both depend on abstraction
type WorkflowService struct {
    repo WorkflowRepository  // Depends on interface
}

type MongoWorkflowRepository struct {
    // Implements WorkflowRepository interface
}
```

### Dependency Direction

- Domain layer: No dependencies (innermost)
- Application layer: Depends on domain
- Infrastructure layer: Depends on application/domain
- Presentation layer: Depends on application/domain

```
┌─────────────────┐
│  Presentation   │  (HTTP handlers, CLI)
└────────┬────────┘
         │ depends on
┌────────▼────────┐
│  Application    │  (Use cases, services)
└────────┬────────┘
         │ depends on
┌────────▼────────┐
│    Domain      │  (Entities, value objects, domain services)
└────────────────┘
         ▲
         │ depends on
┌────────┴────────┐
│ Infrastructure │  (Repositories, external services)
└────────────────┘
```

## Ports and Adapters Pattern

### Ports (Interfaces)

- Define contracts for external dependencies
- Defined in domain/application layer
- Technology-agnostic
- Enable testing and swapping implementations

```go
// Port: Repository interface (defined in domain/application layer)
type WorkflowRepository interface {
    Get(id WorkflowID) (*Workflow, error)
    Save(workflow *Workflow) error
}

// Port: External service interface
type NotificationService interface {
    SendNotification(recipient string, message string) error
}
```

### Adapters (Implementations)

- Implement ports
- Handle technology-specific details
- Located in infrastructure layer
- Can be swapped without changing core logic

```go
// Adapter: MongoDB implementation
type MongoWorkflowRepository struct {
    collection *mongo.Collection
}

func (r *MongoWorkflowRepository) Get(id WorkflowID) (*Workflow, error) {
    // MongoDB-specific implementation
}

// Adapter: In-memory implementation (for testing)
type MemoryWorkflowRepository struct {
    workflows map[WorkflowID]*Workflow
}

func (r *MemoryWorkflowRepository) Get(id WorkflowID) (*Workflow, error) {
    // In-memory implementation
}
```

## Layer Separation

### Domain Layer

- Core business logic
- Entities and value objects
- Domain services
- No external dependencies
- Pure Go code

```go
// Domain Layer: Entities
package workflow

type Workflow struct {
    id    WorkflowID
    graph *Graph
    state WorkflowState
}

func (w *Workflow) Start() error {
    // Domain logic
    if w.state != WorkflowStatePending {
        return ErrInvalidState
    }
    w.state = WorkflowStateRunning
    return nil
}

// Domain Layer: Value Objects
type WorkflowID string

// Domain Layer: Domain Services
type WorkflowValidator struct {
    // Domain validation logic
}
```

### Application Layer

- Use cases and orchestration
- Application services
- Depends on domain layer
- Defines ports (interfaces)
- No infrastructure details

```go
// Application Layer: Use Cases
package services

type WorkflowService struct {
    workflowRepo WorkflowRepository  // Port (interface)
    graphRepo    GraphRepository     // Port (interface)
    validator    *workflow.Validator // Domain service
}

func (s *WorkflowService) CreateWorkflow(schema *GraphSchema) (*Workflow, error) {
    // Orchestrate domain objects
    // Use ports (interfaces), not adapters (implementations)
}
```

### Infrastructure Layer

- Implements ports (adapters)
- Database access
- External service clients
- File system access
- Depends on application/domain layers

```go
// Infrastructure Layer: Adapters
package repositories

type MongoWorkflowRepository struct {
    collection *mongo.Collection
}

func (r *MongoWorkflowRepository) Get(id WorkflowID) (*Workflow, error) {
    // MongoDB-specific implementation
    // Implements WorkflowRepository port
}
```

### Presentation Layer

- HTTP handlers
- CLI commands
- API endpoints
- Depends on application layer
- Handles request/response

```go
// Presentation Layer: HTTP Handlers
package handlers

type WorkflowHandler struct {
    service services.WorkflowService  // Application service
}

func (h *WorkflowHandler) HandlePost(w http.ResponseWriter, r *http.Request) {
    // Handle HTTP request
    // Call application service
    workflow, err := h.service.CreateWorkflow(schema)
    // Return HTTP response
}
```

## Dependency Injection with fx

### Using uber-go/fx

- Constructor-based dependency injection
- Automatic dependency resolution
- Lifecycle management
- Testing support

```go
// Provide dependencies
fx.Provide(
    repositories.NewMemoryWorkflowRepository,
    repositories.NewMemoryGraphRepository,
    services.NewWorkflowService,
)

// Use fx.In for parameter structs
type WorkflowServiceParams struct {
    fx.In
    
    WorkflowRepo repositories.WorkflowRepository
    GraphRepo    repositories.GraphRepository
}

func NewWorkflowService(p WorkflowServiceParams) services.WorkflowService {
    return &workflowService{
        workflowRepo: p.WorkflowRepo,
        graphRepo:    p.GraphRepo,
    }
}
```

### Interface Annotation

```go
// Provide as interface
fx.Provide(
    fx.Annotate(
        repositories.NewMemoryWorkflowRepository,
        fx.As(new(repositories.WorkflowRepository)),
    ),
)
```

## Interface-Based Design

### Define Interfaces Where Used

- Define interfaces in the package that uses them
- Enables multiple implementations
- Makes testing easier
- Reduces coupling

```go
// Application layer defines what it needs
package services

type WorkflowRepository interface {
    Get(id WorkflowID) (*Workflow, error)
    Save(workflow *Workflow) error
}

type WorkflowService struct {
    repo WorkflowRepository  // Uses interface
}

// Infrastructure layer implements interface
package repositories

type MongoWorkflowRepository struct {
    // Implements services.WorkflowRepository
}
```

### Small, Focused Interfaces

```go
// Good: Small, focused interface
type Reader interface {
    Read([]byte) (int, error)
}

type Writer interface {
    Write([]byte) (int, error)
}

// Compose larger interfaces
type ReadWriter interface {
    Reader
    Writer
}
```

## Testing Strategies for Each Layer

### Domain Layer Testing

- Unit tests for entities and value objects
- Test business rules
- No mocks needed (pure logic)
- Fast, isolated tests

```go
func TestWorkflow_Start(t *testing.T) {
    workflow := NewWorkflow(graph)
    
    err := workflow.Start()
    
    require.NoError(t, err)
    assert.Equal(t, WorkflowStateRunning, workflow.State())
}
```

### Application Layer Testing

- Mock ports (interfaces)
- Test use case orchestration
- Verify business logic flow
- Fast, isolated tests

```go
func TestWorkflowService_CreateWorkflow(t *testing.T) {
    mockRepo := new(mocks.MockWorkflowRepository)
    mockGraphRepo := new(mocks.MockGraphRepository)
    
    service := services.NewWorkflowService(mockRepo, mockGraphRepo)
    
    // Test use case
    workflow, err := service.CreateWorkflow(schema)
    
    require.NoError(t, err)
    mockRepo.AssertExpectations(t)
}
```

### Infrastructure Layer Testing

- Integration tests with real dependencies
- Test adapter implementations
- May require test databases/services
- Slower, but verify integration

```go
func TestMongoWorkflowRepository_Integration(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping integration test")
    }
    
    repo := repositories.NewMongoWorkflowRepository(client, config)
    
    // Test with real MongoDB
    workflow, err := repo.Get(id)
    // ...
}
```

### Presentation Layer Testing

- Test HTTP handlers
- Mock application services
- Test request/response handling
- Fast, isolated tests

```go
func TestWorkflowHandler_HandlePost(t *testing.T) {
    mockService := new(mocks.MockWorkflowService)
    handler := handlers.NewWorkflowHandler(mockService)
    
    req := httptest.NewRequest("POST", "/workflows", body)
    w := httptest.NewRecorder()
    
    err := handler.HandlePost(gen.PID{}, w, req)
    
    require.NoError(t, err)
    assert.Equal(t, http.StatusOK, w.Code)
}
```

## Go-Specific Implementation Patterns

### Package Structure

```
internal/
├── domain/           # Domain layer
│   └── workflow/
│       ├── workflow.go
│       ├── graph.go
│       └── node.go
├── application/      # Application layer
│   └── services/
│       └── workflow_service.go
├── infrastructure/  # Infrastructure layer
│   └── repositories/
│       ├── mongo_workflow_repo.go
│       └── memory_workflow_repo.go
└── presentation/     # Presentation layer
    └── handlers/
        └── workflow_handler.go
```

### Error Handling Across Layers

```go
// Domain layer: Domain errors
var (
    ErrWorkflowNotFound = errors.New("workflow not found")
    ErrInvalidState = errors.New("invalid workflow state")
)

// Application layer: Wrap domain errors
func (s *WorkflowService) GetWorkflow(id WorkflowID) (*Workflow, error) {
    workflow, err := s.repo.Get(id)
    if err != nil {
        return nil, fmt.Errorf("failed to get workflow: %w", err)
    }
    return workflow, nil
}

// Presentation layer: Map to HTTP errors
func (h *WorkflowHandler) HandleGet(w http.ResponseWriter, r *http.Request) {
    workflow, err := h.service.GetWorkflow(id)
    if err != nil {
        if errors.Is(err, domain.ErrWorkflowNotFound) {
            http.Error(w, "Not Found", http.StatusNotFound)
            return
        }
        http.Error(w, "Internal Error", http.StatusInternalServerError)
        return
    }
    // Return workflow
}
```

## Best Practices

1. **Dependency inversion** - Depend on abstractions
2. **Layer separation** - Clear boundaries between layers
3. **Ports and adapters** - Interfaces in core, implementations outside
4. **Testability** - Each layer independently testable
5. **No circular dependencies** - Dependencies flow inward
6. **Domain independence** - Domain layer has no external dependencies
7. **Interface-based design** - Use interfaces for all external dependencies
8. **Dependency injection** - Use fx for DI
9. **Small interfaces** - Focused, composable interfaces
10. **Test each layer** - Appropriate testing strategy per layer

## References

- [Clean Architecture](https://blog.cleancoder.com/uncle-bob/2012/08/13/the-clean-architecture.html)
- [Hexagonal Architecture](https://alistair.cockburn.us/hexagonal-architecture/)
- [Go Clean Architecture](https://github.com/bxcodec/go-clean-arch)
