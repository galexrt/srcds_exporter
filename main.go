package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/Sirupsen/logrus"
	steam "github.com/galexrt/go-steam"
	"github.com/galexrt/srcds_exporter/models"
	"github.com/galexrt/srcds_exporter/parser"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	help           bool
	debug          bool
	connectTimeout string
)

var (
	log          = logrus.New()
	metricUpdate = make(chan models.Status)
)

func init() {
	flag.BoolVar(&help, "help", false, "Show the help menu")
	flag.BoolVar(&debug, "debug", false, "Enable debug output")
	flag.StringVar(&connectTimeout, "timeout", "15s", "Connection timeout")
}

func main() {
	flag.Parse()
	if help {
		fmt.Println(os.Args[0] + " [FLAGS]")
		flag.PrintDefaults()
		os.Exit(0)
	}
	log.Out = os.Stdout
	if debug {
		log.Level = logrus.DebugLevel
	}
	steam.SetLog(log)
	addr := os.Getenv("ADDR")
	serverIdentification = addr
	pass := os.Getenv("RCON_PASSWORD")
	if addr == "" || pass == "" {
		fmt.Println("Please set ADDR & RCON_PASSWORD.")
		return
	}
	metricsUpdateTimeDuration, err := time.ParseDuration(metricsUpdateTime)
	if err != nil {
		panic(err)
	}
	go func() {
		manageMetrics()
	}()
	go func() {
		for {
			con, err := steam.Connect(addr, &steam.ConnectOptions{
				RCONPassword: pass,
				Timeout:      connectTimeout,
			})
			if err != nil {
				fmt.Println(err)
				time.Sleep(1 * time.Second)
				continue
			}
			defer con.Close()
			for {
				resp, err := con.Send("status")
				if err != nil {
					log.Error(err)
					break
				}
				log.Debug("Read status command output")
				metricUpdate <- *parser.Parse(resp)

				time.Sleep(metricsUpdateTimeDuration)
			}
		}
	}()
	http.Handle("/metrics", promhttp.Handler())
	log.Fatal(http.ListenAndServe(metricsAddr, nil))
}

func manageMetrics() {
	first := true
	for {
		status := <-metricUpdate
		if first {
			initMetrics(status)
			first = false
			log.Debug("exporter: init metrics")
		}
		log.Debug("exporter: received metrics update")
		updateMetrics(status)
	}
}
