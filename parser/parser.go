package parser

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/galexrt/srcds_exporter/models"
)

var (
	playerRegex = regexp.MustCompile(`^#[ ]+([0-9]+)[ ]+"([^"]*)"[ ]+(STEAM_[0-1]:[0-1]:[0-9]+)[ ]+[0-9:]+[ ]+([0-9]+)[ ]+([0-9]+)[ ]+([a-z]+)[ ]+(([0-9]{1,3}.){3}[0-9]{1,3}):([0-9]+)$`)
)

// Parse the given status command lines
func Parse(resp string) *models.Status {
	respLines := strings.Split(resp, "\n")
	status := &models.Status{Players: make(map[int]models.Player)}

	// Parse hostname, version, map, playerCount and players
	reIsInfo, err := regexp.Compile(`(?m)^([A-Za-z/]+)[ ]*: (.*)$`)
	if err != nil {
		panic(err)
	}
	for _, line := range respLines {
		if line != "" {
			if match := reIsInfo.FindStringSubmatch(line); len(match) > 1 && match[1] != "" {
				switch match[1] {
				case "hostname":
					status.Hostname = parseHostname(match[2])
				case "version":
					status.Version = parseVersion(match[2])
				case "map":
					status.Map = parseMap(match[2])
				case "players":
					status.PlayerCount = *parsePlayerCount(match[2])
				}
			} else if len(line) >= 3 && line[0:3] != "# u" {
				id, player := parsePlayer(line)
				status.Players[id] = player
			}
		}
	}
	return status
}

func parseHostname(line string) string {
	re := regexp.MustCompile(`(?m)^(.*)$`)
	return re.FindStringSubmatch(line)[1]
}

func parseVersion(line string) string {
	re := regexp.MustCompile(`(?m)^(.*)$`)
	return re.FindStringSubmatch(line)[1]
}

func parseMap(line string) string {
	re := regexp.MustCompile(`(?m)^([a-zA-Z_0-9-]+).*$`)
	return re.FindStringSubmatch(line)[1]
}

func parsePlayerCount(line string) *models.PlayerCount {
	re := regexp.MustCompile(`(?m)^([0-9]+) \(([0-9]+) max\)$`)
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
	players := make(map[int]models.Player)
	for _, line := range lines {
		id, player := parsePlayer(line)
		players[id] = player
	}
	return players
}

func parsePlayer(line string) (int, models.Player) {
	line = strings.Replace(line, "\000", "", -1)
	m := playerRegex.FindStringSubmatch(line)
	if len(m) == 0 {
		return 0, models.Player{}
	}
	userID, err := strconv.Atoi(m[1])
	if err != nil {
		panic(err)
	}
	ping, err := strconv.Atoi(m[4])
	if err != nil {
		panic(err)
	}
	loss, err := strconv.Atoi(m[5])
	if err != nil {
		panic(err)
	}
	connPort, err := strconv.Atoi(m[9])
	if err != nil {
		panic(err)
	}
	return userID, models.Player{
		Username: m[2],
		SteamID:  m[3],
		State:    m[8],
		Ping:     ping,
		Loss:     loss,
		IP:       m[7],
		ConnPort: connPort,
	}
}
