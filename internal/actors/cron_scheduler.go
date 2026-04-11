package actors

import (
	"fmt"
	"time"

	"ergo.services/ergo/act"
	"ergo.services/ergo/gen"
	"github.com/open-source-cloud/fuse/internal/actors/actornames"
	"github.com/open-source-cloud/fuse/internal/idempotency"
	"github.com/open-source-cloud/fuse/internal/messaging"
	"github.com/open-source-cloud/fuse/internal/services"
	internalworkflow "github.com/open-source-cloud/fuse/internal/workflow"
	"github.com/open-source-cloud/fuse/pkg/workflow"
	"github.com/robfig/cron/v3"
)

// cronIdempotencyTTL is the TTL for cron idempotency keys.
// Set to 1 hour — long enough to prevent duplicate triggers from multiple
// HA nodes, but short enough to not accumulate stale keys.
const cronIdempotencyTTL = 1 * time.Hour

// CronSchedulerFactory is a factory for creating CronScheduler actors
type CronSchedulerFactory ActorFactory[*CronScheduler]

// NewCronSchedulerFactory creates a new CronSchedulerFactory
func NewCronSchedulerFactory(graphService services.GraphService, idempotencyStore idempotency.Store) *CronSchedulerFactory {
	return &CronSchedulerFactory{
		Factory: func() gen.ProcessBehavior {
			return &CronScheduler{
				graphService:     graphService,
				idempotencyStore: idempotencyStore,
				entries:          make(map[string]cron.EntryID),
			}
		},
	}
}

// CronScheduler is an actor that manages cron-triggered workflows.
// In HA mode, all nodes run their own CronScheduler. Deduplication is
// handled via the idempotency store — only the first node to claim a
// time slot actually triggers the workflow.
type CronScheduler struct {
	act.Actor

	graphService     services.GraphService
	idempotencyStore idempotency.Store
	cronEngine       *cron.Cron
	entries          map[string]cron.EntryID // schemaID -> cron entry
}

// Init loads all cron-triggered schemas and starts the cron engine
func (a *CronScheduler) Init(_ ...any) error {
	a.Log().Info("starting cron scheduler %s", a.PID())
	a.cronEngine = cron.New()

	schemas, err := a.graphService.ListSchemas()
	if err != nil {
		a.Log().Error("failed to list schemas for cron scheduler: %s", err)
		return nil
	}

	for _, item := range schemas {
		graph, gErr := a.graphService.FindByID(item.SchemaID)
		if gErr != nil {
			a.Log().Warning("failed to load schema %s: %s", item.SchemaID, gErr)
			continue
		}
		tc := graph.Schema().TriggerConfig
		if tc == nil || tc.Type != internalworkflow.TriggerCron || tc.Cron == nil {
			continue
		}
		a.registerCronTrigger(graph.Schema().ID, tc.Cron)
	}

	a.cronEngine.Start()
	a.Log().Info("cron scheduler started with %d entries", len(a.entries))
	return nil
}

func (a *CronScheduler) registerCronTrigger(schemaID string, cfg *internalworkflow.CronConfig) {
	entryID, err := a.cronEngine.AddFunc(cfg.Expression, func() {
		// Build a deterministic idempotency key from schema ID + time bucket.
		// Truncate to the minute to handle small scheduling jitter across nodes.
		timeBucket := time.Now().Truncate(time.Minute).Format(time.RFC3339)
		idempotencyKey := fmt.Sprintf("cron:%s:%s", schemaID, timeBucket)
		workflowID := workflow.NewID()

		// Atomic check-and-set: only one node wins for each time bucket
		if existingID, existed := a.idempotencyStore.CheckAndSet(idempotencyKey, workflowID.String(), cronIdempotencyTTL); existed {
			a.Log().Debug("cron trigger for schema %s at %s already claimed by workflow %s, skipping", schemaID, timeBucket, existingID)
			return
		}

		triggerMsg := messaging.NewTriggerWorkflowWithInputMessage(schemaID, workflowID, cfg.Input)
		if sendErr := a.Send(gen.Atom(actornames.WorkflowSupervisorName), triggerMsg); sendErr != nil {
			a.Log().Error("cron trigger failed to send for schema %s: %s", schemaID, sendErr)
		} else {
			a.Log().Info("cron triggered workflow %s for schema %s", workflowID, schemaID)
		}
	})
	if err != nil {
		a.Log().Error("failed to register cron schedule for schema %s: %s", schemaID, err)
		return
	}
	a.entries[schemaID] = entryID
}

// HandleMessage handles messages sent to the CronScheduler
func (a *CronScheduler) HandleMessage(_ gen.PID, _ any) error {
	// Future: handle schema upsert messages to dynamically add/remove cron entries
	return nil
}

// Terminate stops the cron engine
func (a *CronScheduler) Terminate(reason error) {
	if a.cronEngine != nil {
		a.cronEngine.Stop()
	}
	a.Log().Info("cron scheduler terminated: %s", reason)
}
