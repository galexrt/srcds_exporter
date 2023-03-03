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

package parser

import (
	"errors"
	"regexp"
	"strconv"
	"strings"

	"github.com/galexrt/srcds_exporter/parser/models"
)

var (
	hostnameRegex    = regexp.MustCompile(`(?m)^hostname\s*: (.*)$`)
	versionRegex     = regexp.MustCompile(`(?m)^version\s*: (.*)$`)
	mapRegex         = regexp.MustCompile(`(?m)^map\s*: ([a-zA-Z_0-9-]+)( .*)?$`)
	playerCountRegex = regexp.MustCompile(`(?m)^players\s*:\s*((?P<current1>[0-9]+)\s*\((?P<max1>[0-9]+)\s*max\)|(?P<humans>[0-9]+) humans,\s+(?P<bots>[0-9]+) bots\s+\((?P<max2>[0-9]+)(/[0-9]+)?\s+max\)).*$`)
	playerRegex      = regexp.MustCompile(`(?m)^#\s+(?P<userid>[0-9]+)(\s+\d+)?\s+"(?P<username>[^"]*)"\s+(?P<steamid>\S+)\s+(?P<connected>[0-9:]+)\s+(?P<ping>[0-9]+)\s+(?P<loss>[0-9]+)\s+(?P<state>[a-z]+)(\s+\d+)?(\s+(?P<ip>([0-9]{1,3}.){3}[0-9]{1,3}):(?P<connport>[0-9]+))?$`)
)

// ParseHostname parse SRCDS `status` command to retrieve server hostname
func ParseHostname(input string) string {
	result := hostnameRegex.FindStringSubmatch(input)
	if len(result) > 1 {
		return result[1]
	}
	return ""
}

// ParseVersion parse SRCDS `status` command to retrieve server version
func ParseVersion(input string) string {
	result := versionRegex.FindStringSubmatch(input)
	if len(result) > 1 {
		return result[1]
	}
	return ""
}

// ParseMap parse SRCDS `status` command to retrieve server map
func ParseMap(input string) string {
	result := mapRegex.FindStringSubmatch(input)
	if len(result) > 1 {
		return result[1]
	}
	return ""
}

// ParsePlayerCount parse SRCDS `status` command to retrieve player count
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
	}
	return nil, errors.New("no player count found in input")
}

// ParsePlayers parse SRCDS `status` command to retrieve players on server
func ParsePlayers(input string) (map[string]*models.Player, error) {
	input = strings.Replace(input, "\000", "", -1)

	matches := playerRegex.FindAllStringSubmatch(input, -1)
	names := playerRegex.SubexpNames()
	md := make([]map[string]string, len(matches))
	for i, n := range matches {
		md[i] = map[string]string{}
		for k := 0; k < len(n); k++ {
			key := names[k]
			if key == "" {
				continue
			}
			md[i][key] = n[k]
		}
	}
	if len(matches) == 0 {
		return nil, errors.New("no matches found in input")
	}
	players := make(map[string]*models.Player)
	for _, m := range md {
		userID, _ := strconv.Atoi(m["userid"])
		ping, _ := strconv.Atoi(m["ping"])
		loss, _ := strconv.Atoi(m["loss"])
		connPort, _ := strconv.Atoi(m["connport"])
		players[m["steamid"]] = &models.Player{
			Username: m["username"],
			UserID:   userID,
			SteamID:  m["steamid"],
			State:    m["state"],
			Ping:     ping,
			Loss:     loss,
			IP:       m["ip"],
			ConnPort: connPort,
		}
	}
	return players, nil
}
