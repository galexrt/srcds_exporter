package connector

import (
	"time"

	steam "github.com/galexrt/go-steam"
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
func (cn *Connector) NewConnection(name, addr, rconPassword string, connectTimeout string) error {
	if _, ok := cn.connections[addr]; ok {
		return nil
	}
	con, err := steam.Connect(addr, &steam.ConnectOptions{
		RCONPassword: rconPassword,
		Timeout:      connectTimeout,
	})
	if err != nil {
		return err
	}
	cn.connections[addr] = &Connection{
		Name: name,
		con:  con,
		// TODO make cache time configurable?
		cache: *cache.New(10*time.Second, 11*time.Second),
		opts: map[string]string{
			"Address":      addr,
			"RCONPassword": rconPassword,
			"Timeout":      connectTimeout,
		},
		// TODO make time configurable? use connectTimeout
		created: time.Now().Add(2 * time.Second),
	}
	return nil
}

// CloseAll closes all open connections
func (cn *Connector) CloseAll() {
	for _, con := range cn.connections {
		con.Close()
	}
}
