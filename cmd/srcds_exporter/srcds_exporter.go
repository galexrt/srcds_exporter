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

package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/signal"
	"sort"
	"strings"
	"sync"
	"syscall"
	"time"

	yaml "gopkg.in/yaml.v3"

	rcon "github.com/galexrt/go-rcon"
	"github.com/galexrt/srcds_exporter/collector"
	"github.com/galexrt/srcds_exporter/connector"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/version"
	"github.com/sirupsen/logrus"
)

const (
	defaultCollectors = "map,playercount"
)

var (
	showhelp          bool
	showVersion       bool
	showCollectors    bool
	debugMode         bool
	enabledCollectors string
	metricsAddr       string
	metricsPath       string
	configFile        string
)

var (
	connectionDurationDesc = prometheus.NewDesc(
		prometheus.BuildFQName(collector.Namespace, "scrape", "connection_duration_seconds"),
		"srcds_exporter: Duration of the server connection.",
		[]string{"connection"},
		nil,
	)
	connectionSucessDesc = prometheus.NewDesc(
		prometheus.BuildFQName(collector.Namespace, "scrape", "connection_success"),
		"srcds_exporter: Whether the server connection succeeded.",
		[]string{"connection"},
		nil,
	)
	scrapeDurationDesc = prometheus.NewDesc(
		prometheus.BuildFQName(collector.Namespace, "scrape", "collector_duration_seconds"),
		"srcds_exporter: Duration of a collector scrape.",
		[]string{"collector"},
		nil,
	)
	scrapeSuccessDesc = prometheus.NewDesc(
		prometheus.BuildFQName(collector.Namespace, "scrape", "collector_success"),
		"srcds_exporter: Whether a collector succeeded.",
		[]string{"collector"},
		nil,
	)
)

var (
	log         = logrus.New()
	connections *connector.Connector
)

// CurrentConfig current config with a mutex
type CurrentConfig struct {
	sync.RWMutex
	C *Config
}

// Config Config file structure
type Config struct {
	Options Options           `yaml:"options"`
	Servers map[string]Server `yaml:"servers"`
}

// Options Options structure
type Options struct {
	RconTimeout  string `yaml:"rcontimeout"`
	CacheTimeout string `yaml:"cachetimeout"`
}

// Server Server structure
type Server struct {
	Address      string `yaml:"address"`
	RconPassword string `yaml:"rconpassword"`
}

// SRCDSCollector SRCDS Collector map structure
type SRCDSCollector struct {
	collectors map[string]collector.Collector
}

func init() {
	flag.BoolVar(&showhelp, "help", false, "Show help menu")
	flag.BoolVar(&showVersion, "version", false, "Show version information")
	flag.BoolVar(&showCollectors, "collectors.print", false, "If true, print available collectors and exit.")
	flag.BoolVar(&debugMode, "debug", false, "Enable debug output")
	flag.StringVar(&metricsAddr, "web.listen-address", ":9137", "The address to listen on for HTTP requests")
	flag.StringVar(&metricsPath, "web.telemetry-path", "/metrics", "Path the metrics will be exposed under")
	flag.StringVar(&enabledCollectors, "collectors.enabled", defaultCollectors, "Comma separated list of active collectors")
	flag.StringVar(&configFile, "config.file", "./srcds.yaml", "Config file to use.")
}

func (cc *CurrentConfig) reloadConfig(confFile string) (err error) {
	var c = &Config{}

	yamlFile, err := ioutil.ReadFile(confFile)
	if err != nil {
		log.Errorf("Error reading config file: %s", err)
		return err
	}

	if err := yaml.Unmarshal(yamlFile, c); err != nil {
		log.Errorf("Error parsing config file: %s", err)
		return err
	}

	cc.Lock()
	cc.C = c
	loadConnections(cc)
	cc.Unlock()

	log.Infoln("Loaded config file")
	return nil
}

// Describe implements the prometheus.Collector interface.
func (n SRCDSCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- scrapeDurationDesc
	ch <- scrapeSuccessDesc
}

// Collect implements the prometheus.Collector interface.
func (n SRCDSCollector) Collect(ch chan<- prometheus.Metric) {
	wg := sync.WaitGroup{}
	wg.Add(len(n.collectors))
	for name, c := range n.collectors {
		go func(name string, c collector.Collector) {
			execute(name, c, ch)
			wg.Done()
		}(name, c)
	}
	wg.Wait()
}

func filterAvailableCollectors(collectors string) string {
	var availableCollectors []string
	for _, c := range strings.Split(collectors, ",") {
		_, ok := collector.Factories[c]
		if ok {
			availableCollectors = append(availableCollectors, c)
		}
	}
	return strings.Join(availableCollectors, ",")
}

