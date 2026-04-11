package metrics

import (
	"ergo.services/ergo/gen"
	"github.com/prometheus/client_golang/prometheus"
)

// ErgoNodeCollector implements prometheus.Collector and exposes Ergo node-level metrics.
type ErgoNodeCollector struct {
	node gen.Node

	uptime            *prometheus.Desc
	processesTotal    *prometheus.Desc
	processesRunning  *prometheus.Desc
	processesZombee   *prometheus.Desc
	registeredNames   *prometheus.Desc
	registeredAliases *prometheus.Desc
	registeredEvents  *prometheus.Desc
	applicationsTotal *prometheus.Desc
	applicationsRun   *prometheus.Desc
	memoryUsedBytes   *prometheus.Desc
	memoryAllocBytes  *prometheus.Desc
}

// NewErgoNodeCollector creates a prometheus.Collector that exposes Ergo node stats.
func NewErgoNodeCollector(node gen.Node) *ErgoNodeCollector {
	const ns = "ergo"
	labels := []string{"node"}
	return &ErgoNodeCollector{
		node:              node,
		uptime:            prometheus.NewDesc(ns+"_node_uptime_seconds", "Ergo node uptime in seconds.", labels, nil),
		processesTotal:    prometheus.NewDesc(ns+"_processes_total", "Total number of Ergo processes.", labels, nil),
		processesRunning:  prometheus.NewDesc(ns+"_processes_running", "Number of running Ergo processes.", labels, nil),
		processesZombee:   prometheus.NewDesc(ns+"_processes_zombie", "Number of zombie Ergo processes.", labels, nil),
		registeredNames:   prometheus.NewDesc(ns+"_registered_names_total", "Total registered process names.", labels, nil),
		registeredAliases: prometheus.NewDesc(ns+"_registered_aliases_total", "Total registered process aliases.", labels, nil),
		registeredEvents:  prometheus.NewDesc(ns+"_registered_events_total", "Total registered events.", labels, nil),
		applicationsTotal: prometheus.NewDesc(ns+"_applications_total", "Total number of Ergo applications.", labels, nil),
		applicationsRun:   prometheus.NewDesc(ns+"_applications_running", "Number of running Ergo applications.", labels, nil),
		memoryUsedBytes:   prometheus.NewDesc(ns+"_memory_used_bytes", "Current memory usage in bytes.", labels, nil),
		memoryAllocBytes:  prometheus.NewDesc(ns+"_memory_alloc_bytes", "Cumulative bytes allocated.", labels, nil),
	}
}

// Describe implements prometheus.Collector.
func (c *ErgoNodeCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.uptime
	ch <- c.processesTotal
	ch <- c.processesRunning
	ch <- c.processesZombee
	ch <- c.registeredNames
	ch <- c.registeredAliases
	ch <- c.registeredEvents
	ch <- c.applicationsTotal
	ch <- c.applicationsRun
	ch <- c.memoryUsedBytes
	ch <- c.memoryAllocBytes
}

// Collect implements prometheus.Collector.
func (c *ErgoNodeCollector) Collect(ch chan<- prometheus.Metric) {
	info, err := c.node.Info()
	if err != nil {
		return
	}

	nodeName := string(info.Name)
	gauge := func(desc *prometheus.Desc, v float64) prometheus.Metric {
		return prometheus.MustNewConstMetric(desc, prometheus.GaugeValue, v, nodeName)
	}

	ch <- gauge(c.uptime, float64(info.Uptime))
	ch <- gauge(c.processesTotal, float64(info.ProcessesTotal))
	ch <- gauge(c.processesRunning, float64(info.ProcessesRunning))
	ch <- gauge(c.processesZombee, float64(info.ProcessesZombee))
	ch <- gauge(c.registeredNames, float64(info.RegisteredNames))
	ch <- gauge(c.registeredAliases, float64(info.RegisteredAliases))
	ch <- gauge(c.registeredEvents, float64(info.RegisteredEvents))
	ch <- gauge(c.applicationsTotal, float64(info.ApplicationsTotal))
	ch <- gauge(c.applicationsRun, float64(info.ApplicationsRunning))
	ch <- gauge(c.memoryUsedBytes, float64(info.MemoryUsed))
	ch <- gauge(c.memoryAllocBytes, float64(info.MemoryAlloc))
}

// RegisterWith adds this collector to the given prometheus registry.
func (c *ErgoNodeCollector) RegisterWith(reg *prometheus.Registry) {
	reg.MustRegister(c)
}
