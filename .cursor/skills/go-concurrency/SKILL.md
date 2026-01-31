# Go Concurrency Skill

This skill provides expertise in Go concurrency patterns, including goroutines, channels, select statements, context, worker pools, mutexes, and race condition prevention.

## Concurrency vs Parallelism

### Understanding the Difference

- **Concurrency**: Dealing with multiple things at once (structure/design)
- **Parallelism**: Doing multiple things at once (execution)

Go provides concurrency primitives (goroutines, channels) that can run in parallel on multiple CPU cores.

```go
// Concurrency: Structure allows concurrent execution
go processWorkflow(workflow1)
go processWorkflow(workflow2)
go processWorkflow(workflow3)

// Parallelism: Actually running on multiple cores (if available)
// Go runtime schedules goroutines across available CPU cores
```

## Goroutines

### Basic Usage

```go
// Launch goroutine
go func() {
    processWorkflow(workflow)
}()

// Function call in goroutine
go processWorkflow(workflow)
```

### Waiting for Completion

```go
// Use sync.WaitGroup
var wg sync.WaitGroup

for _, workflow := range workflows {
    wg.Add(1)
    go func(w *Workflow) {
        defer wg.Done()
        processWorkflow(w)
    }(workflow)
}

wg.Wait()  // Wait for all goroutines to complete
```

### Goroutine Best Practices

- Always use `defer` for cleanup
- Don't leak goroutines (ensure they complete)
- Use context for cancellation
- Be aware of goroutine lifecycle

```go
// Good: Proper cleanup
go func() {
    defer cleanup()
    processWorkflow(workflow)
}()

// Bad: Goroutine leak (no way to stop)
go func() {
    for {
        processWorkflow(workflow)
        time.Sleep(1 * time.Second)
    }
}()
```

## Channels

### Channel Types

```go
// Unbuffered channel (synchronous)
ch := make(chan int)

// Buffered channel (asynchronous up to buffer size)
ch := make(chan int, 10)

// Send-only channel
var sendCh chan<- int

// Receive-only channel
var recvCh <-chan int
```

### Channel Operations

```go
// Send
ch <- value

// Receive
value := <-ch

// Receive with ok (check if channel closed)
value, ok := <-ch
if !ok {
    // Channel closed
}

// Close channel (only sender should close)
close(ch)

// Range over channel
for value := range ch {
    // Process value
}
```

### Channel Patterns

#### Fan-Out (Distribute Work)

```go
func fanOut(input <-chan Workflow, workers int) []<-chan Workflow {
    outputs := make([]<-chan Workflow, workers)
    
    for i := 0; i < workers; i++ {
        output := make(chan Workflow)
        outputs[i] = output
        
        go func(out chan<- Workflow) {
            defer close(out)
            for workflow := range input {
                // Process workflow
                processed := processWorkflow(workflow)
                out <- processed
            }
        }(output)
    }
    
    return outputs
}
```

#### Fan-In (Collect Results)

```go
func fanIn(inputs ...<-chan Workflow) <-chan Workflow {
    output := make(chan Workflow)
    
    var wg sync.WaitGroup
    wg.Add(len(inputs))
    
    for _, input := range inputs {
        go func(in <-chan Workflow) {
            defer wg.Done()
            for workflow := range in {
                output <- workflow
            }
        }(input)
    }
    
    go func() {
        wg.Wait()
        close(output)
    }()
    
    return output
}
```

#### Pipeline Pattern

```go
func pipeline(input <-chan Workflow) <-chan Result {
    // Stage 1: Validate
    validated := make(chan Workflow)
    go func() {
        defer close(validated)
        for w := range input {
            if validateWorkflow(w) {
                validated <- w
            }
        }
    }()
    
    // Stage 2: Process
    processed := make(chan Result)
    go func() {
        defer close(processed)
        for w := range validated {
            result := processWorkflow(w)
            processed <- result
        }
    }()
    
    return processed
}
```

## Select Statement

### Basic Select

