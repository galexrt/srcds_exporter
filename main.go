package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/galexrt/srcds_exporter/models"
	"github.com/galexrt/srcds_exporter/parser"

	"github.com/james4k/rcon"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	metricsAddr = flag.String("metrics.listen-address", ":9137", "The address to listen on for HTTP requests.")
	debug       = flag.Bool("debug", true, "Debug output")
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

var metricUpdate = make(chan models.Status)

func main() {
	flag.Parse()
	if *debug {
		log.SetLevel(log.DebugLevel)
	}
	addr := os.Getenv("ADDR")
	pass := os.Getenv("RCON_PASSWORD")
	if addr == "" || pass == "" {
		fmt.Println("Please set ADDR & RCON_PASSWORD.")
		return
	}
	go func() {
		manageMetrics()
	}()
	go func() {
		for {
			con, err := rcon.Dial(addr, pass)
			if err != nil {
				fmt.Println(err)
				time.Sleep(1 * time.Second)
				continue
			}
			defer con.Close()
			for {
				_, err := con.Write("status")
				if err != nil {
					fmt.Println(err)
					break
				}
				resp, _, err := con.Read()
				if err != nil {
					fmt.Println(err)
					break
				}
				log.Debug("Read status command output")
				metricUpdate <- *parser.Parse(resp)

				time.Sleep(5 * time.Second)
			}
		}
	}()
	http.Handle("/metrics", promhttp.Handler())
	log.Fatal(http.ListenAndServe(*metricsAddr, nil))
}

func manageMetrics() {
	for {
		status := <-metricUpdate
		log.Debugln("manageMetrics: Received metrics update")
		updateMetrics(status)
	}
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
