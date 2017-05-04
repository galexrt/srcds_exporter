package models

// Status Contains the server status
type Status struct {
	Hostname    string
	Version     string
	Map         string
	PlayerCount PlayerCount
	Players     map[int]Player
}
