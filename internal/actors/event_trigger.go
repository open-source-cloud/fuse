package actors

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"time"

	"ergo.services/ergo/act"
	"ergo.services/ergo/gen"
	"github.com/expr-lang/expr"
	"github.com/open-source-cloud/fuse/internal/actors/actornames"
	"github.com/open-source-cloud/fuse/internal/events"
	"github.com/open-source-cloud/fuse/internal/idempotency"
	"github.com/open-source-cloud/fuse/internal/messaging"
	"github.com/open-source-cloud/fuse/internal/services"
	internalworkflow "github.com/open-source-cloud/fuse/internal/workflow"
	"github.com/open-source-cloud/fuse/pkg/workflow"
)

// eventIdempotencyTTL is the TTL for event trigger idempotency keys.
const eventIdempotencyTTL = 10 * time.Minute

// EventTriggerFactory is a factory for creating EventTrigger actors
type EventTriggerFactory ActorFactory[*EventTrigger]

// NewEventTriggerFactory creates a new EventTriggerFactory
func NewEventTriggerFactory(graphService services.GraphService, eventBus events.EventBus, idempotencyStore idempotency.Store) *EventTriggerFactory {
	return &EventTriggerFactory{
		Factory: func() gen.ProcessBehavior {
			return &EventTrigger{
				graphService:     graphService,
				eventBus:         eventBus,
				idempotencyStore: idempotencyStore,
				subscriptions:    make([]events.SubscriptionID, 0),
			}
		},
	}
}

// EventTrigger subscribes to internal events and triggers matching workflows.
// In HA mode, all nodes subscribe to the same events (published locally).
// Deduplication via idempotency keys prevents duplicate triggers when
// the same event is processed by multiple nodes.
type EventTrigger struct {
	act.Actor

	graphService     services.GraphService
	eventBus         events.EventBus
	idempotencyStore idempotency.Store
	subscriptions    []events.SubscriptionID
}

// Init loads all event-triggered schemas and subscribes to matching events
func (a *EventTrigger) Init(_ ...any) error {
	a.Log().Info("starting event trigger %s", a.PID())

	schemas, err := a.graphService.ListSchemas()
	if err != nil {
		a.Log().Error("failed to list schemas for event trigger: %s", err)
		return nil
	}

	for _, item := range schemas {
		graph, gErr := a.graphService.FindByID(item.SchemaID)
		if gErr != nil {
			continue
		}
		tc := graph.Schema().TriggerConfig
		if tc == nil || tc.Type != internalworkflow.TriggerEvent || tc.Event == nil {
			continue
		}
		a.subscribeEventTrigger(graph.Schema().ID, tc.Event)
	}

	a.Log().Info("event trigger started with %d subscriptions", len(a.subscriptions))
	return nil
}

func (a *EventTrigger) subscribeEventTrigger(schemaID string, cfg *internalworkflow.EventConfig) {
	subID, err := a.eventBus.Subscribe(cfg.EventType, func(event events.Event) error {
		// Apply optional filter expression
		if cfg.Filter != "" {
			matches, evalErr := evaluateFilter(cfg.Filter, event.Data)
			if evalErr != nil {
				a.Log().Warning("event filter evaluation failed for schema %s: %s", schemaID, evalErr)
				return nil
			}
			if !matches {
				return nil
			}
		}

		// Build deterministic idempotency key from event source + type + data hash
		idempotencyKey := buildEventIdempotencyKey(schemaID, event)
		workflowID := workflow.NewID()

		if existingID, existed := a.idempotencyStore.CheckAndSet(idempotencyKey, workflowID.String(), eventIdempotencyTTL); existed {
			a.Log().Debug("event trigger for schema %s already claimed by workflow %s, skipping", schemaID, existingID)
			return nil
		}

		triggerMsg := messaging.NewTriggerWorkflowWithInputMessage(schemaID, workflowID, event.Data)
		if sendErr := a.Send(gen.Atom(actornames.WorkflowSupervisorName), triggerMsg); sendErr != nil {
			a.Log().Error("event trigger failed to send for schema %s: %s", schemaID, sendErr)
		} else {
			a.Log().Info("event %s triggered workflow %s for schema %s", event.Type, workflowID, schemaID)
		}
		return nil
	})
	if err != nil {
		a.Log().Error("failed to subscribe to event %s for schema %s: %s", cfg.EventType, schemaID, err)
		return
	}
	a.subscriptions = append(a.subscriptions, subID)
}

// buildEventIdempotencyKey creates a deterministic key from event properties.
// Uses the event source (workflowID that emitted it) + type + schema target.
func buildEventIdempotencyKey(schemaID string, event events.Event) string {
	// Hash the event data to handle varying payloads
	dataJSON, _ := json.Marshal(event.Data)
	dataHash := fmt.Sprintf("%x", sha256.Sum256(dataJSON))[:16]
	return fmt.Sprintf("evt:%s:%s:%s:%s", schemaID, event.Type, event.Source, dataHash)
}

func evaluateFilter(filter string, data map[string]any) (bool, error) {
	program, err := expr.Compile(filter, expr.Env(data), expr.AsBool())
	if err != nil {
		return false, err
	}
	result, err := expr.Run(program, data)
	if err != nil {
		return false, err
	}
	return result.(bool), nil
}

// HandleMessage handles messages sent to the EventTrigger
func (a *EventTrigger) HandleMessage(_ gen.PID, _ any) error {
	return nil
}

// Terminate unsubscribes from all events
func (a *EventTrigger) Terminate(reason error) {
	for _, subID := range a.subscriptions {
		_ = a.eventBus.Unsubscribe(subID)
	}
	a.Log().Info("event trigger terminated: %s", reason)
}
