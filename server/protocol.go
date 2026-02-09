package main

// Client -> Server message types
const (
	MsgJoin   = "join"
	MsgLeave  = "leave"
	MsgInput  = "input"
	MsgCreate = "create" // create session
	MsgList   = "list"   // list sessions
)

// Server -> Client message types
const (
	MsgState    = "state"
	MsgWelcome  = "welcome"
	MsgDeath    = "death"
	MsgKill     = "kill"
	MsgSessions = "sessions"
	MsgJoined   = "joined"
	MsgError    = "error"
)

// Envelope wraps all messages with a type field
type Envelope struct {
	T    string      `json:"t"`
	Data interface{} `json:"d,omitempty"`
}

// ClientInput is sent by the client at 20Hz
type ClientInput struct {
	MX    float64 `json:"mx"`    // mouse X (world coords)
	MY    float64 `json:"my"`    // mouse Y (world coords)
	Fire  bool    `json:"fire"`  // W key held
	Boost bool    `json:"boost"` // Shift key held
}

// JoinMsg is sent when player wants to join a session
type JoinMsg struct {
	Name      string `json:"name"`
	SessionID string `json:"sid"`
}

// CreateMsg is sent when player wants to create a session
type CreateMsg struct {
	Name        string `json:"name"`
	SessionName string `json:"sname"`
}

// PlayerState is broadcast per player each tick
type PlayerState struct {
	ID   string  `json:"id"`
	Name string  `json:"n"`
	X    float64 `json:"x"`
	Y    float64 `json:"y"`
	R    float64 `json:"r"`  // rotation radians
	VX   float64 `json:"vx"` // velocity X
	VY   float64 `json:"vy"` // velocity Y
	HP   int     `json:"hp"`
	MaxHP int    `json:"mhp"`
	Ship int     `json:"s"`  // ship type 0-3
	Score int    `json:"sc"`
	Alive bool   `json:"a"`
}

// ProjectileState is broadcast per projectile
type ProjectileState struct {
	ID string  `json:"id"`
	X  float64 `json:"x"`
	Y  float64 `json:"y"`
	R  float64 `json:"r"`
	Owner string `json:"o"`
}

// GameState is the full state broadcast
type GameState struct {
	Players     []PlayerState     `json:"p"`
	Projectiles []ProjectileState `json:"pr"`
	Tick        uint64            `json:"tick"`
}

// WelcomeMsg is sent to a player when they join
type WelcomeMsg struct {
	ID   string `json:"id"`
	Ship int    `json:"s"`
}

// DeathMsg notifies a player they died
type DeathMsg struct {
	KillerID   string `json:"kid"`
	KillerName string `json:"kn"`
}

// KillMsg is broadcast to all players in session
type KillMsg struct {
	KillerID   string `json:"kid"`
	KillerName string `json:"kn"`
	VictimID   string `json:"vid"`
	VictimName string `json:"vn"`
}

// SessionInfo is used in the session list
type SessionInfo struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Players int    `json:"players"`
}

// ErrorMsg sends error to client
type ErrorMsg struct {
	Msg string `json:"msg"`
}
