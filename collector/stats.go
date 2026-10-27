/*
Copyright 2026 Alexander Trost <galexrt@googlemail.com>

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

import "github.com/prometheus/client_golang/prometheus"

type statsCollector struct{}

func init() {
	Factories["stats"] = NewStatsCollector
}

// NewStatsCollector returns a collector exposing the SRCDS stats command.
func NewStatsCollector() (Collector, error) {
	return &statsCollector{}, nil
}

func (c *statsCollector) Update(ch chan<- prometheus.Metric) error {
	for server, con := range getConnections() {
		labels := prometheus.Labels{"server": server}
		available := prometheus.NewDesc(
			prometheus.BuildFQName(Namespace, "stats", "available"),
			"Whether stats metrics are available for the server.", nil, labels)
		stats, err := con.GetStats()
		if err != nil {
			ch <- prometheus.MustNewConstMetric(available, prometheus.GaugeValue, 0)
			continue
		}

		ch <- prometheus.MustNewConstMetric(available, prometheus.GaugeValue, 1)
		fps := prometheus.NewDesc(
			prometheus.BuildFQName(Namespace, "stats", "fps"),
			"Current server frames per second.", nil, labels)
		netIn := prometheus.NewDesc(
			prometheus.BuildFQName(Namespace, "stats", "netin_kilobytes_per_second"),
			"Current inbound server network traffic in kilobytes per second.", nil, labels)
		netOut := prometheus.NewDesc(
			prometheus.BuildFQName(Namespace, "stats", "netout_kilobytes_per_second"),
			"Current outbound server network traffic in kilobytes per second.", nil, labels)
		ch <- prometheus.MustNewConstMetric(fps, prometheus.GaugeValue, stats.FPS)
		ch <- prometheus.MustNewConstMetric(netIn, prometheus.GaugeValue, stats.NetInKilobytesPerSecond)
		ch <- prometheus.MustNewConstMetric(netOut, prometheus.GaugeValue, stats.NetOutKilobytesPerSecond)
		if stats.HasConnects {
			connects := prometheus.NewDesc(
				prometheus.BuildFQName(Namespace, "stats", "connects_total"),
				"Total client connections accepted by the server since startup.", nil, labels)
			ch <- prometheus.MustNewConstMetric(connects, prometheus.CounterValue, stats.Connects)
		}
	}
	return nil
}
