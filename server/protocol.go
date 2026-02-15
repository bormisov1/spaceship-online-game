package main

import (
	"encoding/json"
)

// Client -> Server message types
const (
	MsgJoin     = "join"
	MsgLeave    = "leave"
	MsgInput    = "input"
	MsgCreate   = "create"     // create session
	MsgList     = "list"       // list sessions
	MsgCheck    = "check"      // check if session exists
	MsgControl  = "control"    // phone controller attach
	MsgReady    = "ready"      // player ready toggle
	MsgTeamPick = "team_pick"  // player picks team
	MsgRematch  = "rematch"    // request rematch
	MsgRegister    = "register"    // create account
	MsgLogin       = "login"       // login
	MsgAuth        = "auth"        // auth with token
	MsgProfile     = "profile"     // request profile
	MsgLeaderboard   = "leaderboard"    // request leaderboard
	MsgFriendAdd     = "friend_add"     // send friend request
	MsgFriendAccept  = "friend_accept"  // accept friend request
	MsgFriendDecline = "friend_decline" // decline friend request
	MsgFriendRemove  = "friend_remove"  // remove friend
	MsgFriendList    = "friend_list"    // request friends list
	MsgChat          = "chat"           // send chat message
)

// Server -> Client message types
const (
	MsgState       = "state"
	MsgWelcome     = "welcome"
	MsgDeath       = "death"
	MsgKill        = "kill"
	MsgSessions    = "sessions"
	MsgJoined      = "joined"
	MsgCreated     = "created"      // session created, client should navigate
	MsgError       = "error"
	MsgChecked     = "checked"      // session check response
	MsgControlOK   = "control_ok"   // controller attach confirmed
	MsgCtrlOn      = "ctrl_on"      // notify desktop: controller attached
	MsgCtrlOff     = "ctrl_off"     // notify desktop: controller detached
	MsgHit         = "hit"          // damage dealt to an entity
	MsgMobSay      = "mob_say"      // mob speech bubble
	MsgAuthOK         = "auth_ok"         // auth success
	MsgProfileData    = "profile_data"    // profile/stats response
	MsgXPUpdate        = "xp_update"        // XP gained after match
	MsgLeaderboardRes  = "leaderboard_res"  // leaderboard response
	MsgAchievementUnlock = "achievement"    // achievement unlocked
	MsgFriendListRes     = "friend_list_res" // friends list response
	MsgFriendNotify      = "friend_notify"   // friend request notification
	MsgChatMsg           = "chat_msg"        // chat message broadcast
	MsgMatchPhase     = "match_phase"     // match phase changed
	MsgMatchResult = "match_result" // match ended, results
	MsgTeamUpdate  = "team_update"  // team roster/score update
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
	MX      float64 `json:"mx"`      // mouse X (world coords)
	MY      float64 `json:"my"`      // mouse Y (world coords)
	Fire    bool    `json:"fire"`    // W key held
	Boost   bool    `json:"boost"`   // Shift key held
	Thresh  float64 `json:"thresh"`  // distance threshold for speed modulation
	Ability bool    `json:"ability"` // ability key pressed
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
	Mode        int    `json:"mode,omitempty"`
}

// PlayerState is broadcast per player each tick
type PlayerState struct {
	ID    string   `json:"id" msgpack:"id"`
	Name  string   `json:"n" msgpack:"n"`
	X     float64  `json:"x" msgpack:"x"`
	Y     float64  `json:"y" msgpack:"y"`
	R     float64  `json:"r" msgpack:"r"`
	VX    *float64 `json:"vx,omitempty" msgpack:"vx,omitempty"`
	VY    *float64 `json:"vy,omitempty" msgpack:"vy,omitempty"`
	HP    int      `json:"hp" msgpack:"hp"`
	MaxHP int      `json:"mhp" msgpack:"mhp"`
	Ship  int      `json:"s" msgpack:"s"`
	Score int      `json:"sc" msgpack:"sc"`
	Alive bool     `json:"a" msgpack:"a"`
	Boost   bool    `json:"b,omitempty" msgpack:"b,omitempty"`
	Team    int     `json:"tm,omitempty" msgpack:"tm,omitempty"`
	Class   int     `json:"cl,omitempty" msgpack:"cl,omitempty"`
	AbilCD  float64 `json:"acd,omitempty" msgpack:"acd,omitempty"`
	AbilAct bool    `json:"aact,omitempty" msgpack:"aact,omitempty"`
	SpawnP  bool    `json:"sp,omitempty" msgpack:"sp,omitempty"`
}

// ProjectileState is broadcast per projectile
type ProjectileState struct {
	ID    string  `json:"id" msgpack:"id"`
	X     float64 `json:"x" msgpack:"x"`
	Y     float64 `json:"y" msgpack:"y"`
	R     float64 `json:"r" msgpack:"r"`
	Owner string  `json:"o" msgpack:"o"`
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
	Ship  int      `json:"s" msgpack:"s"`
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
	MatchPhase  int               `json:"mp,omitempty" msgpack:"mp,omitempty"`
	TimeLeft    float64           `json:"tl,omitempty" msgpack:"tl,omitempty"`
	TeamRedSc   int               `json:"trs,omitempty" msgpack:"trs,omitempty"`
	TeamBlueSc  int               `json:"tbs,omitempty" msgpack:"tbs,omitempty"`
	HealZones   []HealZoneState   `json:"hz,omitempty" msgpack:"hz,omitempty"`
}

