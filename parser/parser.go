package parser

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/galexrt/srcds_exporter/models"
)

// Parse the given status command lines
func Parse(resp string) *models.Status {
	respLines := strings.Split(resp, "\n")
	status := &models.Status{}

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

func parsePlayerCount(line string) *models.PlayerCount {
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
	return &models.PlayerCount{
		Current: current,
		Max:     max,
	}
}

func parsePlayers(lines []string) map[int]models.Player {
	re := regexp.MustCompile(`(?m)^#[ ]+([0-9]+) "([^"]*)"[ ]+(STEAM_[0-1]:[0-1]:[0-9]+)[ ]+(([0-9]+:)+([0-9]+)?)+[ ]+([0-9]+)[ ]+([0-9]+)[ ]+([a-z]+)[ ]+((1[0-9]{1,2}|2(5[0-6]|[0-4][0-9])|[0-9]{1,2})((\.)(1[0-9]{0,2}|2(5[0-6]|[0-4][0-9])|[0-9]{1,2})){3}):([0-9]+)$`)
	players := make(map[int]models.Player)
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
		players[id] = models.Player{
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
