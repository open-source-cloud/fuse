package actors

import (
	"time"

	"ergo.services/ergo/gen"
	"github.com/open-source-cloud/fuse/internal/handlers"
)

type (
	// Factories is a map of factories
	Factories map[string]gen.ProcessFactory
	// Workers is a map of workers
	Workers struct {
		workers   []WebWorker
		factories Factories
	}
	// WorkerPoolConfig is the configuration for a worker pool
	WorkerPoolConfig struct {
		Name     gen.Atom
		PoolSize int64
	}
	// WebWorker is a worker
	WebWorker struct {
		Name       gen.Atom
		Pattern    string
		Methods    []string
		Timeout    time.Duration
		PoolConfig WorkerPoolConfig
	}
)

// NewWorkers creates a new Workers
func NewWorkers() *Workers {
	return &Workers{
		workers: []WebWorker{
			{
				Name:    handlers.HealthCheckHandlerName,
				Pattern: "/health",
				Methods: []string{"GET"},
				Timeout: 10 * time.Second,
				PoolConfig: WorkerPoolConfig{
					Name:     handlers.HealthCheckHandlerPoolName,
					PoolSize: 3,
				},
			},
			{
				Name:    handlers.TriggerWorkflowHandlerName,
				Pattern: "/v1/workflows/trigger",
				Methods: []string{"POST"},
				Timeout: 10 * time.Second,
				PoolConfig: WorkerPoolConfig{
					Name:     handlers.TriggerWorkflowHandlerPoolName,
					PoolSize: 3,
				},
			},
			{
				Name:    handlers.AsyncFunctionResultHandlerName,
				Pattern: "/v1/workflows/{workflowID}/execs/{execID}",
				Methods: []string{"GET"},
				Timeout: 10 * time.Second,
				PoolConfig: WorkerPoolConfig{
					Name:     handlers.AsyncFunctionResultHandlerPoolName,
					PoolSize: 3,
				},
			},
			{
				Name:    handlers.UpsertWorkflowSchemaHandlerName,
				Pattern: "/v1/schemas/{schemaID}",
				Methods: []string{"PUT", "GET"},
				Timeout: 10 * time.Second,
				PoolConfig: WorkerPoolConfig{
					Name:     handlers.UpsertWorkflowSchemaHandlerPoolName,
					PoolSize: 3,
				},
			},
		},
		factories: make(map[string]gen.ProcessFactory),
	}
}

// AddFactory adds a factory to the factories map
func (w *Workers) AddFactory(name string, factory gen.ProcessFactory) {
	w.factories[name] = factory
}

// GetFactory gets a factory from the factories map
func (w *Workers) GetFactory(name string) (gen.ProcessFactory, bool) {
	factory, ok := w.factories[name]
	return factory, ok
}

// GetAll gets all workers
func (w *Workers) GetAll() []WebWorker {
	return w.workers
}
