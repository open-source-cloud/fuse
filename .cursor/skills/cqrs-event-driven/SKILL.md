# CQRS & Event-Driven Architecture Skill

This skill provides expertise in Command Query Responsibility Segregation (CQRS), event sourcing, event-driven architecture patterns, message brokers, and distributed transaction coordination in Go.

## Command Query Responsibility Segregation (CQRS)

### Core Concept

- Separate read and write models
- Commands change state (write)
- Queries read state (read)
- Optimize each independently
- Often used with event sourcing

### Command Side (Write Model)

- Handles commands that change state
- Validates business rules
- Publishes domain events
- Optimized for writes

```go
// Command: Changes state
type CreateWorkflowCommand struct {
    SchemaID string
    Input    map[string]any
}

type WorkflowCommandHandler struct {
    workflowRepo WorkflowRepository
    eventBus     EventBus
}

func (h *WorkflowCommandHandler) HandleCreateWorkflow(cmd CreateWorkflowCommand) error {
    // 1. Validate command
    if err := h.validateCommand(cmd); err != nil {
        return err
    }
    
    // 2. Create workflow (write model)
    workflow := NewWorkflow(cmd.SchemaID, cmd.Input)
    
    // 3. Save workflow
    if err := h.workflowRepo.Save(workflow); err != nil {
        return err
    }
    
    // 4. Publish domain event
    h.eventBus.Publish(WorkflowCreated{
        WorkflowID: workflow.ID(),
        CreatedAt:  time.Now(),
    })
    
    return nil
}
```

### Query Side (Read Model)

- Handles queries that read state
- Optimized for reads
- Denormalized data
- Updated asynchronously from events

```go
// Query: Reads state
type GetWorkflowQuery struct {
    WorkflowID WorkflowID
}

type WorkflowQueryHandler struct {
    readRepo WorkflowReadRepository  // Read-optimized repository
}

func (h *WorkflowQueryHandler) HandleGetWorkflow(query GetWorkflowQuery) (*WorkflowReadModel, error) {
    // Read from optimized read model
    return h.readRepo.Get(query.WorkflowID)
}

// Read Model (denormalized)
type WorkflowReadModel struct {
    ID          WorkflowID
    Status      string
    NodeCount   int
    EdgeCount   int
    CreatedAt   time.Time
    UpdatedAt   time.Time
    // Denormalized fields for fast reads
}
```

### Benefits of CQRS

- Independent scaling of read/write
- Optimize each side independently
- Simpler models (no read/write conflicts)
- Better performance for high-traffic systems
- Enables event sourcing

## Event Sourcing

### Core Concept

- Store events instead of current state
- Replay events to rebuild state
- Complete audit trail
- Time travel debugging

### Event Store

```go
// Event Store interface
type EventStore interface {
    Append(streamID string, events []DomainEvent, expectedVersion int) error
    GetEvents(streamID string, fromVersion int) ([]DomainEvent, error)
    GetStreamVersion(streamID string) (int, error)
}

// Domain Event
type DomainEvent interface {
    EventType() string
    OccurredAt() time.Time
}

// Workflow Events
type WorkflowCreated struct {
    WorkflowID WorkflowID
    SchemaID   string
    CreatedAt  time.Time
}

func (e WorkflowCreated) EventType() string {
    return "WorkflowCreated"
}

func (e WorkflowCreated) OccurredAt() time.Time {
    return e.CreatedAt
}

type WorkflowStarted struct {
    WorkflowID WorkflowID
    StartedAt  time.Time
}

func (e WorkflowStarted) EventType() string {
    return "WorkflowStarted"
}

func (e WorkflowStarted) OccurredAt() time.Time {
    return e.StartedAt
}
```

### Aggregate with Event Sourcing

```go
// Aggregate that uses event sourcing
type Workflow struct {
    id      WorkflowID
    state   WorkflowState
    version int  // For optimistic concurrency
    events  []DomainEvent  // Uncommitted events
}

func (w *Workflow) Start() {
    if w.state != WorkflowStatePending {
        panic("invalid state")
    }
    
    w.state = WorkflowStateRunning
    w.events = append(w.events, WorkflowStarted{
        WorkflowID: w.id,
        StartedAt:  time.Now(),
    })
}

// Rebuild aggregate from events
func RebuildWorkflowFromEvents(id WorkflowID, events []DomainEvent) *Workflow {
    workflow := &Workflow{id: id}
    
    for _, event := range events {
        workflow.ApplyEvent(event)
    }
    
    return workflow
}

func (w *Workflow) ApplyEvent(event DomainEvent) {
    switch e := event.(type) {
    case WorkflowCreated:
        w.state = WorkflowStatePending
    case WorkflowStarted:
        w.state = WorkflowStateRunning
    case WorkflowCompleted:
        w.state = WorkflowStateCompleted
    }
    w.version++
}
```

