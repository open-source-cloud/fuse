// Package metrics provides Prometheus metrics collection for the FUSE workflow engine.
package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

// FuseMetrics holds all Fuse-specific prometheus metrics.
type FuseMetrics struct {
	// WorkflowsActive is the number of currently active (in-flight) workflows.
	WorkflowsActive prometheus.Gauge
	// WorkflowsCompleted is the total number of successfully completed workflows.
	WorkflowsCompleted prometheus.Counter
	// WorkflowsFailed is the total number of workflows that ended in an error state.
	WorkflowsFailed prometheus.Counter
	// WorkflowsCancelled is the total number of cancelled workflows.
	WorkflowsCancelled prometheus.Counter

	// NodeExecDuration records the duration of individual node (function) executions.
	// Labels: function_id, status (success|error).
	NodeExecDuration *prometheus.HistogramVec

	registry *prometheus.Registry
}

// NewFuseMetrics creates and registers all Fuse metrics in a dedicated prometheus registry.
func NewFuseMetrics() *FuseMetrics {
	reg := prometheus.NewRegistry()

	m := &FuseMetrics{
		registry: reg,

		WorkflowsActive: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: "fuse",
			Name:      "workflows_active",
			Help:      "Number of currently active (in-flight) workflow instances.",
		}),
		WorkflowsCompleted: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: "fuse",
			Name:      "workflows_completed_total",
			Help:      "Total number of workflow instances that completed successfully.",
		}),
		WorkflowsFailed: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: "fuse",
			Name:      "workflows_failed_total",
			Help:      "Total number of workflow instances that ended in an error state.",
		}),
		WorkflowsCancelled: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: "fuse",
			Name:      "workflows_cancelled_total",
			Help:      "Total number of workflow instances that were cancelled.",
		}),

		NodeExecDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: "fuse",
			Name:      "node_exec_duration_seconds",
			Help:      "Duration of individual node (function) executions in seconds.",
			Buckets:   prometheus.DefBuckets,
		}, []string{"function_id", "status"}),
	}

	reg.MustRegister(
		m.WorkflowsActive,
		m.WorkflowsCompleted,
		m.WorkflowsFailed,
		m.WorkflowsCancelled,
		m.NodeExecDuration,
	)

	return m
}

// Registry returns the underlying prometheus registry used by FuseMetrics.
func (m *FuseMetrics) Registry() *prometheus.Registry {
	return m.registry
}
