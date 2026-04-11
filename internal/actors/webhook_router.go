package actors

import (
	"ergo.services/ergo/act"
	"ergo.services/ergo/gen"
	"github.com/open-source-cloud/fuse/internal/services"
	internalworkflow "github.com/open-source-cloud/fuse/internal/workflow"
)

// WebhookRouterFactory is a factory for creating WebhookRouter actors
type WebhookRouterFactory ActorFactory[*WebhookRouter]

// NewWebhookRouterFactory creates a new WebhookRouterFactory
func NewWebhookRouterFactory(graphService services.GraphService) *WebhookRouterFactory {
	return &WebhookRouterFactory{
		Factory: func() gen.ProcessBehavior {
			return &WebhookRouter{
				graphService: graphService,
				routes:       make(map[string]string),
			}
		},
	}
}

// WebhookRouter registers and routes incoming webhooks to the correct workflow schema
type WebhookRouter struct {
	act.Actor

	graphService services.GraphService
	routes       map[string]string // path -> schemaID
}

// Init loads all schemas with webhook triggers and registers their paths
func (a *WebhookRouter) Init(_ ...any) error {
	a.Log().Info("starting webhook router %s", a.PID())

	schemas, err := a.graphService.ListSchemas()
	if err != nil {
		a.Log().Error("failed to list schemas for webhook router: %s", err)
		return nil
	}

	for _, item := range schemas {
		graph, gErr := a.graphService.FindByID(item.SchemaID)
		if gErr != nil {
			continue
		}
		tc := graph.Schema().TriggerConfig
		if tc == nil || tc.Type != internalworkflow.TriggerWebhook || tc.Webhook == nil {
			continue
		}
		a.routes[tc.Webhook.Path] = graph.Schema().ID
		a.Log().Info("registered webhook route %s -> schema %s", tc.Webhook.Path, graph.Schema().ID)
	}

	a.Log().Info("webhook router started with %d routes", len(a.routes))
	return nil
}

// ResolveSchemaID returns the schema ID for the given webhook path, if registered
func (a *WebhookRouter) ResolveSchemaID(path string) (string, bool) {
	schemaID, exists := a.routes[path]
	return schemaID, exists
}

// HandleMessage handles messages sent to the WebhookRouter
func (a *WebhookRouter) HandleMessage(_ gen.PID, _ any) error {
	// Future: handle schema upsert messages to dynamically update routes
	return nil
}

// Terminate cleans up the webhook router
func (a *WebhookRouter) Terminate(reason error) {
	a.Log().Info("webhook router terminated: %s", reason)
}