### Event Store Implementation

```go
type MongoEventStore struct {
    collection *mongo.Collection
}

func (s *MongoEventStore) Append(streamID string, events []DomainEvent, expectedVersion int) error {
    // Optimistic concurrency check
    currentVersion, err := s.GetStreamVersion(streamID)
    if err != nil && err != ErrStreamNotFound {
        return err
    }
    
    if currentVersion != expectedVersion {
        return ErrConcurrencyConflict
    }
    
    // Append events
    for i, event := range events {
        doc := bson.M{
            "streamId":   streamID,
            "version":    expectedVersion + i + 1,
            "eventType":  event.EventType(),
            "eventData": event,
            "occurredAt": event.OccurredAt(),
        }
        
        if _, err := s.collection.InsertOne(context.Background(), doc); err != nil {
            return err
        }
    }
    
    return nil
}
```

## Event-Driven Architecture Patterns

### Event Notification

- Notify other services of events
- Minimal data in event
- Recipients fetch details if needed

```go
type WorkflowCompletedEvent struct {
    WorkflowID WorkflowID
    CompletedAt time.Time
    // Minimal data - recipients fetch details if needed
}
```

### Event Carrying State Transfer

- Include data in event
- Recipients don't need to fetch details
- Larger events, but fewer calls

```go
type WorkflowCompletedEvent struct {
    WorkflowID  WorkflowID
    CompletedAt time.Time
    Result      map[string]any  // Full result data
    NodeResults []NodeResult    // All node results
}
```

### Event Sourcing Pattern

- Events are source of truth
- Replay events to rebuild state
- Complete audit trail
- Time travel debugging

### Saga Pattern

- Coordinates distributed transactions
- Each service performs local transaction
- Compensating actions for rollback
- Eventual consistency

```go
// Saga Orchestrator
type WorkflowExecutionSaga struct {
    steps []SagaStep
}

type SagaStep struct {
    Name       string
    Execute    func() error
    Compensate func() error
}

func (s *WorkflowExecutionSaga) Execute() error {
    completedSteps := []SagaStep{}
    
    for _, step := range s.steps {
        if err := step.Execute(); err != nil {
            // Compensate completed steps
            for i := len(completedSteps) - 1; i >= 0; i-- {
                if compErr := completedSteps[i].Compensate(); compErr != nil {
                    // Log compensation error
                }
            }
            return err
        }
        completedSteps = append(completedSteps, step)
    }
    
    return nil
}
```

## Message Brokers and Event Streams

### RabbitMQ

- Traditional message broker
- Supports multiple messaging patterns
- Durable queues
- Message acknowledgments

```go
// RabbitMQ publisher
type RabbitMQPublisher struct {
    conn *amqp.Connection
    ch   *amqp.Channel
}

func (p *RabbitMQPublisher) Publish(exchange, routingKey string, event DomainEvent) error {
    body, err := json.Marshal(event)
    if err != nil {
        return err
    }
    
    return p.ch.Publish(
        exchange,
        routingKey,
        false,  // mandatory
        false,  // immediate
        amqp.Publishing{
            ContentType: "application/json",
            Body:        body,
        },
    )
}

// RabbitMQ consumer
func (c *RabbitMQConsumer) Consume(queue string, handler EventHandler) error {
    msgs, err := c.ch.Consume(
        queue,
        "",    // consumer
        false, // auto-ack
        false, // exclusive
        false, // no-local
        false, // no-wait
        nil,   // args
    )
    
    for msg := range msgs {
        var event DomainEvent
        if err := json.Unmarshal(msg.Body, &event); err != nil {
            msg.Nack(false, false)
            continue
        }
        
        if err := handler.Handle(event); err != nil {
            msg.Nack(false, true)  // Requeue on error
            continue
        }
        
        msg.Ack(false)
    }
    
    return nil
}
```

### Apache Kafka

- Distributed event streaming platform
- High throughput
- Event retention
- Consumer groups

