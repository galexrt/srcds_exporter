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

package parser

import (
	"testing"

	"github.com/galexrt/srcds_exporter/parser/models"
	"github.com/stretchr/testify/assert"
)

var parseHostnameTests = []struct {
	request  string
	expected string
}{
	{
		`hostname: Example server`,
		"Example server",
	},
	{
		`hostname: [TEST] ÜÄÖÜ server`,
		"[TEST] ÜÄÖÜ server",
	},
	{
		`nope: nope`,
		"",
	},
}

func TestParseHostname(t *testing.T) {
	for _, tt := range parseHostnameTests {
		actual := ParseHostname(tt.request)
		assert.Equal(t, tt.expected, actual)
	}
}

var parseVersionTests = []struct {
	request  string
	expected string
}{
	{
		`version : 16.12.01/24 6729 secure`,
		"16.12.01/24 6729 secure",
	},
	{
		`nope: nope`,
		"",
	},
}

func TestParseVersion(t *testing.T) {
	for _, tt := range parseVersionTests {
		actual := ParseVersion(tt.request)
		assert.Equal(t, tt.expected, actual)
	}
}

var parseMapTests = []struct {
	request  string
	expected string
}{
	{
		`map     : rp_retribution_v2 at: 0 x, 0 y, 0 z`,
		"rp_retribution_v2",
	},
	{
		`nope: nope`,
		"",
	},
}

func TestParseMap(t *testing.T) {
	for _, tt := range parseMapTests {
		actual := ParseMap(tt.request)
		assert.Equal(t, tt.expected, actual)
	}
}

var parsePlayerCountTests = []struct {
	request  string
	expected *models.PlayerCount
	errOkay  bool
}{
	{
		`players : 1 (64 max)`,
		&models.PlayerCount{
			Current: 1,
			Max:     64,
			Humans:  -1,
			Bots:    -1,
		},
		false,
	},
	{
		`nope: nope`,
		nil,
		true,
	},
	{
		`players : 2 humans, 2 bots (26/0 max) (not hibernating)`,
		&models.PlayerCount{
			Current: 4,
			Max:     26,
			Humans:  2,
			Bots:    2,
		},
		false,
	},
	{
		`players : 2 humans, 2 bots (4 max)`,
		&models.PlayerCount{
			Current: 4,
			Max:     4,
			Humans:  2,
			Bots:    2,
		},
		false,
	},
}

func TestParsePlayerCount(t *testing.T) {
	for _, tt := range parsePlayerCountTests {
		actual, err := ParsePlayerCount(tt.request)
		if tt.errOkay {
			assert.Error(t, err)
		} else {
			assert.NoError(t, err)
		}
		assert.Equal(t, tt.expected, actual)
	}
}

var parsePlayersTests = []struct {
	request  string
	expected map[string]*models.Player
	errOkay  bool
}{
	{
		`#    218 "TestUser1"      STEAM_0:0:1015738 07:36       65    0 active 10.10.220.12:27005`,
		map[string]*models.Player{
			"STEAM_0:0:1015738": &models.Player{
				Username: "TestUser1",
				SteamID:  "STEAM_0:0:1015738",
				UserID:   218,
				Ping:     65,
				Loss:     0,
				State:    "active",
				IP:       "10.10.220.12",
				ConnPort: 27005,
			},
		},
		false,
	},
	{
		`NOPE`,
		nil,
		true,
	},
	{
		`#    5 "TestUser2"      [U:1:1234567]      00:11       74    0 active 192.168.1.5:27005`,
		map[string]*models.Player{
			"[U:1:1234567]": &models.Player{
				Username: "TestUser2",
				SteamID:  "[U:1:1234567]",
				UserID:   5,
				Ping:     74,
				Loss:     0,
				State:    "active",
				IP:       "192.168.1.5",
				ConnPort: 27005,
			},
		},
		false,
	},
	{
		`#    5 "TestUser2"      [U:1:1234567]      00:11       74    0 active`,
		map[string]*models.Player{
			"[U:1:1234567]": &models.Player{
				Username: "TestUser2",
				SteamID:  "[U:1:1234567]",
				UserID:   5,
				Ping:     74,
				Loss:     0,
				State:    "active",
				IP:       "",
				ConnPort: 0,
			},
		},
		false,
	},
}

func TestParsePlayers(t *testing.T) {
	for _, tt := range parsePlayersTests {
		actual, err := ParsePlayers(tt.request)
		if tt.errOkay {
			assert.Error(t, err)
		} else {
			assert.NoError(t, err)
		}
		assert.Equal(t, tt.expected, actual)
	}
}
