/*
Copyright 2020 Alexander Trost <galexrt@googlemail.com>

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
	"github.com/galexrt/srcds_exporter/parser"
	"github.com/prometheus/client_golang/prometheus"
)

type mapCollector struct {
	current []*prometheus.Desc
}

func init() {
	Factories["map"] = NewMapCollector
}

// NewMapCollector returns a new Collector exposing the current map.
func NewMapCollector() (Collector, error) {
	current := []*prometheus.Desc{}
	for _, con := range getConnections() {
		resp, err := con.Get("status")
		if err != nil {
			return nil, err
		}
		mapName := parser.ParseMap(resp)
		current = append(current, prometheus.NewDesc(
			prometheus.BuildFQName(Namespace, "", "map"),
			"The current map on the server.",
			nil, prometheus.Labels{
				"server": con.Name,
				"map":    mapName,
			}))
	}
	return &mapCollector{
		current: current,
	}, nil
}

func (c *mapCollector) Update(ch chan<- prometheus.Metric) error {
	for _, con := range getConnections() {
		resp, err := con.Get("status")
		if err != nil {
			return err
		}
		mapName := parser.ParseMap(resp)
		current := prometheus.NewDesc(
			prometheus.BuildFQName(Namespace, "", "map"),
			"The current map on the server.",
			nil, prometheus.Labels{
				"server": con.Name,
				"map":    mapName,
			})
		ch <- prometheus.MustNewConstMetric(
			current, prometheus.GaugeValue, float64(1))
	}
	return nil
}
