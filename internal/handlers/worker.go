package handlers

import (
	"time"

	"ergo.services/ergo/gen"
)

type (
	// Factories is a map of factories
	Factories map[string]gen.ProcessFactory
	// Workers is a map of workers
	Workers struct {
		WebWorkers []WebWorker
		Factories  Factories
	}
	// WorkerPoolConfig is the configuration for a worker pool
	WorkerPoolConfig struct {
		Name     gen.Atom
		PoolSize int64
	}
	// Worker is a worker
	WebWorker struct {
		Name       gen.Atom
		Pattern    string
		Timeout    time.Duration
		PoolConfig WorkerPoolConfig
	}
)

// NewWorkers creates a new Workers
func NewWorkers() *Workers {
	return &Workers{
		WebWorkers: []WebWorker{
			{
				Name:    HealthCheckHandlerName,
				Pattern: "/health",
				Timeout: 10 * time.Second,
				PoolConfig: WorkerPoolConfig{
					Name:     HealthCheckHandlerPoolName,
					PoolSize: 3,
				},
			},
			{
				Name:    TriggerWorkflowHandlerName,
				Pattern: "/v1/workflows/{schemaID}/trigger",
				Timeout: 10 * time.Second,
				PoolConfig: WorkerPoolConfig{
					Name:     TriggerWorkflowHandlerPoolName,
					PoolSize: 3,
				},
			},
			{
				Name:    AsyncFunctionResultHandlerName,
				Pattern: "/v1/workflows/{workflowID}/execs",
				Timeout: 10 * time.Second,
				PoolConfig: WorkerPoolConfig{
					Name:     AsyncFunctionResultHandlerPoolName,
					PoolSize: 3,
				},
			},
			{
				Name:    UpsertWorkflowSchemaHandlerName,
				Pattern: "/v1/schemas/{schemaID}",
				Timeout: 10 * time.Second,
				PoolConfig: WorkerPoolConfig{
					Name:     UpsertWorkflowSchemaHandlerPoolName,
					PoolSize: 3,
				},
			},
		},
		Factories: make(map[string]gen.ProcessFactory),
	}
}

// Add adds a factory to the factories map
func (f *Factories) Add(name string, factory gen.ProcessFactory) {
	(*f)[name] = factory
}
