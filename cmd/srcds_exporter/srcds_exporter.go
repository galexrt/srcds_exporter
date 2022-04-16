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

package main

import (
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

	flag "github.com/spf13/pflag"

	rcon "github.com/galexrt/go-rcon"
	"github.com/galexrt/srcds_exporter/collector"
	"github.com/galexrt/srcds_exporter/config"
	"github.com/galexrt/srcds_exporter/connector"
	"github.com/galexrt/srcds_exporter/connector/connections"
	"github.com/kardianos/service"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/version"
	"github.com/sirupsen/logrus"
	yaml "gopkg.in/yaml.v3"
)

const (
	defaultCollectors = "map,playercount"
)

var (
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

type program struct{}

// CmdLineOpts holds possible command line options/flags
type CmdLineOpts struct {
	version        bool
	showCollectors bool
	logLevel       string

	metricsAddr           string
	metricsPath           string
	enabledCollectors     string
	configFile            string
	reloadEndpointEnabled bool

	cachingEnabled bool
	cacheDuration  int64
}

var (
	log      = logrus.New()
	opts     CmdLineOpts
	flags    = flag.NewFlagSet("srcds_exporter", flag.ExitOnError)
	cons     *connector.Connector
	cc       *CurrentConfig
	reloadCh chan chan error
)

// SRCDSCollector contains the collectors to be used
type SRCDSCollector struct {
	lastCollectTime time.Time
	collectors      map[string]collector.Collector

	// Cache related
	cachingEnabled bool
	cacheDuration  time.Duration
	cache          []prometheus.Metric
	cacheMutex     sync.Mutex
}

func main() {
	// Service setup
	svcConfig := &service.Config{
		Name:        "SRCDSExporter",
		DisplayName: "SRCDS Exporter",
		Description: "Prometheus exporter for SRCDS based Gameservers",
	}

	prg := &program{}
	s, err := service.New(prg, svcConfig)
	if err != nil {
		log.Fatal(err)
	}

	err = s.Run()
	if err != nil {
		log.Error(err)
	}
}

// CurrentConfig current config with a mutex
type CurrentConfig struct {
	sync.RWMutex
	C *config.Config
}

func (p *program) Start(s service.Service) error {
	if err := parseFlagsAndEnvVars(); err != nil {
		log.Fatal(err)
	}

	if opts.version {
		fmt.Fprintln(os.Stdout, version.Print("srcds_exporter"))
		os.Exit(0)
	}

	if opts.showCollectors {
		collectorNames := make(sort.StringSlice, 0, len(collector.Factories))
		for n := range collector.Factories {
			collectorNames = append(collectorNames, n)
		}
		collectorNames.Sort()
		fmt.Printf("Available collectors:\n")
		for _, n := range collectorNames {
			fmt.Printf(" - %s\n", n)
		}
		os.Exit(0)
	}

	log.Out = os.Stdout

	// Set log level
	l, err := logrus.ParseLevel(opts.logLevel)
	if err != nil {
		log.Fatal(err)
	}
	log.SetLevel(l)

	rcon.SetLog(log)

	log.Infoln("Starting srcds_exporter", version.Info())
	log.Infoln("Build context", version.BuildContext())

	if opts.cachingEnabled {
		log.Infof("Caching enabled. Cache Duration: %ds", opts.cacheDuration)
	} else {
		log.Info("Caching is disabled by default")
	}

	cons = connector.NewConnector()
	cc = &CurrentConfig{
		C: &config.Config{},
	}

	if err := cc.reloadConfig(opts.configFile); err != nil {
		log.Fatalf("Error loading config: %s", err)
	}

	hup := make(chan os.Signal)
	reloadCh := make(chan chan error)
	signal.Notify(hup, syscall.SIGHUP)
	go func() {
		for {
			select {
			case <-hup:
				if err := cc.reloadConfig(opts.configFile); err != nil {
					log.Errorf("Error reloading config: %s", err)
				}
			case rc := <-reloadCh:
				if err := cc.reloadConfig(opts.configFile); err != nil {
					log.Errorf("Error reloading config: %s", err)
					rc <- err
				} else {
					rc <- nil
				}
			}
		}
	}()
	collector.SetConnector(cons)

	collectors, err := loadCollectors(opts.enabledCollectors)
	if err != nil {
		log.Fatalf("Couldn't load collectors: %s", err)
	}
	log.Infof("Enabled collectors:")
	for n := range collectors {
		log.Infof(" - %s", n)
	}

	if err = prometheus.Register(NewSRCDSCollector(collectors, opts.cachingEnabled, opts.cacheDuration)); err != nil {
		log.Fatalf("Couldn't register collector: %s", err)
	}

	// non-blocking start
	go p.run()
	return nil
}

func (p *program) Stop(s service.Service) error {
	// non-blocking stop
	return nil
}

func NewSRCDSCollector(collectors map[string]collector.Collector, cachingEnabled bool, cacheDurationSeconds int64) *SRCDSCollector {
	return &SRCDSCollector{
		cache:           make([]prometheus.Metric, 0),
		lastCollectTime: time.Unix(0, 0),
		collectors:      collectors,
		cachingEnabled:  cachingEnabled,
		cacheDuration:   time.Duration(cacheDurationSeconds) * time.Second,
	}
}

func init() {
	flags.BoolVar(&opts.version, "version", false, "Show version information")
	flags.StringVar(&opts.logLevel, "log-level", "INFO", "Set log level")

	flags.BoolVar(&opts.showCollectors, "collectors.print", false, "If true, print available collectors and exit.")
	flags.StringVar(&opts.enabledCollectors, "collectors.enabled", defaultCollectors, "Comma separated list of active collectors")

	flags.StringVar(&opts.metricsAddr, "web.listen-address", ":9137", "The address to listen on for HTTP requests")
	flags.StringVar(&opts.metricsPath, "web.telemetry-path", "/metrics", "Path the metrics will be exposed under")
	flags.BoolVar(&opts.reloadEndpointEnabled, "web.reload-endpoint-enabled", false, "Enable/Disable the POST config reload endpoint.")

	flags.StringVar(&opts.configFile, "config.file", "./srcds.yaml", "Config file to use.")
}

func flagNameFromEnvName(s string) string {
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, "_", "-")
	return s
}

