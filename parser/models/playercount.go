package models

// PlayerCount contains current and max players of server
type PlayerCount struct {
	Current int
	Max     int
	Humans  int
	Bots    int
}