```go
select {
case msg := <-ch1:
    // Handle message from ch1
case msg := <-ch2:
    // Handle message from ch2
case <-time.After(5 * time.Second):
    // Timeout
default:
    // Non-blocking (if no case ready)
}
```

### Select with Timeout

```go
select {
case result := <-resultCh:
    // Process result
case <-time.After(10 * time.Second):
    return fmt.Errorf("operation timed out")
}
```

### Select for Cancellation

```go
select {
case result := <-resultCh:
    // Process result
case <-ctx.Done():
    return ctx.Err()
}
```

### Non-Blocking Select

```go
select {
case msg := <-ch:
    // Process message
default:
    // No message available, continue
}
```

## Context for Cancellation

### Context Basics

```go
// Create context with cancellation
ctx, cancel := context.WithCancel(context.Background())
defer cancel()  // Always call cancel to release resources

// Create context with timeout
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

// Create context with deadline
deadline := time.Now().Add(10 * time.Second)
ctx, cancel := context.WithDeadline(context.Background(), deadline)
defer cancel()
```

### Using Context

```go
func processWorkflow(ctx context.Context, workflow *Workflow) error {
    // Check if context cancelled
    if err := ctx.Err(); err != nil {
        return err
    }
    
    // Pass context to operations
    result, err := executeNode(ctx, workflow.TriggerNode())
    if err != nil {
        return err
    }
    
    // Use context in select
    select {
    case <-ctx.Done():
        return ctx.Err()
    case result := <-resultCh:
        return processResult(result)
    }
}
```

### Context in HTTP Handlers

```go
func (h *Handler) HandleGet(ctx context.Context, w http.ResponseWriter, r *http.Request) {
    // Use request context
    ctx := r.Context()
    
    // Pass context to service
    workflow, err := h.service.GetWorkflow(ctx, id)
    if err != nil {
        if err == context.DeadlineExceeded {
            http.Error(w, "Request timeout", http.StatusRequestTimeout)
            return
        }
        http.Error(w, "Internal error", http.StatusInternalServerError)
        return
    }
    
    // Return response
}
```

## Worker Pools

### Basic Worker Pool

```go
type WorkerPool struct {
    workers    int
    jobQueue   chan Job
    resultQueue chan Result
}

func NewWorkerPool(workers int) *WorkerPool {
    return &WorkerPool{
        workers:     workers,
        jobQueue:    make(chan Job, 100),
        resultQueue: make(chan Result, 100),
    }
}

func (p *WorkerPool) Start() {
    for i := 0; i < p.workers; i++ {
        go p.worker(i)
    }
}

func (p *WorkerPool) worker(id int) {
    for job := range p.jobQueue {
        result := processJob(job)
        p.resultQueue <- result
    }
}

func (p *WorkerPool) Submit(job Job) {
    p.jobQueue <- job
}

func (p *WorkerPool) Results() <-chan Result {
    return p.resultQueue
}
```

### Worker Pool with Context

```go
func (p *WorkerPool) Start(ctx context.Context) {
    for i := 0; i < p.workers; i++ {
        go p.worker(ctx, i)
    }
}

func (p *WorkerPool) worker(ctx context.Context, id int) {
    for {
        select {
        case <-ctx.Done():
            return
        case job := <-p.jobQueue:
            result := processJob(job)
            select {
            case p.resultQueue <- result:
            case <-ctx.Done():
                return
            }
        }
    }
}
```

## Mutexes and RWMutexes

### Mutex (Mutual Exclusion)

```go
type SafeCounter struct {
    mu    sync.Mutex
    value int
}

func (c *SafeCounter) Increment() {
    c.mu.Lock()
    defer c.mu.Unlock()
    c.value++
}

func (c *SafeCounter) Value() int {
    c.mu.Lock()
    defer c.mu.Unlock()
    return c.value
}
```

### RWMutex (Read-Write Mutex)