func execute(name string, c collector.Collector, ch chan<- prometheus.Metric) {
	begin := time.Now()
	err := c.Update(ch)
	duration := time.Since(begin)
	var success float64

	if err != nil {
		log.Errorf("ERROR: %s collector failed after %fs: %s", name, duration.Seconds(), err)
		success = 0
	} else {
		log.Debugf("OK: %s collector succeeded after %fs.", name, duration.Seconds())
		success = 1
	}
	ch <- prometheus.MustNewConstMetric(scrapeDurationDesc, prometheus.GaugeValue, duration.Seconds(), name)
	ch <- prometheus.MustNewConstMetric(scrapeSuccessDesc, prometheus.GaugeValue, success, name)
}

func loadCollectors(list string) (map[string]collector.Collector, error) {
	collectors := map[string]collector.Collector{}
	for _, name := range strings.Split(list, ",") {
		fn, ok := collector.Factories[name]
		if !ok {
			return nil, fmt.Errorf("collector '%s' not available", name)
		}
		c, err := fn()
		if err != nil {
			return nil, err
		}
		collectors[name] = c
	}
	return collectors, nil
}

func loadConnections(cc *CurrentConfig) *connector.Connector {
	for name, server := range cc.C.Servers {
		var err error
		for i := 0; i < 5; i++ {
			if err = connections.NewConnection(name,
				&connector.ConnectionOptions{
					Addr:           server.Address,
					RconPassword:   server.RconPassword,
					ConnectTimeout: cc.C.Options.RconTimeout,
					CacheTimeout:   cc.C.Options.CacheTimeout,
				}); err == nil {
				break
			}
		}
		if err != nil {
			log.Fatalf("Error connecting to server: %v", server.Address)
		}
		log.Debugf("Connected to server: %v", server.Address)
	}
	return connections
}

func main() {
	flag.Parse()
	if showhelp {
		fmt.Println(os.Args[0] + " [FLAGS]")
		flag.PrintDefaults()
		return
	}
	if showVersion {
		fmt.Fprintln(os.Stdout, version.Print("srcds_exporter"))
		return
	}
	if showCollectors {
		collectorNames := make(sort.StringSlice, 0, len(collector.Factories))
		for n := range collector.Factories {
			collectorNames = append(collectorNames, n)
		}
		collectorNames.Sort()
		fmt.Printf("Available collectors:\n")
		for _, n := range collectorNames {
			fmt.Printf(" - %s\n", n)
		}
		return
	}
	log.Out = os.Stdout
	if debugMode {
		log.Level = logrus.DebugLevel
	}
	rcon.SetLog(log)
	log.Infoln("Starting srcds_exporter", version.Info())
	log.Infoln("Build context", version.BuildContext())

	connections = connector.NewConnector()
	cc := &CurrentConfig{
		C: &Config{},
	}

	if err := cc.reloadConfig(configFile); err != nil {
		log.Fatalf("Error loading config: %s", err)
	}

	hup := make(chan os.Signal)
	reloadCh := make(chan chan error)
	signal.Notify(hup, syscall.SIGHUP)
	go func() {
		for {
			select {
			case <-hup:
				if err := cc.reloadConfig(configFile); err != nil {
					log.Errorf("Error reloading config: %s", err)
				}
			case rc := <-reloadCh:
				if err := cc.reloadConfig(configFile); err != nil {
					log.Errorf("Error reloading config: %s", err)
					rc <- err
				} else {
					rc <- nil
				}
			}
		}
	}()
	collector.SetConnector(connections)
	defer connections.CloseAll()
	collectors, err := loadCollectors(enabledCollectors)
	if err != nil {
		log.Fatalf("Couldn't load collectors: %s", err)
	}
	log.Infof("Enabled collectors:")
	for n := range collectors {
		log.Infof(" - %s", n)
	}

	if err = prometheus.Register(SRCDSCollector{collectors: collectors}); err != nil {
		log.Fatalf("Couldn't register collector: %s", err)
	}
	handler := promhttp.HandlerFor(prometheus.DefaultGatherer,
		promhttp.HandlerOpts{
			ErrorLog:      log,
			ErrorHandling: promhttp.ContinueOnError,
		})

	http.HandleFunc(metricsPath, func(w http.ResponseWriter, r *http.Request) {
		cc.RLock()
		handler.ServeHTTP(w, r)
		cc.RUnlock()
	})
	http.HandleFunc("/-/reload", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			w.WriteHeader(http.StatusMethodNotAllowed)
			fmt.Fprintf(w, "This endpoint requires a POST request.\n")
			return
		}

		rc := make(chan error)
		reloadCh <- rc
		if err = <-rc; err != nil {
			http.Error(w, fmt.Sprintf("failed to reload config: %s", err), http.StatusInternalServerError)
		}
	})
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<html>
			<head><title>SRCDS Exporter</title></head>
			<body>
			<h1>SRCDS Exporter</h1>
			<p><a href="` + metricsPath + `">Metrics</a></p>
			</body>
			</html>`))
	})
	err = http.ListenAndServe(metricsAddr, nil)
	if err != nil {
		log.Fatal(err)
	}
}
