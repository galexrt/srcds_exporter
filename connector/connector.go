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

package connector

import (
	"time"

	rcon "github.com/galexrt/go-rcon"
	cache "github.com/patrickmn/go-cache"
)

// Connector struct contains the connections
type Connector struct {
	connections map[string]*Connection
}

// NewConnector creates a new Connector object
func NewConnector() *Connector {
	return &Connector{
		connections: make(map[string]*Connection),
	}
}

// GetConnections holds all connections and reconnects/reopens them if necessary
func (cn *Connector) GetConnections() (map[string]*Connection, error) {
	return cn.connections, nil
}

// NewConnection Add a new connection and initiates first contact connection
func (cn *Connector) NewConnection(name string, opts *ConnectionOptions) error {
	if _, ok := cn.connections[opts.Addr]; ok {
		return nil
	}
	con, err := rcon.Connect(opts.Addr,
		&rcon.ConnectOptions{
			RCONPassword: opts.RCONPassword,
			Timeout:      opts.ConnectTimeout,
		})
	if err != nil {
		return err
	}
	cn.connections[opts.Addr] = &Connection{
		Name:  name,
		con:   con,
		cache: *cache.New(opts.CacheExpiration, opts.CacheCleanupInterval),
		opts: map[string]string{
			"Address":      opts.Addr,
			"RCONPassword": opts.RCONPassword,
			"Timeout":      opts.ConnectTimeout.String(),
		},
		created: time.Now().Add(opts.ConnectTimeout),
	}
	return nil
}

// CloseAll closes all open connections
func (cn *Connector) CloseAll() {
	for _, con := range cn.connections {
		con.Close()
	}
}
