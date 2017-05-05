package collector

import (
	"github.com/galexrt/srcds_exporter/connector"
	"github.com/prometheus/client_golang/prometheus"
)

const Namespace = "srcds"

// Factories contains the list of all available collectors.
var Factories = make(map[string]func() (Collector, error))

var connections *connector.Connector

// Collector is the interface a collector has to implement.
type Collector interface {
	// Get new metrics and expose them via prometheus registry.
	Update(ch chan<- prometheus.Metric) error
}

// SetConnector a given connector for the collectors
func SetConnector(con *connector.Connector) {
	connections = con
}
