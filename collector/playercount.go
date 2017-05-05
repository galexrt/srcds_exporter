package collector

import (
	"github.com/galexrt/srcds_exporter/parser"
	"github.com/prometheus/client_golang/prometheus"
)

type playerCountCollector struct {
	current []*prometheus.Desc
	limit   []*prometheus.Desc
}

func init() {
	Factories["playercount"] = NewPlayerCountCollector
}

// NewPlayerCountCollector returns a new Collector exposing the current map.
func NewPlayerCountCollector() (Collector, error) {
	current := []*prometheus.Desc{}
	limit := []*prometheus.Desc{}
	for _, con := range getConnections() {
		current = append(current, prometheus.NewDesc(
			prometheus.BuildFQName(Namespace, "playercount", "current"),
			"The current player count of the server.",
			nil, prometheus.Labels{
				"server": con.Name,
			}))
		limit = append(limit, prometheus.NewDesc(
			prometheus.BuildFQName(Namespace, "playercount", "limit"),
			"The current player count of the server.",
			nil, prometheus.Labels{
				"server": con.Name,
			}))
	}
	return &playerCountCollector{
		current: current,
		limit:   limit,
	}, nil
}

func (c *playerCountCollector) Update(ch chan<- prometheus.Metric) error {
	for _, con := range getConnections() {
		resp, err := con.Get("status")
		if err != nil {
			return err
		}
		playerCount, err := parser.ParsePlayerCount(resp)
		if err != nil {
			return err
		}
		current := prometheus.NewDesc(
			prometheus.BuildFQName(Namespace, "playercount", "current"),
			"The current map on the server.",
			nil, prometheus.Labels{
				"server": con.Name,
			})
		limit := prometheus.NewDesc(
			prometheus.BuildFQName(Namespace, "playercount", "limit"),
			"The limit of players on the server.",
			nil, prometheus.Labels{
				"server": con.Name,
			})
		ch <- prometheus.MustNewConstMetric(
			current, prometheus.GaugeValue, float64(playerCount.Current))
		ch <- prometheus.MustNewConstMetric(
			limit, prometheus.GaugeValue, float64(playerCount.Max))
	}
	return nil
}
