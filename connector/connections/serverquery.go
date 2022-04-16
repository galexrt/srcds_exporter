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

	"github.com/galexrt/srcds_exporter/parser/models"
	"github.com/patrickmn/go-cache"
	"github.com/xv-chang/rconGo/core"
)

type ServerQuery struct {
	opts  *ConnectionOptions
	cache *cache.Cache
	con   *core.ServerQuery
	cmu   sync.Mutex
}

func NewServerQuery(name string, opts *ConnectionOptions) IConnection {
	return &ServerQuery{
		opts:  opts,
		cache: cache.New(opts.CacheExpiration, opts.CacheCleanupInterval),
	}
}

func (c *ServerQuery) Reconnect() error {
	c.cmu.Lock()
	defer c.cmu.Unlock()
	if c.con != nil {
		c.con.Close()
	}
	c.con = core.NewServerQuery(c.opts.Addr)
	return nil
}

// Close closes the RCON connection
func (c *ServerQuery) Close() {
	c.con.Close()
}

// GetMap return map of server
func (c *ServerQuery) GetMap() (string, error) {
	if c.con == nil {
		c.Reconnect()
	}

	c.cmu.Lock()
	defer c.cmu.Unlock()

	return c.con.GetInfo().Map, nil
}

// GetPlayerCount return server player count
func (c *ServerQuery) GetPlayerCount() (*models.PlayerCount, error) {
	if c.con == nil {
		c.Reconnect()
	}

	c.cmu.Lock()
	defer c.cmu.Unlock()

	playerCount := &models.PlayerCount{
		Current: int(c.con.GetInfo().Players),
		Max:     int(c.con.GetInfo().MaxPlayers),
		Bots:    int(c.con.GetInfo().Bots),
		Humans:  int(c.con.GetInfo().Players),
	}

	return playerCount, nil
}

func (c *ServerQuery) GetPlayers() (map[string]*models.Player, error) {
	return nil, nil
}
