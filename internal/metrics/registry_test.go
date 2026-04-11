package metrics_test

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-source-cloud/fuse/internal/metrics"
)

func TestNewFuseMetrics_registersAllMetrics(t *testing.T) {
	m := metrics.NewFuseMetrics()
	require.NotNil(t, m)
	require.NotNil(t, m.Registry())
}

func TestFuseMetrics_workflowsActive(t *testing.T) {
	m := metrics.NewFuseMetrics()

	m.WorkflowsActive.Inc()
	m.WorkflowsActive.Inc()
	m.WorkflowsActive.Dec()

	count := testutil.ToFloat64(m.WorkflowsActive)
	assert.Equal(t, 1.0, count)
}

func TestFuseMetrics_workflowCounters(t *testing.T) {
	m := metrics.NewFuseMetrics()

	m.WorkflowsCompleted.Inc()
	m.WorkflowsCompleted.Inc()
	m.WorkflowsFailed.Inc()
	m.WorkflowsCancelled.Inc()

	assert.Equal(t, 2.0, testutil.ToFloat64(m.WorkflowsCompleted))
	assert.Equal(t, 1.0, testutil.ToFloat64(m.WorkflowsFailed))
	assert.Equal(t, 1.0, testutil.ToFloat64(m.WorkflowsCancelled))
}

func TestFuseMetrics_nodeExecDuration(t *testing.T) {
	m := metrics.NewFuseMetrics()

	m.NodeExecDuration.WithLabelValues("my/pkg/my_fn", "success").Observe(0.42)
	m.NodeExecDuration.WithLabelValues("my/pkg/my_fn", "error").Observe(0.01)

	// Gather and check that the metric names are present.
	gathered, err := m.Registry().Gather()
	require.NoError(t, err)

	found := false
	for _, mf := range gathered {
		if mf.GetName() == "fuse_node_exec_duration_seconds" {
			found = true
			assert.Len(t, mf.GetMetric(), 2, "expected two label combinations")
		}
	}
	assert.True(t, found, "fuse_node_exec_duration_seconds not found in gathered metrics")
}

func TestFuseMetrics_prometheusTextOutput(t *testing.T) {
	m := metrics.NewFuseMetrics()
	m.WorkflowsActive.Set(3)
	m.WorkflowsCompleted.Add(10)

	count, err := testutil.GatherAndCount(m.Registry())
	require.NoError(t, err)
	assert.Greater(t, count, 0)
}

func TestFuseMetrics_separateRegistries(t *testing.T) {
	m1 := metrics.NewFuseMetrics()
	m2 := metrics.NewFuseMetrics()

	m1.WorkflowsCompleted.Inc()

	assert.Equal(t, 1.0, testutil.ToFloat64(m1.WorkflowsCompleted))
	assert.Equal(t, 0.0, testutil.ToFloat64(m2.WorkflowsCompleted), "registries must be independent")
}

// Verify the registry satisfies the prometheus.Gatherer interface (compile-time check).
func TestFuseMetrics_registryImplementsGatherer(_ *testing.T) {
	m := metrics.NewFuseMetrics()
	var _ prometheus.Gatherer = m.Registry()
}