func parseFlagsAndEnvVars() error {
	for _, v := range os.Environ() {
		vals := strings.SplitN(v, "=", 2)

		if !strings.HasPrefix(vals[0], "SRCDS_EXPORTER_") {
			continue
		}
		flagName := flagNameFromEnvName(strings.ReplaceAll(vals[0], "SRCDS_EXPORTER_", ""))

		fn := flags.Lookup(flagName)
		if fn == nil || fn.Changed {
			continue
		}

		if err := fn.Value.Set(vals[1]); err != nil {
			return err
		}
	}

	return flags.Parse(os.Args[1:])
}

func (cc *CurrentConfig) reloadConfig(confFile string) (err error) {
	var c = &config.Config{}

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
func (n *SRCDSCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- scrapeDurationDesc
	ch <- scrapeSuccessDesc
}

// Collect implements the prometheus.Collector interface.
func (n *SRCDSCollector) Collect(outgoingCh chan<- prometheus.Metric) {
	if n.cachingEnabled {
		n.cacheMutex.Lock()
		defer n.cacheMutex.Unlock()

		expiry := n.lastCollectTime.Add(n.cacheDuration)
		if time.Now().Before(expiry) {
			log.Debugf("Using cache. Now: %s, Expiry: %s, LastCollect: %s", time.Now().String(), expiry.String(), n.lastCollectTime.String())
			for _, cachedMetric := range n.cache {
				log.Debugf("Pushing cached metric %s to outgoingCh", cachedMetric.Desc().String())
				outgoingCh <- cachedMetric
			}
			return
		}
		// Clear cache, but keep slice
		n.cache = n.cache[:0]
	}

	metricsCh := make(chan prometheus.Metric)

	// Wait to ensure outgoingCh is not closed before the goroutine is finished
	wgOutgoing := sync.WaitGroup{}
	wgOutgoing.Add(1)
	go func() {
		for metric := range metricsCh {
			outgoingCh <- metric
			if n.cachingEnabled {
				log.Debugf("Appending metric %s to cache", metric.Desc().String())
				n.cache = append(n.cache, metric)
			}
		}
		log.Debug("Finished pushing metrics from metricsCh to outgoingCh")
		wgOutgoing.Done()
	}()

	wgCollection := sync.WaitGroup{}
	wgCollection.Add(len(n.collectors))
	for name, coll := range n.collectors {
		go func(name string, coll collector.Collector) {
			execute(name, coll, metricsCh)
			wgCollection.Done()
		}(name, coll)
	}

	log.Debug("Waiting for collectors")
	wgCollection.Wait()
	log.Debug("Finished waiting for collectors")

	n.lastCollectTime = time.Now()
	log.Debugf("Updated lastCollectTime to %s", n.lastCollectTime.String())

	close(metricsCh)

	log.Debug("Waiting for outgoing Adapter")
	wgOutgoing.Wait()
	log.Debug("Finished waiting for outgoing Adapter")
}

func execute(name string, c collector.Collector, ch chan<- prometheus.Metric) {
	begin := time.Now()
	err := c.Update(ch)
	duration := time.Since(begin)
	var success float64

	if err != nil {
		log.Errorf("%s collector failed after %fs: %s", name, duration.Seconds(), err)
		success = 0
	} else {
		log.Debugf("%s collector succeeded after %fs.", name, duration.Seconds())
		success = 1
	}
	ch <- prometheus.MustNewConstMetric(scrapeDurationDesc, prometheus.GaugeValue, duration.Seconds(), name)
	ch <- prometheus.MustNewConstMetric(scrapeSuccessDesc, prometheus.GaugeValue, success, name)
}

func loadConnections(cc *CurrentConfig) error {
	for name, server := range cc.C.Servers {
		var err error
		for i := 0; i < 5; i++ {
			if err = cons.NewConnection(name,
				&connections.ConnectionOptions{
					Addr:                 server.Address,
					Mode:                 server.Mode,
					RCONPassword:         server.RCONPassword,
					ConnectTimeout:       cc.C.Options.ConnectTimeout,
					CacheCleanupInterval: cc.C.Options.CacheCleanupInterval,
					CacheExpiration:      cc.C.Options.CacheExpiration,
				}); err == nil {
				break
			}
		}
		if err != nil {
			log.Fatalf("Error connecting to %v server after 5 tries: %+v", server.Address, err)
		}
		log.Debugf("Connected to server: %v", server.Address)
	}
	return nil
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

func (p *program) run() {
	// Defer connection closing
	defer cons.CloseAll()

	// Background work
	handler := promhttp.HandlerFor(prometheus.DefaultGatherer,
		promhttp.HandlerOpts{
			ErrorLog:      log,
			ErrorHandling: promhttp.ContinueOnError,
		})

	http.HandleFunc(opts.metricsPath, func(w http.ResponseWriter, r *http.Request) {
		cc.RLock()
		handler.ServeHTTP(w, r)
		cc.RUnlock()
	})

	// Enable reload endpoint only when enabled by the flag
	if opts.reloadEndpointEnabled {
		http.HandleFunc("/-/reload", func(w http.ResponseWriter, r *http.Request) {
			if r.Method != "POST" {
				w.WriteHeader(http.StatusMethodNotAllowed)
				fmt.Fprintf(w, "This endpoint requires a POST request.\n")
				return
			}

			rc := make(chan error)
			reloadCh <- rc
			if err := <-rc; err != nil {
				http.Error(w, fmt.Sprintf("failed to reload config: %s", err), http.StatusInternalServerError)
			}
		})
	}
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<!DOCTYPE html>
		<html>
			<head><title>SRCDS Exporter</title></head>
			<body>
				<h1>SRCDS Exporter</h1>
				<p><a href="` + opts.metricsPath + `">Metrics</a></p>
			</body>
		</html>`))
	})

	log.Info("Listening on " + opts.metricsAddr)
	if err := http.ListenAndServe(opts.metricsAddr, nil); err != nil {
		log.Fatal(err)
	}
}
