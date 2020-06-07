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

package connector

import (
	"sync"
	"time"

	rcon "github.com/galexrt/go-rcon"
	cache "github.com/patrickmn/go-cache"
)

// ConnectionOptions options for a Connection
type ConnectionOptions struct {
	Addr           string
	RconPassword   string
	ConnectTimeout string
	CacheTimeout   string
}

// Connection struct contains all variables necessary for the connection
type Connection struct {
	Name    string
	cmu     sync.Mutex
	con     *rcon.Server
	mu      sync.Mutex
	cache   cache.Cache
	opts    map[string]string
	created time.Time
}

func (c *Connection) reconnect() error {
	con, err := rcon.Connect(c.opts["Address"], &rcon.ConnectOptions{
		RCONPassword: c.opts["RCONPassword"],
		Timeout:      c.opts["Timeout"],
	})
	if err != nil {
		return err
	}
	c.con = con
	c.created = time.Now()
	return nil
}

// Get return rcon command response
func (c *Connection) Get(cmd string) (string, error) {
	c.cmu.Lock()
	defer c.cmu.Unlock()
	out, found := c.cache.Get(cmd)
	if !found {
		if (time.Now().Unix() - c.created.Unix()) > 5 {
			if err := c.reconnect(); err != nil {
				return "", err
			}
			c.created = time.Now()
		}
		var err error
		if out, err = c.con.Send(cmd); err != nil {
			return "", err
		}
		c.cache.Add(cmd, out.(string), cache.DefaultExpiration)
	}
	return out.(string), nil
}

// Close closes a single connection
func (c *Connection) Close() {
	c.con.Close()
}