// HealZoneState is broadcast per heal zone
type HealZoneState struct {
	ID string  `json:"id" msgpack:"id"`
	X  float64 `json:"x" msgpack:"x"`
	Y  float64 `json:"y" msgpack:"y"`
	R  float64 `json:"r" msgpack:"r"` // radius
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
	Mode    int    `json:"mode,omitempty"`
	Phase   int    `json:"phase,omitempty"`
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

// MatchPhaseMsg is sent when match phase changes
type MatchPhaseMsg struct {
	Phase     int     `json:"phase"`
	Mode      int     `json:"mode"`
	TimeLeft  float64 `json:"time_left,omitempty"`
	Countdown float64 `json:"countdown,omitempty"`
}

// TeamPickMsg is sent by client to pick a team
type TeamPickMsg struct {
	Team int `json:"team"`
}

// MatchResultMsg is sent when a match ends
type MatchResultMsg struct {
	WinnerTeam int                 `json:"winner_team"`
	Players    []PlayerMatchResult `json:"players"`
	Duration   float64             `json:"duration"`
}

// PlayerMatchResult holds end-of-match stats for one player
type PlayerMatchResult struct {
	ID      string `json:"id"`
	Name    string `json:"n"`
	Team    int    `json:"tm"`
	Kills   int    `json:"k"`
	Deaths  int    `json:"d"`
	Assists int    `json:"a"`
	Score   int    `json:"sc"`
	MVP     bool   `json:"mvp,omitempty"`
}

// TeamUpdateMsg is sent to update team rosters
type TeamUpdateMsg struct {
	Red  []TeamPlayerInfo `json:"red"`
	Blue []TeamPlayerInfo `json:"blue"`
}

// TeamPlayerInfo holds info about a player on a team
type TeamPlayerInfo struct {
	ID    string `json:"id"`
	Name  string `json:"n"`
	Ready bool   `json:"ready"`
}

// RegisterMsg is sent by client to create an account
type RegisterMsg struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// LoginMsg is sent by client to log in
type LoginMsg struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// AuthMsg is sent by client to authenticate with a stored token
type AuthMsg struct {
	Token string `json:"token"`
}

// AuthOKMsg is sent to client on successful auth
type AuthOKMsg struct {
	Token    string `json:"token"`
	Username string `json:"username"`
	PlayerID int64  `json:"pid"`
	IsGuest  bool   `json:"guest,omitempty"`
}

// ProfileDataMsg is sent to client with profile/stats info
type ProfileDataMsg struct {
	Username string  `json:"username"`
	Level    int     `json:"level"`
	XP       int     `json:"xp"`
	XPNext   int     `json:"xp_next"` // XP needed for next level
	Kills    int     `json:"kills"`
	Deaths   int     `json:"deaths"`
	Wins     int     `json:"wins"`
	Losses   int     `json:"losses"`
	Playtime float64 `json:"playtime"`
}

// XPUpdateMsg is sent to a player after a match with XP gain details
type XPUpdateMsg struct {
	XPGained  int  `json:"xp_gained"`
	TotalXP   int  `json:"total_xp"`
	Level     int  `json:"level"`
	PrevLevel int  `json:"prev_level"`
	XPNext    int  `json:"xp_next"` // XP needed for next level
	LeveledUp bool `json:"leveled_up,omitempty"`
}

// LeaderboardMsg is sent to client with leaderboard data
type LeaderboardMsg struct {
	Entries []LeaderboardEntry `json:"entries"`
}

// AchievementMsg is sent when a player unlocks an achievement
type AchievementMsg struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"desc"`
}

// FriendActionMsg is sent by client for friend add/accept/decline/remove
type FriendActionMsg struct {
	Username string `json:"username"` // target username
}

// FriendInfo represents a friend in the friend list
type FriendInfo struct {
	Username string `json:"username"`
	Level    int    `json:"level"`
	Online   bool   `json:"online"`
	Status   int    `json:"status"` // 0=pending, 1=accepted
}

// FriendListMsg is sent to client with friends list
type FriendListMsg struct {
	Friends  []FriendInfo `json:"friends"`
	Requests []FriendInfo `json:"requests"` // incoming pending requests
}

// FriendNotifyMsg notifies a player about a friend event
type FriendNotifyMsg struct {
	Type     string `json:"type"` // "request", "accepted"
	Username string `json:"username"`
}

// ChatSendMsg is sent by client to send a chat message
type ChatSendMsg struct {
	Text string `json:"text"`
	Team bool   `json:"team"` // team-only chat
}

// ChatBroadcastMsg is sent to clients with a chat message
type ChatBroadcastMsg struct {
	From string `json:"from"`
	Text string `json:"text"`
	Team bool   `json:"team"`
}
