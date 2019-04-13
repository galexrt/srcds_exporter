package parser

import (
	"errors"
	"regexp"
	"strconv"
	"strings"

	"github.com/galexrt/srcds_exporter/parser/models"
)

var (
	hostnameRegex    = regexp.MustCompile(`(?m)^hostname[ ]*: (.*)$`)
	versionRegex     = regexp.MustCompile(`(?m)^version[ ]*: (.*)$`)
	mapRegex         = regexp.MustCompile(`(?m)^map[ ]*: ([a-zA-Z_0-9-]+) .*$`)
	playerCountRegex = regexp.MustCompile(`(?m)^players[ ]*:[ ]*((?P<current1>[0-9]+)[ ]*\((?P<max1>[0-9]+)[ ]*max\)|(?P<humans>[0-9]+) humans,[ ]+(?P<bots>[0-9]+) bots[ ]+\((?P<max2>[0-9]+)/[0-9]+[ ]+max\)).*$`)
	playerRegex      = regexp.MustCompile(`(?m)^#[ ]+([0-9]+)[ ]+"([^"]*)"[ ]+(STEAM_[0-1]:[0-1]:[0-9]+)[ ]+[0-9:]+[ ]+([0-9]+)[ ]+([0-9]+)[ ]+([a-z]+)[ ]+(([0-9]{1,3}.){3}[0-9]{1,3}):([0-9]+)$`)
)

// ParseHostname
func ParseHostname(input string) string {
	result := hostnameRegex.FindStringSubmatch(input)
	if len(result) > 1 {
		return result[1]
	} else {
		return ""
	}
}

// ParseVersion
func ParseVersion(input string) string {
	result := versionRegex.FindStringSubmatch(input)
	if len(result) > 1 {
		return result[1]
	} else {
		return ""
	}
}

// ParseMap
func ParseMap(input string) string {
	result := mapRegex.FindStringSubmatch(input)
	if len(result) > 1 {
		return result[1]
	} else {
		return ""
	}
}

// ParsePlayerCount
func ParsePlayerCount(input string) (*models.PlayerCount, error) {
	match := playerCountRegex.FindStringSubmatch(input)

	result := make(map[string]string)
	for i, name := range playerCountRegex.SubexpNames() {
		if i != 0 && name != "" && len(match) >= i {
			result[name] = match[i]
		}
	}

	if len(result) > 0 {
		var currentRaw string
		if result["current1"] != "" {
			currentRaw = result["current1"]
		} else {
			currentRaw = "0"
		}

		var currentMax string
		if result["max1"] != "" {
			currentMax = result["max1"]
		} else if result["max2"] != "" {
			currentMax = result["max2"]
		} else {
			currentMax = "0"
		}

		current, _ := strconv.Atoi(currentRaw)
		max, _ := strconv.Atoi(currentMax)

		humansRaw := "-1"
		if result["humans"] != "" {
			humansRaw = result["humans"]
		}
		humans, _ := strconv.Atoi(humansRaw)

		botsRaw := "-1"
		if result["bots"] != "" {
			botsRaw = result["bots"]
		}
		bots, _ := strconv.Atoi(botsRaw)

		if bots != -1 && humans != -1 {
			current = bots + humans
		}

		return &models.PlayerCount{
			Current: current,
			Max:     max,
			Humans:  humans,
			Bots:    bots,
		}, nil
	} else {
		return nil, errors.New("no player count found in input")
	}
}

// ParsePlayers
func ParsePlayers(input string) (map[string]*models.Player, error) {
	input = strings.Replace(input, "\000", "", -1)
	matches := playerRegex.FindAllStringSubmatch(input, -1)
	if len(matches) == 0 {
		return nil, errors.New("no matches found in input")
	}
	players := make(map[string]*models.Player)
	for _, m := range matches {
		userID, _ := strconv.Atoi(m[1])
		ping, _ := strconv.Atoi(m[4])
		loss, _ := strconv.Atoi(m[5])
		connPort, _ := strconv.Atoi(m[9])
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
	return players, nil
}
