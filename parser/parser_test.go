package parser

import (
	"reflect"
	"testing"

	"github.com/galexrt/srcds_exporter/parser/models"
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
		if actual != tt.expected {
			t.Errorf("parseHostname(%s): expected %s, actual %s", tt.request, tt.expected, actual)
		}
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
		if actual != tt.expected {
			t.Errorf("parseVersion(%s): expected %s, actual %s", tt.request, tt.expected, actual)
		}
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
		if actual != tt.expected {
			t.Errorf("parseMap(%s): expected %s, actual %s", tt.request, tt.expected, actual)
		}
	}
}

var parsePlayerCountTests = []struct {
	request  string
	expected *models.PlayerCount
}{
	{
		`players : 1 (64 max)`,
		&models.PlayerCount{
			Current: 1,
			Max:     64,
		},
	},
	{
		`nope: nope`,
		nil,
	},
	{
		`players : 2 humans, 2 bots (26/0 max) (not hibernating)`,
		&models.PlayerCount{
			Current: 26,
			Max:     0,
		},
	},
}

func TestParsePlayerCount(t *testing.T) {
	for _, tt := range parsePlayerCountTests {
		actual, _ := ParsePlayerCount(tt.request)
		if !reflect.DeepEqual(actual, tt.expected) {
			t.Errorf("ParsePlayerCount(%s): expected %v, actual %v", tt.request, tt.expected, actual)
		}
	}
}

var parsePlayersTests = []struct {
	request  string
	expected map[string]*models.Player
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
	},
	{
		`NOPE`,
		nil,
	},
}

func TestParsePlayers(t *testing.T) {
	for _, tt := range parsePlayersTests {
		actual, _ := ParsePlayers(tt.request)
		if !reflect.DeepEqual(actual, tt.expected) {
			t.Errorf("parsePlayers(%s): expected %v, actual %v", tt.request, tt.expected, actual)
		}
	}
}
