package main

import (
	"encoding/json"
)

// Client -> Server message types
const (
	MsgJoin    = "join"
	MsgLeave   = "leave"
	MsgInput   = "input"
	MsgCreate  = "create"  // create session
	MsgList    = "list"    // list sessions
	MsgCheck   = "check"   // check if session exists
	MsgControl = "control" // phone controller attach
)

// Server -> Client message types
const (
	MsgState    = "state"
	MsgWelcome  = "welcome"
	MsgDeath    = "death"
	MsgKill     = "kill"
	MsgSessions = "sessions"
	MsgJoined   = "joined"
	MsgCreated  = "created" // session created, client should navigate
	MsgError    = "error"
	MsgChecked    = "checked"    // session check response
	MsgControlOK  = "control_ok"  // controller attach confirmed
	MsgCtrlOn     = "ctrl_on"     // notify desktop: controller attached
	MsgCtrlOff    = "ctrl_off"    // notify desktop: controller detached
	MsgHit        = "hit"         // damage dealt to an entity
	MsgMobSay     = "mob_say"     // mob speech bubble
)

// Envelope wraps all outgoing messages with a type field
type Envelope struct {
	T    string      `json:"t"`
	Data interface{} `json:"d,omitempty"`
}

// InEnvelope is used for incoming messages — json.RawMessage avoids double-unmarshal
type InEnvelope struct {
	T string          `json:"t"`
	D json.RawMessage `json:"d,omitempty"`
}

// ClientInput is sent by the client at 20Hz
type ClientInput struct {
	MX    float64 `json:"mx"`    // mouse X (world coords)
	MY    float64 `json:"my"`    // mouse Y (world coords)
	Fire  bool    `json:"fire"`  // W key held
	Boost bool    `json:"boost"` // Shift key held
	Thresh float64 `json:"thresh"` // distance threshold for speed modulation
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
	ID   string  `json:"id" msgpack:"id"`
	Name string  `json:"n" msgpack:"n"`
	X    float64 `json:"x" msgpack:"x"`
	Y    float64 `json:"y" msgpack:"y"`
	R    float64 `json:"r" msgpack:"r"`
	VX   *float64 `json:"vx,omitempty" msgpack:"vx,omitempty"`
	VY   *float64 `json:"vy,omitempty" msgpack:"vy,omitempty"`
	HP   int     `json:"hp" msgpack:"hp"`
	MaxHP int    `json:"mhp" msgpack:"mhp"`
	Ship int     `json:"s" msgpack:"s"`
	Score int    `json:"sc" msgpack:"sc"`
	Alive bool   `json:"a" msgpack:"a"`
	Boost bool   `json:"b,omitempty" msgpack:"b,omitempty"`
}

// ProjectileState is broadcast per projectile
type ProjectileState struct {
	ID string  `json:"id" msgpack:"id"`
	X  float64 `json:"x" msgpack:"x"`
	Y  float64 `json:"y" msgpack:"y"`
	R  float64 `json:"r" msgpack:"r"`
	Owner string `json:"o" msgpack:"o"`
}

// MobState is broadcast per mob
type MobState struct {
	ID    string   `json:"id" msgpack:"id"`
	X     float64  `json:"x" msgpack:"x"`
	Y     float64  `json:"y" msgpack:"y"`
	R     float64  `json:"r" msgpack:"r"`
	VX    *float64 `json:"vx,omitempty" msgpack:"vx,omitempty"`
	VY    *float64 `json:"vy,omitempty" msgpack:"vy,omitempty"`
	HP    int      `json:"hp" msgpack:"hp"`
	MaxHP int      `json:"mhp" msgpack:"mhp"`
	Alive bool     `json:"a" msgpack:"a"`
}

// AsteroidState is broadcast per asteroid
type AsteroidState struct {
	ID string  `json:"id" msgpack:"id"`
	X  float64 `json:"x" msgpack:"x"`
	Y  float64 `json:"y" msgpack:"y"`
	R  float64 `json:"r" msgpack:"r"`
}

// PickupState is broadcast per pickup
type PickupState struct {
	ID string  `json:"id" msgpack:"id"`
	X  float64 `json:"x" msgpack:"x"`
	Y  float64 `json:"y" msgpack:"y"`
}

// GameState is the full state broadcast
type GameState struct {
	Players     []PlayerState     `json:"p" msgpack:"p"`
	Projectiles []ProjectileState `json:"pr" msgpack:"pr"`
	Mobs        []MobState        `json:"m" msgpack:"m"`
	Asteroids   []AsteroidState   `json:"a" msgpack:"a"`
	Pickups     []PickupState     `json:"pk" msgpack:"pk"`
	Tick        uint64            `json:"tick" msgpack:"tick"`
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

// ControlMsg is sent by a phone controller to attach to a player
type ControlMsg struct {
	SID      string `json:"sid"`
	PlayerID string `json:"pid"`
}

// CheckMsg is sent by client to check if a session exists
type CheckMsg struct {
	SID string `json:"sid"`
}

// CheckedMsg is the response to a session check
type CheckedMsg struct {
	SID     string `json:"sid"`
	Exists  bool   `json:"exists"`
	Name    string `json:"name,omitempty"`
	Players int    `json:"players,omitempty"`
}

// HitMsg is broadcast when damage is dealt
type HitMsg struct {
	X          float64 `json:"x"`
	Y          float64 `json:"y"`
	Dmg        int     `json:"dmg"`
	VictimID   string  `json:"vid"`
	AttackerID string  `json:"aid"`
}

// MobSayMsg is broadcast when a mob says a phrase
type MobSayMsg struct {
	MobID string `json:"mid"`
	Text  string `json:"text"`
}
