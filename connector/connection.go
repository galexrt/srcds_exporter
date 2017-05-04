package connector

import (
	"sync"
	"time"

	steam "github.com/galexrt/go-steam"
	cache "github.com/patrickmn/go-cache"
)

// Connection struct contains all variables necessary for the connection
type Connection struct {
	Name    string
	cmu     sync.Mutex
	con     *steam.Server
	mu      sync.Mutex
	cache   cache.Cache
	opts    map[string]string
	created time.Time
}

func (c *Connection) reconnect() error {
	con, err := steam.Connect(c.opts["Address"], &steam.ConnectOptions{
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
