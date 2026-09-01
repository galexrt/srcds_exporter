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
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"sort"
	"strings"
	"sync"
	"syscall"
	"time"

	rcon "github.com/galexrt/go-rcon"
	"github.com/galexrt/srcds_exporter/collector"
	"github.com/galexrt/srcds_exporter/config"
	"github.com/galexrt/srcds_exporter/connector"
	"github.com/galexrt/srcds_exporter/connector/connections"
	"github.com/kardianos/service"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/version"
	sloglogrus "github.com/samber/slog-logrus/v2"
	"github.com/sirupsen/logrus"
	flag "github.com/spf13/pflag"
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

	a2sEnabled bool
}

var (
	logger   = logrus.New()
	slogger  = slog.New(sloglogrus.Option{Logger: logger}.NewLogrusHandler())
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
		fatal(err)
	}

	err = s.Run()
	if err != nil {
		slogger.Error("service failed", "error", err)
	}
}

// CurrentConfig current config with a mutex
type CurrentConfig struct {
	sync.RWMutex
	C *config.Config
}

func mapLogrusToSlogLevel(l logrus.Level) slog.Level {
	switch l {
	case logrus.PanicLevel:
		return slog.LevelError
	case logrus.FatalLevel:
		return slog.LevelError
	case logrus.ErrorLevel:
		return slog.LevelError
	case logrus.WarnLevel:
		return slog.LevelWarn
	case logrus.InfoLevel:
		return slog.LevelInfo
	case logrus.DebugLevel:
		return slog.LevelDebug
	case logrus.TraceLevel:
		return slog.LevelDebug
	default:
		return slog.LevelInfo
	}
}

func fatal(err error) {
	slogger.Error("fatal error", "error", err)
	os.Exit(1)
}

func fatalf(format string, args ...any) {
	slogger.Error(fmt.Sprintf(format, args...))
	os.Exit(1)
}

func (p *program) Start(s service.Service) error {
	if err := parseFlagsAndEnvVars(); err != nil {
		fatal(err)
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

	logger.Out = os.Stdout

	// Set log level
	l, err := logrus.ParseLevel(opts.logLevel)
	if err != nil {
		fatal(err)
	}
	logger.SetLevel(l)

	slogger = slog.New(sloglogrus.Option{Level: mapLogrusToSlogLevel(l), Logger: logger}.NewLogrusHandler())
	rcon.SetLog(slogger)

	slogger.Info("Starting srcds_exporter", "version", version.Info())
	slogger.Info("Build context", "context", version.BuildContext())

	if opts.cachingEnabled {
		slogger.Info("Caching enabled", "cache_duration_seconds", opts.cacheDuration)
	} else {
		slogger.Info("Caching is disabled by default")
	}

	if opts.a2sEnabled {
		slogger.Info("A2S query support enabled")
	}

	cons = connector.NewConnector(logger, opts.a2sEnabled)
	cc = &CurrentConfig{
		C: &config.Config{},
	}

	if err := cc.reloadConfig(opts.configFile); err != nil {
		fatalf("Error loading config: %s", err)
	}

	hup := make(chan os.Signal, 1)
	reloadCh := make(chan chan error)
	signal.Notify(hup, syscall.SIGHUP)
	go func() {
		for {
			select {
			case <-hup:
				if err := cc.reloadConfig(opts.configFile); err != nil {
					slogger.Error("Error reloading config", "error", err)
				}

			case rc := <-reloadCh:
				if err := cc.reloadConfig(opts.configFile); err != nil {
					slogger.Error("Error reloading config", "error", err)
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
		fatalf("Couldn't load collectors: %s", err)
	}
	slogger.Info("Enabled collectors:")
	for n := range collectors {
		slogger.Info("enabled collector", "collector", n)
	}

	if err = prometheus.Register(NewSRCDSCollector(collectors, opts.cachingEnabled, opts.cacheDuration)); err != nil {
		fatalf("Couldn't register collector: %s", err)
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

	flags.BoolVar(&opts.a2sEnabled, "a2s", false, "Enable A2S query support (opt-in, required for servers configured with mode: A2S).")
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
	c := &config.Config{}

	yamlFile, err := os.ReadFile(confFile)
	if err != nil {
		slogger.Error("Error reading config file", "error", err)
		return err
	}

	if err := yaml.Unmarshal(yamlFile, c); err != nil {
		slogger.Error("Error parsing config file", "error", err)
		return err
	}

	cc.Lock()
	cc.C = c
	loadConnections(cc)
	cc.Unlock()

	slogger.Info("Loaded config file")
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
			slogger.Debug("Using cache", "now", time.Now(), "expiry", expiry, "last_collect", n.lastCollectTime)
			for _, cachedMetric := range n.cache {
				slogger.Debug("Pushing cached metric to outgoingCh", "metric", cachedMetric.Desc().String())
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
	wgOutgoing.Go(func() {
		for metric := range metricsCh {
			outgoingCh <- metric
			if n.cachingEnabled {
				slogger.Debug("Appending metric to cache", "metric", metric.Desc().String())
				n.cache = append(n.cache, metric)
			}
		}
		slogger.Debug("Finished pushing metrics from metricsCh to outgoingCh")
	})

	wgCollection := sync.WaitGroup{}
	wgCollection.Add(len(n.collectors))
	for name, coll := range n.collectors {
		go func(name string, coll collector.Collector) {
			execute(name, coll, metricsCh)
			wgCollection.Done()
		}(name, coll)
	}

	slogger.Debug("Waiting for collectors")
	wgCollection.Wait()
	slogger.Debug("Finished waiting for collectors")

	n.lastCollectTime = time.Now()
	slogger.Debug("Updated lastCollectTime", "last_collect", n.lastCollectTime)

	close(metricsCh)

	slogger.Debug("Waiting for outgoing Adapter")
	wgOutgoing.Wait()
	slogger.Debug("Finished waiting for outgoing Adapter")
}

func execute(name string, c collector.Collector, ch chan<- prometheus.Metric) {
	begin := time.Now()
	err := c.Update(ch)
	duration := time.Since(begin)
	var success float64

	if err != nil {
		slogger.Error("collector failed", "collector", name, "duration_seconds", duration.Seconds(), "error", err)
		success = 0
	} else {
		slogger.Debug("collector succeeded", "collector", name, "duration_seconds", duration.Seconds())
		success = 1
	}
	ch <- prometheus.MustNewConstMetric(scrapeDurationDesc, prometheus.GaugeValue, duration.Seconds(), name)
	ch <- prometheus.MustNewConstMetric(scrapeSuccessDesc, prometheus.GaugeValue, success, name)
}

func loadConnections(cc *CurrentConfig) error {
	for name, server := range cc.C.Servers {
		var err error
		for range 5 {
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
			fatalf("Error connecting to %v server after 5 tries: %+v", server.Address, err)
		}
		slogger.Debug("Connected to server", "address", server.Address)
	}
	return nil
}

func loadCollectors(list string) (map[string]collector.Collector, error) {
	collectors := map[string]collector.Collector{}
	for name := range strings.SplitSeq(list, ",") {
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
			ErrorLog:      logger,
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

	slogger.Info("Listening", "address", opts.metricsAddr)
	if err := http.ListenAndServe(opts.metricsAddr, nil); err != nil {
		fatal(err)
	}
}
