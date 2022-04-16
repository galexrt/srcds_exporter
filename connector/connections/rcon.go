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

	"github.com/galexrt/go-rcon"
	"github.com/galexrt/srcds_exporter/config"
	"github.com/galexrt/srcds_exporter/parser"
	"github.com/galexrt/srcds_exporter/parser/models"
	"github.com/patrickmn/go-cache"
)

type RCON struct {
	opts    *ConnectionOptions
	cache   *cache.Cache
	created time.Time
	rcon    *rcon.Server
	cmu     sync.Mutex
}

func NewRCON(name string, opts *ConnectionOptions) IConnection {
	return &RCON{
		opts:    opts,
		cache:   cache.New(opts.CacheExpiration, opts.CacheCleanupInterval),
		created: time.Now(),
	}
}

func (c *RCON) Reconnect() error {
	if c.opts.Mode != config.ServerQueryMode {
		rcon, err := rcon.Connect(c.opts.Addr, &rcon.ConnectOptions{
			RCONPassword: c.opts.RCONPassword,
			Timeout:      c.opts.ConnectTimeout,
		})
		if err != nil {
			return err
		}
		c.rcon = rcon
	}
	c.created = time.Now()
	return nil
}

// Close closes the RCON connection
func (c *RCON) Close() {
	c.rcon.Close()
}

// runRCONCommand run rcon command and return response
func (c *RCON) runRCONCommand(cmd string) (string, error) {
	c.cmu.Lock()
	defer c.cmu.Unlock()
	out, found := c.cache.Get(cmd)
	if !found {
		if (time.Now().Unix() - c.created.Unix()) > 5 {
			if err := c.Reconnect(); err != nil {
				return "", err
			}
			c.created = time.Now()
		}
		var err error
		if out, err = c.rcon.Send(cmd); err != nil {
			return "", err
		}
		c.cache.Add(cmd, out.(string), cache.DefaultExpiration)
	}
	return out.(string), nil
}

// GetMap return map of server
func (c *RCON) GetMap() (string, error) {
	resp, err := c.runRCONCommand("status")
	if err != nil {
		return "", err
	}
	mapName := parser.ParseMap(resp)
	return mapName, err
}

// GetPlayerCount return server player count
func (c *RCON) GetPlayerCount() (*models.PlayerCount, error) {
	resp, err := c.runRCONCommand("status")
	if err != nil {
		return nil, err
	}

	playerCount, err := parser.ParsePlayerCount(resp)
	if err != nil {
		return nil, err
	}

	return playerCount, nil
}

func (c *RCON) GetPlayers() (map[string]*models.Player, error) {
	resp, err := c.runRCONCommand("status")
	if err != nil {
		return nil, err
	}
	players, err := parser.ParsePlayers(resp)
	if err != nil {
		return nil, err
	}

	return players, nil
}
