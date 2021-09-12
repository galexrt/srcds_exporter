/*
Copyright 2021 Alexander Trost <galexrt@googlemail.com>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package collector

import (
	"github.com/galexrt/srcds_exporter/connector"
	"github.com/prometheus/client_golang/prometheus"
)

// Namespace metric namespace name
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
