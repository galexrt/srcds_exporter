package collector

import (
	"log"

	"github.com/galexrt/srcds_exporter/connector"
)

func getConnections() map[string]*connector.Connection {
	connections, err := connections.GetConnections()
	if err != nil {
		log.Fatal(err)
	}
	return connections
}
