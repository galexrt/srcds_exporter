package models

// Player contains player information like username, steamID, etc.
type Player struct {
	Username string
	UserID   int
	SteamID  string
	State    string
	Ping     int
	Loss     int
	IP       string
	ConnPort int
}