```go
type SafeMap struct {
    mu   sync.RWMutex
    data map[string]interface{}
}

// Read operation (multiple readers allowed)
func (m *SafeMap) Get(key string) (interface{}, bool) {
    m.mu.RLock()
    defer m.mu.RUnlock()
    value, ok := m.data[key]
    return value, ok
}

// Write operation (exclusive access)
func (m *SafeMap) Set(key string, value interface{}) {
    m.mu.Lock()
    defer m.mu.Unlock()
    m.data[key] = value
}
```

### Mutex Best Practices

- Always use `defer` to unlock
- Keep critical sections small
- Use RWMutex for read-heavy workloads
- Don't hold locks while doing I/O
- Avoid nested locks (deadlock risk)

```go
// Good: Small critical section
func (r *Repository) Get(id string) (*Entity, error) {
    r.mu.RLock()
    entity, ok := r.entities[id]
    r.mu.RUnlock()  // Unlock before I/O
    
    if !ok {
        return nil, ErrNotFound
    }
    
    // Do I/O outside lock
    return entity.LoadDetails()
}

// Bad: Hold lock during I/O
func (r *Repository) Get(id string) (*Entity, error) {
    r.mu.RLock()
    defer r.mu.RUnlock()
    
    entity, ok := r.entities[id]
    if !ok {
        return nil, ErrNotFound
    }
    
    // I/O while holding lock - blocks other readers
    return entity.LoadDetails()
}
```

## Atomic Operations

### Atomic Counters

```go
var counter int64

// Atomic increment
atomic.AddInt64(&counter, 1)

// Atomic read
value := atomic.LoadInt64(&counter)

// Atomic compare-and-swap
swapped := atomic.CompareAndSwapInt64(&counter, old, new)
```

### When to Use Atomics

- Simple counters
- Flags
- Simple state updates
- Performance-critical code
- Avoid for complex operations (use mutex instead)

## Race Condition Detection

### Using Race Detector

```bash
# Run with race detector
go test -race ./...

# Build with race detector
go build -race ./cmd/fuse
```

### Common Race Conditions

```go
// Race condition: Unsafe concurrent access
var counter int

func increment() {
    counter++  // Race condition!
}

// Fix: Use mutex or atomic
var counter int64
var mu sync.Mutex

func increment() {
    mu.Lock()
    defer mu.Unlock()
    counter++
}

// Or use atomic
func increment() {
    atomic.AddInt64(&counter, 1)
}
```

## Deadlock Prevention

### Common Deadlock Scenarios

```go
// Deadlock: Lock ordering
func transfer(from, to *Account, amount int) {
    from.mu.Lock()
    to.mu.Lock()  // Deadlock if another goroutine locks in reverse order
    
    from.balance -= amount
    to.balance += amount
    
    to.mu.Unlock()
    from.mu.Unlock()
}

// Fix: Consistent lock ordering
func transfer(from, to *Account, amount int) {
    // Always lock in same order (e.g., by ID)
    first, second := from, to
    if from.id > to.id {
        first, second = to, from
    }
    
    first.mu.Lock()
    second.mu.Lock()
    defer second.mu.Unlock()
    defer first.mu.Unlock()
    
    from.balance -= amount
    to.balance += amount
}
```

### Deadlock Detection

- Use `go tool pprof` to detect deadlocks
- Use timeouts to detect hanging goroutines
- Monitor goroutine counts
- Use context cancellation

## Best Practices

1. **Use channels for communication** - Share by communicating
2. **Avoid shared mutable state** - Use channels or mutexes
3. **Use context for cancellation** - Proper cleanup
4. **Don't leak goroutines** - Ensure they complete
5. **Use RWMutex for read-heavy** - Better performance
6. **Keep critical sections small** - Minimize lock time
7. **Use defer to unlock** - Always unlock, even on panic
8. **Detect races** - Use `-race` flag
9. **Avoid deadlocks** - Consistent lock ordering
10. **Profile before optimizing** - Measure, don't guess

## References

- [Go Concurrency Patterns](https://go.dev/blog/pipelines)
- [Effective Go - Concurrency](https://go.dev/doc/effective_go#concurrency)
- [Go Memory Model](https://go.dev/ref/mem)
