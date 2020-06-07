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

/*var (
	currentPlayers     = map[string]map[string]struct{}{}
	playersToBeRemoved = map[string]map[string]struct{}{}
)*/

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
	for _, con := range getConnections() {
		/*currentPlayers[con.Name] = map[string]struct{}{}
		playersToBeRemoved[con.Name] = map[string]struct{}{}*/
		list = append(list, prometheus.NewDesc(
			prometheus.BuildFQName(Namespace, "players", "online"),
			"The current players on the server.",
			nil, prometheus.Labels{
				"server": con.Name,
			}))
		ping = append(ping, prometheus.NewDesc(
			prometheus.BuildFQName(Namespace, "players", "ping"),
			"The current players ping on the server.",
			nil, prometheus.Labels{
				"server": con.Name,
			}))
		loss = append(loss, prometheus.NewDesc(
			prometheus.BuildFQName(Namespace, "players", "loss"),
			"The current players loss on the server.",
			nil, prometheus.Labels{
				"server": con.Name,
			}))
		/*resp, _ := con.Get("status")
		players := parser.ParsePlayers(resp)
		for steamID := range players {
			currentPlayers[con.Name][steamID] = struct{}{}
		}*/
	}
	return &playersCollector{
		list: list,
		ping: ping,
		loss: loss,
	}, nil
}

func (c *playersCollector) Update(ch chan<- prometheus.Metric) error {
	for _, con := range getConnections() {
		resp, err := con.Get("status")
		if err != nil {
			return err
		}
		players, err := parser.ParsePlayers(resp)
		if err != nil {
			return err
		}
		//var value = 1
		for _, player := range players {
			/*if _, ok := playersToBeRemoved[con.Name][player.SteamID]; ok {
				fmt.Println("LINE: 51")
				delete(playersToBeRemoved[con.Name], player.SteamID)
				currentPlayers[con.Name][player.SteamID] = struct{}{}
			} else {
				if _, ok := currentPlayers[con.Name][player.SteamID]; !ok {
					fmt.Println("LINE: 56")
					currentPlayers[con.Name][player.SteamID] = struct{}{}
				} else {
					fmt.Println("LINE: 59")
					value = 0
					delete(currentPlayers[con.Name], player.SteamID)
					playersToBeRemoved[con.Name][player.SteamID] = struct{}{}
				}
			}*/
			list := prometheus.NewDesc(
				prometheus.BuildFQName(Namespace, "players", "online"),
				"The current players on the server.",
				nil, prometheus.Labels{
					"server":  con.Name,
					"steamid": player.SteamID,
				})
			ping := prometheus.NewDesc(
				prometheus.BuildFQName(Namespace, "players", "ping"),
				"The current players ping on the server.",
				nil, prometheus.Labels{
					"server":  con.Name,
					"steamid": player.SteamID,
				})
			loss := prometheus.NewDesc(
				prometheus.BuildFQName(Namespace, "players", "loss"),
				"The current players loss on the server.",
				nil, prometheus.Labels{
					"server":  con.Name,
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
