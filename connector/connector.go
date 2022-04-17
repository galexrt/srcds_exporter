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
	"github.com/galexrt/srcds_exporter/config"
	"github.com/galexrt/srcds_exporter/connector/connections"
	"github.com/sirupsen/logrus"
)

// Connector struct contains the connections
type Connector struct {
	log         *logrus.Logger
	connections map[string]connections.IConnection
}

// NewConnector creates a new Connector object
func NewConnector(log *logrus.Logger) *Connector {
	return &Connector{
		log:         log,
		connections: make(map[string]connections.IConnection),
	}
}

// GetConnections holds all connections and reconnects/reopens them if necessary
func (cn *Connector) GetConnections() (map[string]connections.IConnection, error) {
	return cn.connections, nil
}

// NewConnection Add a new connection and initiates first contact connection
func (cn *Connector) NewConnection(name string, opts *connections.ConnectionOptions) error {
	if _, ok := cn.connections[opts.Addr]; ok {
		return nil
	}
	if opts.Mode == config.RCONMode {
		cn.connections[opts.Addr] = connections.NewRCON(name, opts)
	} else {
		cn.connections[opts.Addr] = connections.NewServerQuery(name, opts, cn.log)
	}

	return nil
}

// CloseAll closes all open connections
func (cn *Connector) CloseAll() {
	for _, con := range cn.connections {
		con.Close()
	}
}
