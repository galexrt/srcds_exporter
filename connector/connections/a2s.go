/*
Copyright 2022 Alexander Trost <galexrt@googlemail.com>

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

package connections

import (
	"sync"
	"time"

	"github.com/galexrt/srcds_exporter/parser/models"
	"github.com/patrickmn/go-cache"
	a2s "github.com/rumblefrog/go-a2s"
	"github.com/sirupsen/logrus"
)

// A2S is a connection using the Valve A2S query protocol (via go-a2s)
type A2S struct {
	log     *logrus.Entry
	opts    *ConnectionOptions
	cache   *cache.Cache
	client  *a2s.Client
	cmu     sync.Mutex
	created time.Time
}

// NewA2S creates a new A2S based IConnection
func NewA2S(name string, opts *ConnectionOptions, log *logrus.Logger) IConnection {
	return &A2S{
		log:     log.WithFields(logrus.Fields{"server": name}),
		opts:    opts,
		cache:   cache.New(opts.CacheExpiration, opts.CacheCleanupInterval),
		created: time.Time{},
	}
}

func (c *A2S) Reconnect() error {
	client, err := a2s.NewClient(c.opts.Addr, a2s.TimeoutOption(c.opts.ConnectTimeout))
	if err != nil {
		return err
	}

	if c.client != nil {
		c.client.Close()
	}
	c.client = client
	c.created = time.Now()

	return nil
}

// Close closes the A2S connection
func (c *A2S) Close() {
	if c.client != nil {
		c.client.Close()
	}
}

func (c *A2S) ensureConnected() error {
	if c.client == nil || (time.Now().Unix()-c.created.Unix()) > 5 {
		return c.Reconnect()
	}
	return nil
}

func (c *A2S) getInfo() (*a2s.ServerInfo, error) {
	c.cmu.Lock()
	defer c.cmu.Unlock()

	out, found := c.cache.Get("info")
	if !found {
		if err := c.ensureConnected(); err != nil {
			return nil, err
		}

		info, err := c.client.QueryInfo()
		if err != nil {
			return nil, err
		}
		c.cache.Add("info", info, cache.DefaultExpiration)
		out = info
	}

	return out.(*a2s.ServerInfo), nil
}

// GetMap return map of server
func (c *A2S) GetMap() (string, error) {
	info, err := c.getInfo()
	if err != nil {
		return "", err
	}

	return info.Map, nil
}

// GetPlayerCount return server player count
func (c *A2S) GetPlayerCount() (*models.PlayerCount, error) {
	info, err := c.getInfo()
	if err != nil {
		return nil, err
	}

	return &models.PlayerCount{
		Current: int(info.Players),
		Max:     int(info.MaxPlayers),
		Bots:    int(info.Bots),
		Humans:  int(info.Players) - int(info.Bots),
	}, nil
}

// GetPlayers return the players connected to the server.
//
// The A2S_PLAYER query does not expose SteamIDs, ping or packet loss, so
// players are keyed by name and those fields are left at their zero value.
func (c *A2S) GetPlayers() (map[string]*models.Player, error) {
	c.cmu.Lock()
	defer c.cmu.Unlock()

	out, found := c.cache.Get("players")
	if !found {
		if err := c.ensureConnected(); err != nil {
			return nil, err
		}

		playerInfo, err := c.client.QueryPlayer()
		if err != nil {
			return nil, err
		}

		players := make(map[string]*models.Player, len(playerInfo.Players))
		for _, p := range playerInfo.Players {
			players[p.Name] = &models.Player{
				Username: p.Name,
				UserID:   int(p.Index),
			}
		}
		c.cache.Add("players", players, cache.DefaultExpiration)
		out = players
	}

	return out.(map[string]*models.Player), nil
}
