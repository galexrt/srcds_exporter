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
	"fmt"
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
			RCONPassword: opts.RconPassword,
			Timeout:      opts.ConnectTimeout,
		})
	if err != nil {
		return err
	}
	var (
		conTimeoutParsed   time.Duration
		cacheTimeoutParsed time.Duration
	)
	conTimeoutParsed, err = time.ParseDuration(opts.ConnectTimeout)
	if err != nil {
		return err
	}
	cacheTimeoutParsed, err = time.ParseDuration(opts.CacheTimeout)
	if err != nil {
		return err
	}
	fmt.Print(cacheTimeoutParsed)
	// TODO make cache time configurable?
	cn.connections[opts.Addr] = &Connection{
		Name:  name,
		con:   con,
		cache: *cache.New(cacheTimeoutParsed, 11*time.Second),
		opts: map[string]string{
			"Address":      opts.Addr,
			"RCONPassword": opts.RconPassword,
			"Timeout":      opts.ConnectTimeout,
		},
		created: time.Now().Add(conTimeoutParsed),
	}
	return nil
}

// CloseAll closes all open connections
func (cn *Connector) CloseAll() {
	for _, con := range cn.connections {
		con.Close()
	}
}
