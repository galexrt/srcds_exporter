package main

import (
	"flag"
	"strings"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/galexrt/srcds_exporter/models"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	metricsAddr       string
	metricsUpdateTime string
)

var (
	serverIdentification string
)

// Metric vars
var (
	metricServerMap           prometheus.Counter
	metricPlayerCountCurrent  prometheus.Gauge
	metricPlayerCountMax      prometheus.Gauge
	metricPlayersMetrics      = make(map[int]prometheus.Counter)
	metricsPlayersToBeRemoved = make(map[int]prometheus.Counter)
	metricsMapsToBeRemoved    = make(map[int]prometheus.Counter)
)

func init() {
	flag.StringVar(&metricsAddr, "metrics.listen-address", ":9137", "The address to listen on for HTTP requests.")
	flag.StringVar(&metricsUpdateTime, "metrics.update-time", "12s", "Metrics update time")
}

func initMetrics(status models.Status) {
	metricServerMap = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: "gameserver",
		Subsystem: "map",
		Name:      "current",
		Help:      "Current map played.",
		ConstLabels: map[string]string{
			"server": serverIdentification,
			"map":    status.Map,
		},
	})
	metricPlayerCountCurrent = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "gameserver",
		Subsystem: "player_count",
		Name:      "current",
		Help:      "Current player count on the server.",
		ConstLabels: map[string]string{
			"server": serverIdentification,
		},
	})
	metricPlayerCountMax = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "gameserver",
		Subsystem: "player_count",
		Name:      "max",
		Help:      "Maximum player count on the server.",
		ConstLabels: map[string]string{
			"server": serverIdentification,
		},
	})
	metricServerMap.Inc()
	prometheus.MustRegister(metricServerMap)
	prometheus.MustRegister(metricPlayerCountCurrent)
	prometheus.MustRegister(metricPlayerCountMax)
	updatePlayersMetrics(status.Players)
	go func() {
		for {
			<-time.After(3 * time.Minute)
			cleanupMetrics()
		}
	}()
}

func updateMetrics(status models.Status) {
	if !strings.Contains(metricServerMap.Desc().String(), "map=\""+status.Map+"\"") {
		log.WithFields(logrus.Fields{
			"map": status.Map,
		}).Debug("exporter: map name update required")
		metricServerMap.Desc()
		var key int
		for key = range metricsMapsToBeRemoved {
		}
		metricServerMap.Desc()
		metricsMapsToBeRemoved[key+1] = metricServerMap
		metricServerMap = prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: "gameserver",
			Subsystem: "map",
			Name:      "current",
			Help:      "Current map played.",
			ConstLabels: map[string]string{
				"server": serverIdentification,
				"map":    status.Map,
			},
		})
		metricServerMap.Inc()
		prometheus.MustRegister(metricServerMap)
	} else {
		log.WithFields(logrus.Fields{
			"map": status.Map,
		}).Debug("exporter: no map name update required")
	}
	metricPlayerCountCurrent.Set(float64(status.PlayerCount.Current))
	metricPlayerCountMax.Set(float64(status.PlayerCount.Max))
	updatePlayersMetrics(status.Players)
}

func updatePlayersMetrics(players map[int]models.Player) {
	log.Debugln("updatePlayersMetrics: called")
	for userID, player := range players {
		if metric, ok := metricsPlayersToBeRemoved[userID]; ok {
			metricPlayersMetrics[userID] = metric
			delete(metricsPlayersToBeRemoved, userID)
			log.WithFields(logrus.Fields{
				"userid": userID,
				"player": player,
			}).Debug("updatePlayersMetrics: user seemed to have reconnected or something, removing user from toBeRemoved queue")
			continue
		}
		if _, ok := metricPlayersMetrics[userID]; !ok {
			metricPlayersMetrics[userID] = prometheus.NewCounter(prometheus.CounterOpts{
				Namespace: "gameserver",
				Subsystem: "players",
				Name:      "current",
				Help:      "Current users by Steam ID playing on the server.",
				ConstLabels: map[string]string{
					"server":  serverIdentification,
					"steamid": player.SteamID,
				},
			})
			metricPlayersMetrics[userID].Inc()
			err := prometheus.Register(metricPlayersMetrics[userID])
			log.WithFields(logrus.Fields{
				"userid": userID,
				"player": player,
				"error":  err,
			}).Debug("updatePlayersMetrics: added user metric")
		} else {
			log.WithFields(logrus.Fields{
				"userid": userID,
				"player": player,
			}).Debug("updatePlayersMetrics: user already has metric")
		}
	}
	for userID, metric := range metricPlayersMetrics {
		if _, ok := players[userID]; !ok {
			metric.Desc()
			metricsPlayersToBeRemoved[userID] = metric
			delete(metricPlayersMetrics, userID)
			log.WithFields(logrus.Fields{
				"userid": userID,
			}).Debug("updatePlayersMetrics: removed user metric")
		}
	}
}

func cleanupMetrics() {
	for _, metric := range metricsMapsToBeRemoved {
		prometheus.Unregister(metric)
	}
	for _, metric := range metricsPlayersToBeRemoved {
		prometheus.Unregister(metric)
	}
}
