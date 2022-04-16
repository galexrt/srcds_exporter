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
	"github.com/prometheus/client_golang/prometheus"
)

type playersCollector struct {
	list []*prometheus.Desc
	ping []*prometheus.Desc
	loss []*prometheus.Desc
}

func init() {
	Factories["players"] = NewPlayersCollector
}

// NewPlayersCollector returns a new Collector exposing the current players.
func NewPlayersCollector() (Collector, error) {
	list := []*prometheus.Desc{}
	ping := []*prometheus.Desc{}
	loss := []*prometheus.Desc{}
	for server := range getConnections() {
		list = append(list, prometheus.NewDesc(
			prometheus.BuildFQName(Namespace, "players", "online"),
			"The current players on the server.",
			nil, prometheus.Labels{
				"server": server,
			}))
		ping = append(ping, prometheus.NewDesc(
			prometheus.BuildFQName(Namespace, "players", "ping"),
			"The current players ping on the server.",
			nil, prometheus.Labels{
				"server": server,
			}))
		loss = append(loss, prometheus.NewDesc(
			prometheus.BuildFQName(Namespace, "players", "loss"),
			"The current players loss on the server.",
			nil, prometheus.Labels{
				"server": server,
			}))
	}
	return &playersCollector{
		list: list,
		ping: ping,
		loss: loss,
	}, nil
}

func (c *playersCollector) Update(ch chan<- prometheus.Metric) error {
	for server, con := range getConnections() {
		players, err := con.GetPlayers()
		if err != nil {
			return err
		}

		for _, player := range players {
			list := prometheus.NewDesc(
				prometheus.BuildFQName(Namespace, "players", "online"),
				"The current players on the server.",
				nil, prometheus.Labels{
					"server":  server,
					"steamid": player.SteamID,
				})
			ping := prometheus.NewDesc(
				prometheus.BuildFQName(Namespace, "players", "ping"),
				"The current players ping on the server.",
				nil, prometheus.Labels{
					"server":  server,
					"steamid": player.SteamID,
				})
			loss := prometheus.NewDesc(
				prometheus.BuildFQName(Namespace, "players", "loss"),
				"The current players loss on the server.",
				nil, prometheus.Labels{
					"server":  server,
					"steamid": player.SteamID,
				})
			ch <- prometheus.MustNewConstMetric(
				list, prometheus.GaugeValue, float64(1))
			ch <- prometheus.MustNewConstMetric(
				ping, prometheus.GaugeValue, float64(player.Ping))
			ch <- prometheus.MustNewConstMetric(
				loss, prometheus.GaugeValue, float64(player.Loss))
		}
	}
	return nil
}
