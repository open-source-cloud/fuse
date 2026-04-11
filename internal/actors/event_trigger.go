package actors

import (
	"ergo.services/ergo/act"
	"ergo.services/ergo/gen"
	"github.com/expr-lang/expr"
	"github.com/open-source-cloud/fuse/internal/actors/actornames"
	"github.com/open-source-cloud/fuse/internal/events"
	"github.com/open-source-cloud/fuse/internal/messaging"
	"github.com/open-source-cloud/fuse/internal/services"
	internalworkflow "github.com/open-source-cloud/fuse/internal/workflow"
	"github.com/open-source-cloud/fuse/pkg/workflow"
)

// EventTriggerFactory is a factory for creating EventTrigger actors
type EventTriggerFactory ActorFactory[*EventTrigger]

// NewEventTriggerFactory creates a new EventTriggerFactory
func NewEventTriggerFactory(graphService services.GraphService, eventBus events.EventBus) *EventTriggerFactory {
	return &EventTriggerFactory{
		Factory: func() gen.ProcessBehavior {
			return &EventTrigger{
				graphService:  graphService,
				eventBus:      eventBus,
				subscriptions: make([]events.SubscriptionID, 0),
			}
		},
	}
}

// EventTrigger subscribes to internal events and triggers matching workflows
type EventTrigger struct {
	act.Actor

	graphService  services.GraphService
	eventBus      events.EventBus
	subscriptions []events.SubscriptionID
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

		triggerMsg := messaging.NewTriggerWorkflowWithInputMessage(schemaID, workflow.NewID(), event.Data)
		if sendErr := a.Send(gen.Atom(actornames.WorkflowSupervisorName), triggerMsg); sendErr != nil {
			a.Log().Error("event trigger failed to send for schema %s: %s", schemaID, sendErr)
		} else {
			a.Log().Info("event %s triggered workflow for schema %s", event.Type, schemaID)
		}
		return nil
	})
	if err != nil {
		a.Log().Error("failed to subscribe to event %s for schema %s: %s", cfg.EventType, schemaID, err)
		return
	}
	a.subscriptions = append(a.subscriptions, subID)
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
