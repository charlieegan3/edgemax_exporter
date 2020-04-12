package edgemaxexporter

import (
	"github.com/charlieegan3/edgemax_exporter/edgemax"
	"github.com/prometheus/client_golang/prometheus"
)

// A InterfacesCollector is a Prometheus collector for metrics regarding Ubiquiti
// UniFi devices.
type InterfacesCollector struct {
	ReceivedBytes    *prometheus.GaugeVec
	TransmittedBytes *prometheus.GaugeVec

	statC <-chan edgemax.Interfaces
}

// Verify that the Exporter implements the prometheus.Collector interface.
var _ prometheus.Collector = &InterfacesCollector{}

// NewInterfacesCollector creates a new InterfacesCollector which collects metrics for
// a specified site.
func NewInterfacesCollector(statC <-chan edgemax.Interfaces) *InterfacesCollector {
	const (
		subsystem = "interfaces"
	)

	var (
		labelsInterfaces = []string{"name", "mac"}
	)

	collector := &InterfacesCollector{
		ReceivedBytes: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "received_bytes",
				Help:      "Number of bytes received by interfaces, partitioned by network interface",
			},
			labelsInterfaces,
		),

		TransmittedBytes: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "transmitted_bytes",
				Help:      "Number of bytes transmitted by interfaces, partitioned by network interface",
			},
			labelsInterfaces,
		),

		statC: statC,
	}

	go collector.collect()

	return collector
}

// collectors contains a list of collectors which are collected each time
// the exporter is scraped.  This list must be kept in sync with the collectors
// in InterfacesCollector.
func (c *InterfacesCollector) collectors() []prometheus.Collector {
	return []prometheus.Collector{
		c.ReceivedBytes,
		c.TransmittedBytes,
	}
}

// collect begins a metrics collection task for all metrics related to UniFi
// devices.
func (c *InterfacesCollector) collect() {
	for s := range c.statC {
		for _, ifi := range s {
			labels := []string{
				ifi.Name,
				ifi.MAC.String(),
			}

			c.ReceivedBytes.WithLabelValues(labels...).Set(float64(ifi.Stats.ReceiveBytes))
			c.TransmittedBytes.WithLabelValues(labels...).Set(float64(ifi.Stats.TransmitBytes))
		}
	}
}

// Describe sends the descriptors of each metric over to the provided channel.
// The corresponding metric values are sent separately.
func (c *InterfacesCollector) Describe(ch chan<- *prometheus.Desc) {
	for _, m := range c.collectors() {
		m.Describe(ch)
	}
}

// Collect sends the metric values for each metric pertaining to the global
// cluster usage over to the provided prometheus Metric channel.
func (c *InterfacesCollector) Collect(ch chan<- prometheus.Metric) {
	for _, m := range c.collectors() {
		m.Collect(ch)
	}
}
