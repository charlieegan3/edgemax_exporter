// Package exporter provides the Exporter type used in the edgemax_exporter
// Prometheus exporter.
package exporter

import (
	"sync"

	"github.com/charlieegan3/edgemax_exporter/pkg/edgemax"
	"github.com/prometheus/client_golang/prometheus"
)

const (
	// namespace is the top-level namespace for this UniFi exporter.
	namespace = "edgemax"
)

// An Exporter is a Prometheus exporter for Ubiquiti UniFi Controller API
// metrics.  It wraps all UniFi metrics collectors and provides a single global
// exporter which can serve metrics. It also ensures that the collection
// is done in a thread-safe manner, the necessary requirement stated by
// Prometheus. It implements the prometheus.Collector interface in order to
// register with Prometheus.
type Exporter struct {
	mu         sync.Mutex
	collectors []prometheus.Collector
}

// Verify that the Exporter implements the prometheus.Collector interface.
var _ prometheus.Collector = &Exporter{}

// New creates a new Exporter which collects metrics from one or mote sites.
func New(c *edgemax.Client) (*Exporter, func() error, error) {
	statC, done, err := c.Stats()
	if err != nil {
		return nil, nil, err
	}

	dpiStatsC := make(chan edgemax.DPIStats)
	interfacesC := make(chan edgemax.Interfaces)
	systemStatsC := make(chan *edgemax.SystemStats)

	e := &Exporter{
		collectors: []prometheus.Collector{
			NewDPICollector(dpiStatsC),
			NewInterfacesCollector(interfacesC),
			NewSystemCollector(systemStatsC),
		},
	}

	go func() {
		for s := range statC {
			switch s := s.(type) {
			case edgemax.DPIStats:
				dpiStatsC <- s
			case edgemax.Interfaces:
				interfacesC <- s
			case *edgemax.SystemStats:
				systemStatsC <- s
			}
		}
	}()

	return e, done, err
}

// Describe sends all the descriptors of the collectors included to
// the provided channel.
func (c *Exporter) Describe(ch chan<- *prometheus.Desc) {
	for _, cc := range c.collectors {
		cc.Describe(ch)
	}
}

// Collect sends the collected metrics from each of the collectors to
// prometheus. Collect could be called several times concurrently
// and thus its run is protected by a single mutex.
func (c *Exporter) Collect(ch chan<- prometheus.Metric) {
	c.mu.Lock()
	defer c.mu.Unlock()

	for _, cc := range c.collectors {
		cc.Collect(ch)
	}
}
