# Microservices Architecture Skill

This skill provides expertise in designing and implementing microservices architectures, including service boundaries, API design, service communication, fault tolerance, and distributed systems patterns.

## Service Boundaries and Autonomy

### Service Independence

- Each service is independently deployable
- Services own their data (database per service)
- Services communicate through well-defined APIs
- Services can be developed and scaled independently

### Bounded Contexts

- Each service represents a bounded context (DDD)
- Clear domain boundaries prevent coupling
- Services share nothing except APIs
- Data consistency through eventual consistency

### Service Decomposition

- Decompose by business capability
- Decompose by subdomain (DDD)
- Avoid decomposition by technical layers
- Keep services cohesive and loosely coupled

## API Design

### RESTful APIs

- Use standard HTTP methods (GET, POST, PUT, DELETE)
- Resource-based URLs
- Proper HTTP status codes
- Version APIs (e.g., `/v1/workflows`)
- Use JSON for request/response bodies

```go
// Good RESTful design
GET    /v1/workflows/{id}        // Get workflow
POST   /v1/workflows             // Create workflow
PUT    /v1/workflows/{id}        // Update workflow
DELETE /v1/workflows/{id}        // Delete workflow
```

### gRPC APIs

- Use Protocol Buffers for schema definition
- Type-safe, efficient binary protocol
- Streaming support (unary, server streaming, client streaming, bidirectional)
- Better performance than REST for inter-service communication

```protobuf
service WorkflowService {
    rpc GetWorkflow(GetWorkflowRequest) returns (Workflow);
    rpc CreateWorkflow(CreateWorkflowRequest) returns (Workflow);
    rpc StreamWorkflows(StreamRequest) returns (stream Workflow);
}
```

### API Versioning

- Version APIs from the start
- Use URL versioning: `/v1/`, `/v2/`
- Maintain backward compatibility when possible
- Deprecate old versions gracefully

### API Documentation

- Use OpenAPI/Swagger for REST APIs
- Document all endpoints, request/response schemas
- Provide examples
- Keep documentation up to date

## Service Discovery and Communication

### Service Discovery

- Services need to find each other
- Options: DNS-based, service registry (Consul, etcd), client-side discovery, server-side discovery
- Health checks for service availability
- Load balancing across service instances

### Synchronous Communication

- HTTP/REST for request-response
- gRPC for high-performance RPC
- Use timeouts and circuit breakers
- Handle partial failures gracefully

```go
// HTTP client with timeout and retry
client := &http.Client{
    Timeout: 5 * time.Second,
}

// Use circuit breaker pattern
breaker := gobreaker.NewCircuitBreaker(gobreaker.Settings{
    MaxRequests: 3,
    Interval:    60 * time.Second,
    Timeout:     30 * time.Second,
})
```

### Asynchronous Communication

- Message queues (RabbitMQ, Kafka, SQS)
- Event-driven architecture
- Publish-subscribe patterns
- Event sourcing for audit trails

## Distributed Systems Patterns

### Saga Pattern

- Manages distributed transactions
- Each service performs local transaction
- Compensating actions for rollback
- Eventual consistency

```go
// Saga orchestrator coordinates distributed transaction
type SagaOrchestrator struct {
    steps []SagaStep
}

type SagaStep struct {
    Execute func() error
    Compensate func() error
}
```

### Circuit Breaker

- Prevents cascading failures
- Opens circuit after failure threshold
- Half-open state for recovery testing
- Closes circuit when healthy

### Bulkhead Pattern

- Isolate resources to prevent cascading failures
- Separate thread pools per service
- Isolate database connections
- Prevent one service failure from affecting others

### Retry Pattern

- Retry transient failures
- Exponential backoff
- Maximum retry attempts
- Idempotent operations

```go
func retryWithBackoff(fn func() error, maxRetries int) error {
    for i := 0; i < maxRetries; i++ {
        err := fn()
        if err == nil {
            return nil
        }
        time.Sleep(time.Duration(i+1) * time.Second)
    }
    return fmt.Errorf("max retries exceeded")
}
```

## Fault Tolerance and Resilience

### Timeouts

- Set timeouts for all external calls
- Use context.Context for cancellation
- Prevent hanging requests

```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

resp, err := client.Do(req.WithContext(ctx))
```

### Retries

- Retry transient failures
- Exponential backoff
- Jitter to prevent thundering herd
- Idempotent operations

### Health Checks

- Implement health check endpoints
- Check dependencies (database, external services)
- Return service status
- Use for load balancer health checks

```go
func (h *Handler) HandleGetHealth(w http.ResponseWriter, r *http.Request) {
    health := HealthStatus{
        Status: "healthy",
        Checks: map[string]string{
            "database": checkDatabase(),
            "cache": checkCache(),
        },
    }
    // Return health status
}
```

### Graceful Degradation

- Provide fallback behavior
- Cache responses when possible
- Return partial data if needed
- Log degradation events

## Service Mesh Concepts

### Service Mesh Benefits

- Traffic management (routing, load balancing)
- Security (mTLS, authentication)
- Observability (metrics, tracing, logging)
- Policy enforcement

### Common Service Meshes

- Istio: Full-featured, complex
- Linkerd: Simpler, performance-focused
- Consul Connect: Integrated with Consul
- AWS App Mesh: AWS-native

## Event-Driven Communication

### Event Sourcing

- Store events instead of current state
- Replay events to rebuild state
- Audit trail built-in
- Time travel debugging

### CQRS (Command Query Responsibility Segregation)

- Separate read and write models
- Optimize each independently
- Event sourcing often used with CQRS
- Read models updated asynchronously

### Message Brokers

- RabbitMQ: Traditional message broker
- Apache Kafka: Distributed event streaming
- AWS SQS/SNS: Managed message queues
- NATS: Lightweight, high-performance

### Event Patterns

- **Event Notification**: Notify other services
- **Event Carrying State Transfer**: Include data in event
- **Event Sourcing**: Store events as source of truth
- **Saga**: Coordinate distributed transactions

## Data Consistency Patterns

### Eventual Consistency

- Accept temporary inconsistency
- Reconcile eventually
- Use for non-critical data
- Provide strong consistency where needed

### Distributed Transactions

- Two-Phase Commit (2PC): Synchronous, blocking
- Saga Pattern: Asynchronous, compensating
- Choose based on consistency requirements

### Conflict Resolution

- Last-write-wins
- Vector clocks
- CRDTs (Conflict-free Replicated Data Types)
- Application-level conflict resolution

## Best Practices

1. **Design for failure** - Services will fail
2. **Use timeouts** - Prevent hanging requests
3. **Implement circuit breakers** - Prevent cascading failures
4. **Monitor everything** - Metrics, logs, traces
5. **Version APIs** - Plan for evolution
6. **Document APIs** - OpenAPI/Swagger
7. **Test integration** - Test service interactions
8. **Use async communication** - Decouple services
9. **Implement health checks** - Enable monitoring
10. **Design for scalability** - Horizontal scaling

## References

- [Microservices Patterns](https://microservices.io/patterns/)
- [Building Microservices](https://www.oreilly.com/library/view/building-microservices/9781491950340/)
- [Distributed Systems](https://www.distributed-systems.net/)
