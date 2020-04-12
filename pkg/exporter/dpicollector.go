package exporter

import (
	"github.com/charlieegan3/edgemax_exporter/pkg/edgemax"
	"github.com/prometheus/client_golang/prometheus"
)

// A DPICollector is a Prometheus collector for metrics regarding EdgeMAX
// deep packet inspection statistics.
type DPICollector struct {
	ReceivedBytes    *prometheus.GaugeVec
	TransmittedBytes *prometheus.GaugeVec

	statC <-chan edgemax.DPIStats
}

// Verify that the Exporter implements the prometheus.Collector interface.
var _ prometheus.Collector = &DPICollector{}

// NewDPICollector creates a new DPICollector.
func NewDPICollector(statC <-chan edgemax.DPIStats) *DPICollector {
	const (
		subsystem = "dpi"
	)

	var (
		labelsDPIs = []string{"client_ip", "category", "type"}
	)

	collector := &DPICollector{
		ReceivedBytes: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "received_bytes",
				Help:      "Number of bytes received by devices (client download)",
			},
			labelsDPIs,
		),

		TransmittedBytes: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "transmitted_bytes",
				Help:      "Number of bytes transmitted by devices (client upload)",
			},
			labelsDPIs,
		),

		statC: statC,
	}

	go collector.collect()

	return collector
}

// collectors contains a list of collectors which are collected each time
// the exporter is scraped.  This list must be kept in sync with the collectors
// in DPICollector.
func (c *DPICollector) collectors() []prometheus.Collector {
	return []prometheus.Collector{
		c.ReceivedBytes,
		c.TransmittedBytes,
	}
}

// collect begins a metrics collection task for all metrics related to UniFi
// devices.
func (c *DPICollector) collect() {
	for stat := range c.statC {
		for _, s := range stat {
			labels := []string{
				s.IP.String(),
				s.Category,
				s.Type,
			}

			c.ReceivedBytes.WithLabelValues(labels...).Set(float64(s.ReceiveBytes))
			c.TransmittedBytes.WithLabelValues(labels...).Set(float64(s.TransmitBytes))
		}
	}
}

// Describe sends the descriptors of each metric over to the provided channel.
// The corresponding metric values are sent separately.
func (c *DPICollector) Describe(ch chan<- *prometheus.Desc) {
	for _, m := range c.collectors() {
		m.Describe(ch)
	}
}

// Collect sends the metric values for each metric pertaining to deep packet
// inspection to the provided prometheus Metric channel.
func (c *DPICollector) Collect(ch chan<- prometheus.Metric) {
	for _, m := range c.collectors() {
		m.Collect(ch)
	}
}
