package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"

	"github.com/james4k/rcon"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	metricsAddr = flag.String("metrics.listen-address", ":9137", "The address to listen on for HTTP requests.")
	debug       = flag.Bool("debug", true, "Debug output")
)

// Status Contains the server status
type Status struct {
	Hostname    string
	Version     string
	Map         string
	PlayerCount PlayerCount
	Players     map[int]Player
}

// PlayerCount contains current and max players of server
type PlayerCount struct {
	Current int
	Max     int
}

// Player contains player information like username, steamID, etc.
type Player struct {
	Username string
	SteamID  string
	State    string
	Ping     int
	Loss     int
	IP       string
	ConnPort int
}

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

var metricUpdate = make(chan Status)

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
				metricUpdate <- *parseStatus(resp)

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
func updateMetrics(status Status) {
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
	playerCountCurrent.Set(float64(status.PlayerCount.Current))
	playerCountMax.Set(float64(status.PlayerCount.Max))
}

func parseStatus(resp string) *Status {
	status := &Status{}
	respLines := strings.Split(resp, "\n")

	// Parse hostname, version, map, playerCount and players
	status.Hostname = parseHostname(respLines[0])
	status.Version = parseVersion(respLines[1])
	status.Map = parseMap(respLines[3])
	status.PlayerCount = *parsePlayerCount(respLines[4])
	status.Players = parsePlayers(respLines[7:])
	return status
}

func parseHostname(line string) string {
	re := regexp.MustCompile(`(?m)^hostname: (.*)$`)
	return re.FindStringSubmatch(line)[1]
}

func parseVersion(line string) string {
	re := regexp.MustCompile(`(?m)^version[ ]*: (.*)$`)
	return re.FindStringSubmatch(line)[1]
}

func parseMap(line string) string {
	re := regexp.MustCompile(`(?m)^map[ ]*: ([a-zA-Z_]+).*$`)
	return re.FindStringSubmatch(line)[1]
}

func parsePlayerCount(line string) *PlayerCount {
	re := regexp.MustCompile(`(?m)^players[ ]*: ([0-9]+) \(([0-9]+) max\)$`)
	parsed := re.FindStringSubmatch(line)
	current, err := strconv.Atoi(parsed[1])
	if err != nil {
		panic(err)
	}
	max, err := strconv.Atoi(parsed[2])
	if err != nil {
		panic(err)
	}
	return &PlayerCount{
		Current: current,
		Max:     max,
	}
}

func parsePlayers(lines []string) map[int]Player {
	re := regexp.MustCompile(`(?m)^#[ ]+([0-9]+) "([^"]*)"[ ]+(STEAM_[0-1]:[0-1]:[0-9]+)[ ]+(([0-9]+:)+([0-9]+)?)+[ ]+([0-9]+)[ ]+([0-9]+)[ ]+([a-z]+)[ ]+((1[0-9]{1,2}|2(5[0-6]|[0-4][0-9])|[0-9]{1,2})((\.)(1[0-9]{0,2}|2(5[0-6]|[0-4][0-9])|[0-9]{1,2})){3}):([0-9]+)$`)
	players := make(map[int]Player)
	for _, line := range lines {
		m := re.FindStringSubmatch(line)
		if line == "" {
			continue
		}
		id, err := strconv.Atoi(m[1])
		if err != nil {
			panic(err)
		}
		ping, err := strconv.Atoi(m[7])
		if err != nil {
			panic(err)
		}
		loss, err := strconv.Atoi(m[8])
		if err != nil {
			panic(err)
		}
		connPort, err := strconv.Atoi(m[11])
		if err != nil {
			panic(err)
		}
		players[id] = Player{
			Username: m[2],
			SteamID:  m[3],
			State:    m[9],
			Ping:     ping,
			Loss:     loss,
			IP:       m[10],
			ConnPort: connPort,
		}
	}
	return players
}
