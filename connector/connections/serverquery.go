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
	"github.com/sirupsen/logrus"
	"github.com/xv-chang/rconGo/core"
)

type ServerQuery struct {
	log     *logrus.Entry
	opts    *ConnectionOptions
	cache   *cache.Cache
	con     *core.ServerQuery
	cmu     sync.Mutex
	created time.Time
}

func NewServerQuery(name string, opts *ConnectionOptions, log *logrus.Logger) IConnection {
	return &ServerQuery{
		log:     log.WithFields(logrus.Fields{"server": name}),
		opts:    opts,
		cache:   cache.New(opts.CacheExpiration, opts.CacheCleanupInterval),
		created: time.Now(),
	}
}

func (c *ServerQuery) Reconnect() error {
	if (time.Now().Unix() - c.created.Unix()) > 5 {
		c.cmu.Lock()
		defer c.cmu.Unlock()
		if c.con != nil {
			c.con.Close()
		}

		c.con = core.NewServerQuery(c.opts.Addr)
		c.con.Conn.SetDeadline(time.Now().Add(c.opts.ConnectTimeout))
		c.created = time.Now()
	}

	return nil
}

// Close closes the RCON connection
func (c *ServerQuery) Close() {
	c.con.Close()
}

func (c *ServerQuery) getInfo() *core.ServerInfo {
	defer func() {
		if recover() != nil {
			c.log.Errorf("got panic while connecting to server, reconnecting")
		}
	}()

	out, found := c.cache.Get("data")
	if !found {
		c.Reconnect()

		out = c.con.GetInfo()
		c.cache.Add("data", out, cache.DefaultExpiration)
	}

	return out.(*core.ServerInfo)
}

// GetMap return map of server
func (c *ServerQuery) GetMap() (string, error) {
	info := c.getInfo()
	if info == nil {
		return "", nil
	}

	return info.Map, nil
}

// GetPlayerCount return server player count
func (c *ServerQuery) GetPlayerCount() (*models.PlayerCount, error) {
	info := c.getInfo()
	if info == nil {
		return nil, nil
	}

	playerCount := &models.PlayerCount{
		Current: int(info.Players),
		Max:     int(info.MaxPlayers),
		Bots:    int(info.Bots),
		Humans:  int(info.Players),
	}

	return playerCount, nil
}

func (c *ServerQuery) GetPlayers() (map[string]*models.Player, error) {
	return nil, nil
}