```go
// Kafka producer
type KafkaProducer struct {
    producer *kafka.Producer
}

func (p *KafkaProducer) Publish(topic string, event DomainEvent) error {
    value, err := json.Marshal(event)
    if err != nil {
        return err
    }
    
    msg := &kafka.Message{
        TopicPartition: kafka.TopicPartition{Topic: &topic, Partition: kafka.PartitionAny},
        Value:          value,
        Headers: []kafka.Header{
            {Key: "eventType", Value: []byte(event.EventType())},
        },
    }
    
    return p.producer.Produce(msg, nil)
}

// Kafka consumer
func (c *KafkaConsumer) Consume(topic string, handler EventHandler) error {
    c.consumer.SubscribeTopics([]string{topic}, nil)
    
    for {
        msg, err := c.consumer.ReadMessage(-1)
        if err != nil {
            return err
        }
        
        var event DomainEvent
        if err := json.Unmarshal(msg.Value, &event); err != nil {
            continue
        }
        
        if err := handler.Handle(event); err != nil {
            // Handle error (log, dead letter queue, etc.)
        }
    }
}
```

### AWS SQS/SNS

- Managed message queues
- Serverless-friendly
- Auto-scaling
- Dead letter queues

```go
// SQS publisher
type SQSPublisher struct {
    client *sqs.SQS
    queueURL string
}

func (p *SQSPublisher) Publish(event DomainEvent) error {
    body, err := json.Marshal(event)
    if err != nil {
        return err
    }
    
    _, err = p.client.SendMessage(&sqs.SendMessageInput{
        QueueUrl:    &p.queueURL,
        MessageBody: aws.String(string(body)),
        MessageAttributes: map[string]*sqs.MessageAttributeValue{
            "EventType": {
                DataType:    aws.String("String"),
                StringValue: aws.String(event.EventType()),
            },
        },
    })
    
    return err
}
```

## Eventual Consistency

### Accepting Temporary Inconsistency

- Read models updated asynchronously
- Temporary inconsistency is acceptable
- Eventually consistent
- Strong consistency where needed

### Read Model Updates

```go
// Event handler updates read model
type WorkflowReadModelUpdater struct {
    readRepo WorkflowReadRepository
}

func (u *WorkflowReadModelUpdater) HandleWorkflowCreated(event WorkflowCreated) error {
    readModel := &WorkflowReadModel{
        ID:        event.WorkflowID,
        Status:    "pending",
        CreatedAt: event.CreatedAt,
    }
    return u.readRepo.Save(readModel)
}

func (u *WorkflowReadModelUpdater) HandleWorkflowStarted(event WorkflowStarted) error {
    readModel, err := u.readRepo.Get(event.WorkflowID)
    if err != nil {
        return err
    }
    
    readModel.Status = "running"
    readModel.UpdatedAt = event.StartedAt
    return u.readRepo.Save(readModel)
}
```

## Go Implementation Patterns

### Event Bus

```go
type EventBus interface {
    Publish(event DomainEvent) error
    Subscribe(eventType string, handler EventHandler) error
}

type InMemoryEventBus struct {
    handlers map[string][]EventHandler
    mu       sync.RWMutex
}

func (b *InMemoryEventBus) Publish(event DomainEvent) error {
    b.mu.RLock()
    handlers := b.handlers[event.EventType()]
    b.mu.RUnlock()
    
    for _, handler := range handlers {
        if err := handler.Handle(event); err != nil {
            // Log error, continue with other handlers
        }
    }
    
    return nil
}

func (b *InMemoryEventBus) Subscribe(eventType string, handler EventHandler) error {
    b.mu.Lock()
    defer b.mu.Unlock()
    
    b.handlers[eventType] = append(b.handlers[eventType], handler)
    return nil
}
```

### Event Handler Interface

```go
type EventHandler interface {
    Handle(event DomainEvent) error
}

// Multiple handlers can handle same event
type WorkflowMetricsHandler struct {
    metrics MetricsCollector
}

func (h *WorkflowMetricsHandler) Handle(event DomainEvent) error {
    switch e := event.(type) {
    case WorkflowCreated:
        h.metrics.IncrementCounter("workflows.created")
    case WorkflowCompleted:
        h.metrics.IncrementCounter("workflows.completed")
        h.metrics.RecordDuration("workflow.duration", e.Duration)
    }
    return nil
}
```

## Best Practices

1. **Separate read/write models** - Optimize independently
2. **Use event sourcing** - Complete audit trail
3. **Publish domain events** - Cross-aggregate communication
4. **Handle eventual consistency** - Accept temporary inconsistency
5. **Use message brokers** - Reliable event delivery
6. **Implement sagas** - Coordinate distributed transactions
7. **Idempotent handlers** - Handle duplicate events
8. **Event versioning** - Support schema evolution
9. **Dead letter queues** - Handle failed events
10. **Monitor event flow** - Observability is critical

## References

- [CQRS Pattern](https://martinfowler.com/bliki/CQRS.html)
- [Event Sourcing](https://martinfowler.com/eaaDev/EventSourcing.html)
- [Saga Pattern](https://microservices.io/patterns/data/saga.html)
