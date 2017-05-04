package parser

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/galexrt/srcds_exporter/parser/models"
)

var (
	hostnameRegex    = regexp.MustCompile(`(?m)^hostname[ ]*: (.*)$`)
	versionRegex     = regexp.MustCompile(`(?m)^version[ ]*: (.*)$`)
	mapRegex         = regexp.MustCompile(`(?m)^map[ ]*: ([a-zA-Z_0-9-]+) .*$`)
	playerCountRegex = regexp.MustCompile(`(?m)^players[ ]*: ([0-9]+) \(([0-9]+) max\)$`)
	playerRegex      = regexp.MustCompile(`(?m)^#[ ]+([0-9]+)[ ]+"([^"]*)"[ ]+(STEAM_[0-1]:[0-1]:[0-9]+)[ ]+[0-9:]+[ ]+([0-9]+)[ ]+([0-9]+)[ ]+([a-z]+)[ ]+(([0-9]{1,3}.){3}[0-9]{1,3}):([0-9]+)$`)
)

func ParseHostname(input string) string {
	result := hostnameRegex.FindStringSubmatch(input)
	if len(result) > 1 {
		return result[1]
	} else {
		return ""
	}
}

func ParseVersion(input string) string {
	result := versionRegex.FindStringSubmatch(input)
	if len(result) > 1 {
		return result[1]
	} else {
		return ""
	}
}

func ParseMap(input string) string {
	result := mapRegex.FindStringSubmatch(input)
	if len(result) > 1 {
		return result[1]
	} else {
		return ""
	}
}

func ParsePlayerCount(input string) *models.PlayerCount {
	result := playerCountRegex.FindStringSubmatch(input)
	if len(result) > 2 {
		current, err := strconv.Atoi(result[1])
		if err != nil {
			panic(err)
		}
		max, err := strconv.Atoi(result[2])
		if err != nil {
			panic(err)
		}
		return &models.PlayerCount{
			Current: current,
			Max:     max,
		}
	} else {
		return &models.PlayerCount{}
	}
}

func ParsePlayers(input string) map[string]*models.Player {
	input = strings.Replace(input, "\000", "", -1)
	matches := playerRegex.FindAllStringSubmatch(input, -1)
	players := make(map[string]*models.Player)
	if len(matches) == 0 {
		return players
	}
	for _, m := range matches {
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
		players[m[3]] = &models.Player{
			Username: m[2],
			UserID:   userID,
			SteamID:  m[3],
			State:    m[6],
			Ping:     ping,
			Loss:     loss,
			IP:       m[7],
			ConnPort: connPort,
		}
	}
	return players
}
