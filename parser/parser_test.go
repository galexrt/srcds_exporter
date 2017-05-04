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
		`players : 16.12.01/24 6729 secure`,
		"16.12.01/24 6729 secure",
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
		`1 (64 max)`,
		&models.PlayerCount{
			Current: 1,
			Max:     64,
		},
	},
}

func TestParsePlayerCount(t *testing.T) {
	for _, tt := range parsePlayerCountTests {
		actual := ParsePlayerCount(tt.request)
		if !reflect.DeepEqual(actual, tt.expected) {
			t.Errorf("ParsePlayerCount(%s): expected %v, actual %v", tt.request, tt.expected, actual)
		}
	}
}

var parsePlayersTests = []struct {
	request  string
	expected map[int]models.Player
}{
	{
		`#    218 "TestUser1"      STEAM_0:0:1015738 07:36       65    0 active 10.10.220.12:27005`,
		map[int]models.Player{
			218: *&models.Player{
				Username: "TestUser1",
				SteamID:  "STEAM_0:0:1015738",
				Ping:     65,
				Loss:     0,
				State:    "active",
				IP:       "10.10.220.12",
				ConnPort: 27005,
			},
		},
	},
}

func TestParsePlayers(t *testing.T) {
	for _, tt := range parsePlayersTests {
		actual := ParsePlayers(tt.request)
		if !reflect.DeepEqual(actual, tt.expected) {
			t.Errorf("parsePlayers(%s): expected %v, actual %v", tt.request, tt.expected, actual)
		}
	}
}
