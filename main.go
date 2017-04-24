package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/galexrt/srcds_exporter/models"
	"github.com/galexrt/srcds_exporter/parser"

	"github.com/james4k/rcon"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	debug = flag.Bool("debug", true, "Debug output")
)

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
