package main

import (
	"flag"
	"strings"

	"github.com/galexrt/srcds_exporter/models"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/log"
)

var (
	metricsAddr = flag.String("metrics.listen-address", ":9137", "The address to listen on for HTTP requests.")
)

var (
	serverMap = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: "gameserver",
		Subsystem: "map",
		Name:      "current",
		Help:      "Current map played.",
		ConstLabels: map[string]string{
			"map": "N/A",
		},
	})
	playerCountCurrent = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace:   "gameserver",
		Subsystem:   "player_count",
		Name:        "current",
		Help:        "Current player count on the server.",
		ConstLabels: map[string]string{},
	})
	playerCountMax = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace:   "gameserver",
		Subsystem:   "player_count",
		Name:        "max",
		Help:        "Maximum player count on the server.",
		ConstLabels: map[string]string{},
	})
)

func init() {
	prometheus.MustRegister(serverMap)
	prometheus.MustRegister(playerCountCurrent)
	prometheus.MustRegister(playerCountMax)
}

func updateMetrics(status models.Status) {
	if !strings.Contains(serverMap.Desc().String(), "map=\""+status.Map+"\"") {
		log.Debug("Update map metrics with new map name")
		prometheus.Unregister(serverMap)
		serverMap = prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: "gameserver",
			Subsystem: "map",
			Name:      "current",
			Help:      "Current map played.",
			ConstLabels: map[string]string{
				"map": status.Map,
			},
		})
		prometheus.MustRegister(serverMap)
		serverMap.Inc()
	} else {
		log.Debug("No map update needed")
	}
	playerCountCurrent.Set(float64(status.PlayerCount.Current))
	playerCountMax.Set(float64(status.PlayerCount.Max))
}
