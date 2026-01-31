# Go Senior Developer Skill

This skill provides comprehensive expertise in Go development, covering best practices, SOLID principles, clean code, error handling, interface design, concurrency, testing, and performance optimization.

## Go Best Practices

### Effective Go Guidelines

- Follow [Effective Go](https://golang.org/doc/effective_go) idioms and patterns
- Use meaningful variable and function names (PascalCase for exported, camelCase for private)
- Write comprehensive comments for public APIs
- Keep functions focused and small (single responsibility)
- Use interfaces to enable modularity and testability
- Prefer composition over inheritance
- Use zero values effectively
- Leverage Go's type system for safety

### Code Organization

```go
// Package structure: one primary type per file
// File: workflow.go
package workflow

type Workflow struct {
    // fields
}

func NewWorkflow() *Workflow {
    // constructor
}

func (w *Workflow) Execute() error {
    // methods
}
```

### Import Grouping

```go
import (
    // Standard library
    "context"
    "fmt"
    "time"

    // External dependencies
    "ergo.services/ergo/act"
    "github.com/rs/zerolog/log"

    // Internal imports
    "github.com/open-source-cloud/fuse/internal/workflow"
)
```

## SOLID Principles in Go

### Single Responsibility Principle (SRP)

Each type/function should have one reason to change:

```go
// Good: Separate concerns
type WorkflowRepository interface {
    Get(id string) (*Workflow, error)
    Save(workflow *Workflow) error
}

type WorkflowValidator struct {
    // Only validates workflows
}

// Bad: Mixed responsibilities
type WorkflowManager struct {
    // Validates, persists, executes, logs - too many responsibilities
}
```

### Open/Closed Principle (OCP)

Open for extension, closed for modification:

```go
// Good: Extensible through interfaces
type NodeExecutor interface {
    Execute(ctx context.Context, input map[string]any) (map[string]any, error)
}

type HTTPNodeExecutor struct {
    // implements NodeExecutor
}

type LogicNodeExecutor struct {
    // implements NodeExecutor
}

// Can add new executors without modifying existing code
```

### Liskov Substitution Principle (LSP)

Implementations must be substitutable:

```go
// Good: All implementations satisfy the interface contract
type Repository interface {
    Get(id string) (*Entity, error)
}

// MemoryRepository and MongoRepository both satisfy Repository
// Can be used interchangeably
```

### Interface Segregation Principle (ISP)

Clients shouldn't depend on interfaces they don't use:

```go
// Good: Small, focused interfaces
type Reader interface {
    Read() ([]byte, error)
}

type Writer interface {
    Write([]byte) error
}

// Bad: Large interface forcing unnecessary methods
type ReadWriter interface {
    Read() ([]byte, error)
    Write([]byte) error
    Close() error
    Flush() error
    // Many clients only need Read or Write
}
```

### Dependency Inversion Principle (DIP)

Depend on abstractions (interfaces), not concretions:

```go
// Good: Service depends on interface
type WorkflowService struct {
    repo WorkflowRepository // Interface, not concrete type
}

// Bad: Service depends on concrete implementation
type WorkflowService struct {
    repo *MongoWorkflowRepository // Concrete type
}
```

## Clean Code Principles

### Meaningful Names

- Use descriptive names that reveal intent
- Avoid abbreviations unless widely understood
- Use verbs for functions, nouns for types
- Be consistent with naming conventions

```go
// Good
func CalculateTotalPrice(items []Item) float64
func ValidateUserInput(input UserInput) error

// Bad
func Calc(x []Item) float64
func Val(i UserInput) error
```

### Functions

- Small and focused (single responsibility)
- Do one thing well
- Descriptive names
- Few parameters (prefer structs for many parameters)
- No side effects (when possible)

```go
// Good: Small, focused function
func ValidateEmail(email string) error {
    if !strings.Contains(email, "@") {
        return ErrInvalidEmail
    }
    return nil
}

// Bad: Does multiple things
func ProcessUser(user User) error {
    // Validates, saves, sends email, logs - too many responsibilities
}
```

### Error Handling

- Always handle errors explicitly
- Wrap errors with context
- Use custom error types for domain errors
- Don't ignore errors

```go
// Good: Explicit error handling with context
func (s *Service) ProcessWorkflow(id string) error {
    workflow, err := s.repo.Get(id)
    if err != nil {
        return fmt.Errorf("failed to get workflow %s: %w", id, err)
    }
    
    if err := s.executor.Execute(workflow); err != nil {
        return fmt.Errorf("failed to execute workflow %s: %w", id, err)
    }
    
    return nil
}
```

## Error Handling Patterns

### Error Wrapping

```go
// Wrap errors with context
if err != nil {
    return fmt.Errorf("operation failed: %w", err)
}
```

### Custom Error Types

```go
var (
    ErrWorkflowNotFound = errors.New("workflow not found")
    ErrInvalidSchema = errors.New("invalid schema")
)

// Check for specific errors
if errors.Is(err, ErrWorkflowNotFound) {
    // Handle not found
}
```

### Error Checking

```go
// Always check errors
result, err := doSomething()
if err != nil {
    return err
}

// Don't ignore errors
_ = doSomething() // BAD!
```

## Interface Design

### Small, Focused Interfaces

```go
// Good: Small, focused interface
type Reader interface {
    Read([]byte) (int, error)
}

// Compose larger interfaces from small ones
type ReadWriter interface {
    Reader
    Writer
}
```

### Interface Location

- Define interfaces where they're used, not where they're implemented
- This enables multiple implementations and testing

```go
// Good: Interface defined in service package
package service

type WorkflowRepository interface {
    Get(id string) (*workflow.Workflow, error)
}

type WorkflowService struct {
    repo WorkflowRepository // Uses interface
}
```

## Concurrency vs Parallelism

### Understanding the Difference

- **Concurrency**: Dealing with multiple things at once (structure)
- **Parallelism**: Doing multiple things at once (execution)

Go provides concurrency primitives (goroutines, channels) that can run in parallel on multiple cores.

### Goroutines

```go
// Launch goroutine for concurrent execution
go func() {
    processWorkflow(workflow)
}()

// Use sync.WaitGroup to wait for completion
var wg sync.WaitGroup
wg.Add(1)
go func() {
    defer wg.Done()
    processWorkflow(workflow)
}()
wg.Wait()
```

### Channels

```go
// Unbuffered channel (synchronous)
ch := make(chan int)

// Buffered channel (asynchronous up to buffer size)
ch := make(chan int, 10)

// Send
ch <- value

// Receive
value := <-ch

// Close channel when done
close(ch)
```

### Select Statement

```go
select {
case msg := <-ch1:
    // Handle message from ch1
case msg := <-ch2:
    // Handle message from ch2
case <-time.After(5 * time.Second):
    // Timeout
default:
    // Non-blocking
}
```

## Testing Strategies

### Unit Tests

- Test individual functions/methods
- Use table-driven tests for multiple scenarios
- Mock external dependencies
- Test error paths, not just happy paths

```go
func TestValidateEmail(t *testing.T) {
    tests := []struct {
        name    string
        email   string
        wantErr bool
    }{
        {"valid email", "test@example.com", false},
        {"invalid email", "invalid", true},
        {"empty email", "", true},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := ValidateEmail(tt.email)
            if (err != nil) != tt.wantErr {
                t.Errorf("ValidateEmail() error = %v, wantErr %v", err, tt.wantErr)
            }
        })
    }
}
```

### Integration Tests

- Test component interactions
- Use real dependencies (databases, services)
- Clean up resources
- Skip in short mode

```go
func TestMongoRepository_Integration(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping integration test")
    }
    // Test with real MongoDB
}
```

### Test Helpers

```go
// Create test helpers to reduce boilerplate
func createTestWorkflow() *Workflow {
    // Setup test data
}

func createTestGraph() *Graph {
    // Setup test graph
}
```

## Performance Optimization

### Profiling

- Use `go tool pprof` for CPU and memory profiling
- Identify bottlenecks before optimizing
- Measure, don't guess

### Memory Management

- Avoid unnecessary allocations
- Use object pooling for frequently allocated objects
- Be aware of escape analysis
- Use sync.Pool for temporary objects

### Efficient Data Structures

- Choose appropriate data structures
- Use maps for O(1) lookups
- Use slices efficiently (pre-allocate capacity)
- Avoid unnecessary copies

## Best Practices Summary

1. **Write tests first** (TDD approach)
2. **Use interfaces** for modularity and testability
3. **Handle errors explicitly** with context
4. **Keep functions small** and focused
5. **Use meaningful names** that reveal intent
6. **Follow Effective Go** guidelines
7. **Apply SOLID principles** appropriately
8. **Understand concurrency** vs parallelism
9. **Profile before optimizing**
10. **Document public APIs** comprehensively

## References

- [Effective Go](https://golang.org/doc/effective_go)
- [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- [Go Best Practices](https://github.com/cristaloleg/go-advices)
- [Clean Code in Go](https://github.com/Pungyeon/clean-go-article)
