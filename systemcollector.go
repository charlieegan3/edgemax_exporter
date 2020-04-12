package edgemaxexporter

import (
	"time"

	"github.com/charlieegan3/edgemax_exporter/edgemax"
	"github.com/prometheus/client_golang/prometheus"
)

// A SystemCollector is a Prometheus collector for metrics regarding Ubiquiti
// UniFi devices.
type SystemCollector struct {
	CPUPercent    prometheus.Gauge
	UptimeSeconds prometheus.Gauge
	MemoryPercent prometheus.Gauge

	statC <-chan *edgemax.SystemStats
}

// Verify that the Exporter implements the prometheus.Collector interface.
var _ prometheus.Collector = &SystemCollector{}

// NewSystemCollector creates a new SystemCollector which collects metrics for
// a specified site.
func NewSystemCollector(statC <-chan *edgemax.SystemStats) *SystemCollector {
	const (
		subsystem = "system"
	)

	collector := &SystemCollector{
		CPUPercent: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "cpu_percent",
			Help:      "System CPU usage percentage",
		}),

		UptimeSeconds: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "uptime_seconds",
			Help:      "System uptime in seconds",
		}),

		MemoryPercent: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "memory_percent",
			Help:      "System memory usage percentage",
		}),

		statC: statC,
	}

	go collector.collect()

	return collector
}

// collectors contains a list of collectors which are collected each time
// the exporter is scraped.  This list must be kept in sync with the collectors
// in SystemCollector.
func (c *SystemCollector) metrics() []prometheus.Metric {
	return []prometheus.Metric{
		c.CPUPercent,
		c.UptimeSeconds,
		c.MemoryPercent,
	}
}

// collect begins a metrics collection task for all metrics related to UniFi
// devices.
func (c *SystemCollector) collect() {
	for s := range c.statC {
		c.CPUPercent.Set(float64(s.CPU))
		c.UptimeSeconds.Set(float64(s.Uptime / time.Second))
		c.MemoryPercent.Set(float64(s.Memory))
	}
}

// Describe sends the descriptors of each metric over to the provided channel.
// The corresponding metric values are sent separately.
func (c *SystemCollector) Describe(ch chan<- *prometheus.Desc) {
	for _, m := range c.metrics() {
		ch <- m.Desc()
	}
}

// Collect sends the metric values for each metric pertaining to the global
// cluster usage over to the provided prometheus Metric channel.
func (c *SystemCollector) Collect(ch chan<- prometheus.Metric) {
	for _, m := range c.metrics() {
		ch <- m
	}
}
