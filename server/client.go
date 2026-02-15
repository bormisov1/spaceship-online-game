package main

import (
	"encoding/json"
	"log"
	"time"

	"github.com/gorilla/websocket"
)

const (
	writeWait         = 10 * time.Second
	pongWait          = 60 * time.Second
	pingPeriod        = (pongWait * 9) / 10
	maxMessageSize    = 4096
	sendBufSize       = 256
	maxMessagesPerSec = 50
	maxNameLen        = 16
)

// Client represents a WebSocket connection
type Client struct {
	hub          *Hub
	conn         *websocket.Conn
	send         chan []byte
	playerID     string
	sessionID    string
	remoteAddr   string
	isController bool
	msgCount     int
	msgResetAt   time.Time
	// Auth state
	authPlayerID int64  // 0 = unauthenticated/guest
	authUsername  string // "" = unauthenticated
}

// NewClient creates a new Client
func NewClient(hub *Hub, conn *websocket.Conn, remoteAddr string) *Client {
	return &Client{
		hub:        hub,
		conn:       conn,
		send:       make(chan []byte, sendBufSize),
		remoteAddr: remoteAddr,
	}
}

// ReadPump reads messages from the WebSocket connection
func (c *Client) ReadPump() {
	defer func() {
		c.hub.TrackDisconnect(c.remoteAddr)
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		msgType, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				log.Printf("ws error: %v", err)
			}
			break
		}

		// Rate limiting
		now := time.Now()
		if now.After(c.msgResetAt) {
			c.msgCount = 0
			c.msgResetAt = now.Add(time.Second)
		}
		c.msgCount++
		if c.msgCount > maxMessagesPerSec {
			log.Printf("rate limit exceeded for %s, disconnecting", c.remoteAddr)
			break
		}

		// Binary input messages: 8 bytes [0x01, mx_hi, mx_lo, my_hi, my_lo, flags, thresh_hi, thresh_lo]
		if msgType == websocket.BinaryMessage && len(message) == 8 && message[0] == 0x01 {
			c.handleBinaryInput(message)
		} else {
			c.handleMessage(message)
		}
	}
}

// WritePump writes messages to the WebSocket connection
func (c *Client) WritePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			// Check for binary marker (0xFF prefix from SendBinary)
			var err error
			if len(message) > 0 && message[0] == 0xFF {
				err = c.conn.WriteMessage(websocket.BinaryMessage, message[1:])
			} else {
				err = c.conn.WriteMessage(websocket.TextMessage, message)
			}
			if err != nil {
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// SendJSON sends a JSON message to the client
func (c *Client) SendJSON(msg interface{}) {
	data, err := json.Marshal(msg)
	if err != nil {
		log.Printf("marshal error: %v", err)
		return
	}
	c.SendRaw(data)
}

// SendRaw sends pre-marshaled bytes as a text message to the client
func (c *Client) SendRaw(data []byte) {
	defer func() { recover() }()
	select {
	case c.send <- data:
	default:
		// Client too slow, drop message
	}
}

// SendBinary sends pre-marshaled bytes as a binary WebSocket message
// Prefixes with 0xFF marker byte so WritePump can distinguish from text
func (c *Client) SendBinary(data []byte) {
	defer func() { recover() }()
	msg := make([]byte, len(data)+1)
	msg[0] = 0xFF // binary marker
	copy(msg[1:], data)
	select {
	case c.send <- msg:
	default:
	}
}

// handleMessage routes incoming messages (single-pass decode via InEnvelope)
func (c *Client) handleMessage(raw []byte) {
	var env InEnvelope
	if err := json.Unmarshal(raw, &env); err != nil {
		log.Printf("unmarshal error: %v", err)
		return
	}

	switch env.T {
	case MsgList:
		c.handleList()
	case MsgCreate:
		c.handleCreate(env.D)
	case MsgJoin:
		c.handleJoin(env.D)
	case MsgInput:
		c.handleInput(env.D)
	case MsgLeave:
		c.handleLeave()
	case MsgCheck:
		c.handleCheck(env.D)
	case MsgControl:
		c.handleControl(env.D)
	case MsgReady:
		c.handleReady()
	case MsgTeamPick:
		c.handleTeamPick(env.D)
	case MsgRematch:
		c.handleRematch()
	case MsgRegister:
		c.handleRegister(env.D)
	case MsgLogin:
		c.handleLogin(env.D)
	case MsgAuth:
		c.handleAuth(env.D)
	case MsgProfile:
		c.handleProfile()
	}
}

func (c *Client) handleList() {
	sessions := c.hub.sessions.ListSessions()
	c.SendJSON(Envelope{T: MsgSessions, Data: sessions})
}

func (c *Client) handleCreate(data json.RawMessage) {
	var msg CreateMsg
	if err := json.Unmarshal(data, &msg); err != nil {
		return
	}
	name := msg.Name
	if name == "" {
		name = "Pilot"
	}
	if len(name) > maxNameLen {
		name = name[:maxNameLen]
	}
	sname := msg.SessionName
	if sname == "" {
		sname = "Battle Arena"
	}
	if len(sname) > 30 {
		sname = sname[:30]
	}

	mode := GameMode(msg.Mode)
	if mode < ModeFFA || mode > ModeWaveSurvival {
		mode = ModeFFA
	}
	sess := c.hub.sessions.CreateSession(sname, mode, c.hub.db)
	if sess == nil {
		c.SendJSON(Envelope{T: MsgError, Data: ErrorMsg{Msg: "too many active sessions"}})
		return
	}

	c.hub.sessions.MarkActive(sess.ID)
	c.SendJSON(Envelope{T: MsgCreated, Data: map[string]string{"sid": sess.ID}})
}

func (c *Client) handleJoin(data json.RawMessage) {
	var msg JoinMsg
	if err := json.Unmarshal(data, &msg); err != nil {
		return
	}
	name := msg.Name
	if name == "" {
		name = "Pilot"
	}
	if len(name) > maxNameLen {
		name = name[:maxNameLen]
	}

	sess := c.hub.sessions.GetSession(msg.SessionID)
	if sess == nil {
		c.SendJSON(Envelope{T: MsgError, Data: ErrorMsg{Msg: "session not found"}})
		return
	}

	player := sess.Game.AddPlayer(name)
	if player == nil {
		c.SendJSON(Envelope{T: MsgError, Data: ErrorMsg{Msg: "session full"}})
		return
	}
	c.hub.sessions.MarkActive(sess.ID)
	c.playerID = player.ID
	c.sessionID = sess.ID

	// Link auth to in-game player
	player.AuthPlayerID = c.authPlayerID

	sess.Game.SetClient(player.ID, c)

	c.SendJSON(Envelope{T: MsgJoined, Data: map[string]string{"sid": sess.ID}})
	c.SendJSON(Envelope{T: MsgWelcome, Data: WelcomeMsg{ID: player.ID, Ship: player.ShipType}})
}

// handleBinaryInput decodes a compact 8-byte binary input message
func (c *Client) handleBinaryInput(msg []byte) {
	if c.sessionID == "" || c.playerID == "" {
		return
	}
	// Decode: [0x01, mx_hi, mx_lo, my_hi, my_lo, flags, thresh_hi, thresh_lo]
	mx := float64(int16(uint16(msg[1])<<8 | uint16(msg[2])))
	my := float64(int16(uint16(msg[3])<<8 | uint16(msg[4])))
	flags := msg[5]
	thresh := float64(uint16(msg[6])<<8 | uint16(msg[7]))

	input := ClientInput{
		MX:      mx,
		MY:      my,
		Fire:    flags&0x01 != 0,
		Boost:   flags&0x02 != 0,
		Ability: flags&0x04 != 0,
		Thresh:  thresh,
	}
	sess := c.hub.sessions.GetSession(c.sessionID)
	if sess == nil {
		return
	}
	sess.Game.HandleInput(c.playerID, input)
}

func (c *Client) handleInput(data json.RawMessage) {
	if c.sessionID == "" || c.playerID == "" {
		return
	}
	var input ClientInput
	if err := json.Unmarshal(data, &input); err != nil {
		return
	}
	sess := c.hub.sessions.GetSession(c.sessionID)
	if sess == nil {
		return
	}
	sess.Game.HandleInput(c.playerID, input)
}

func (c *Client) handleCheck(data json.RawMessage) {
	var msg CheckMsg
	if err := json.Unmarshal(data, &msg); err != nil {
		return
	}
	sess := c.hub.sessions.GetSession(msg.SID)
	if sess == nil {
		c.SendJSON(Envelope{T: MsgChecked, Data: CheckedMsg{SID: msg.SID, Exists: false}})
		return
	}
	c.SendJSON(Envelope{T: MsgChecked, Data: CheckedMsg{
		SID:     msg.SID,
		Exists:  true,
		Name:    sess.Name,
		Players: sess.Game.PlayerCount(),
	}})
}

func (c *Client) handleLeave() {
	if c.sessionID != "" {
		if c.isController {
			sess := c.hub.sessions.GetSession(c.sessionID)
			if sess != nil {
				sess.Game.RemoveController(c.playerID)
			}
		} else {
			c.hub.sessions.RemovePlayer(c.sessionID, c.playerID)
		}
		c.sessionID = ""
		c.playerID = ""
		c.isController = false
	}
}

func (c *Client) handleControl(data json.RawMessage) {
	var msg ControlMsg
	if err := json.Unmarshal(data, &msg); err != nil {
		return
	}
	sess := c.hub.sessions.GetSession(msg.SID)
	if sess == nil {
		c.SendJSON(Envelope{T: MsgError, Data: ErrorMsg{Msg: "session not found"}})
		return
	}
	if !sess.Game.HasPlayer(msg.PlayerID) {
		c.SendJSON(Envelope{T: MsgError, Data: ErrorMsg{Msg: "player not found"}})
		return
	}

	c.sessionID = msg.SID
	c.playerID = msg.PlayerID
	c.isController = true

	sess.Game.SetController(msg.PlayerID, c)
	c.SendJSON(Envelope{T: MsgControlOK, Data: map[string]string{"pid": msg.PlayerID}})
}

func (c *Client) handleReady() {
	if c.sessionID == "" || c.playerID == "" {
		return
	}
	sess := c.hub.sessions.GetSession(c.sessionID)
	if sess == nil {
		return
	}
	sess.Game.HandleReady(c.playerID)
}

func (c *Client) handleTeamPick(data json.RawMessage) {
	if c.sessionID == "" || c.playerID == "" {
		return
	}
	var msg TeamPickMsg
	if err := json.Unmarshal(data, &msg); err != nil {
		return
	}
	sess := c.hub.sessions.GetSession(c.sessionID)
	if sess == nil {
		return
	}
	sess.Game.HandleTeamPick(c.playerID, msg.Team)
}

func (c *Client) handleRematch() {
	if c.sessionID == "" || c.playerID == "" {
		return
	}
	sess := c.hub.sessions.GetSession(c.sessionID)
	if sess == nil {
		return
	}
	sess.Game.HandleRematch(c.playerID)
}

func (c *Client) handleRegister(data json.RawMessage) {
	if c.hub.auth == nil {
		return
	}
	var msg RegisterMsg
	if err := json.Unmarshal(data, &msg); err != nil {
		return
	}
	id, token, err := c.hub.auth.Register(msg.Username, msg.Password)
	if err != nil {
		c.SendJSON(Envelope{T: MsgError, Data: ErrorMsg{Msg: err.Error()}})
		return
	}
	c.authPlayerID = id
	c.authUsername = msg.Username
	c.SendJSON(Envelope{T: MsgAuthOK, Data: AuthOKMsg{
		Token:    token,
		Username: msg.Username,
		PlayerID: id,
	}})
}

func (c *Client) handleLogin(data json.RawMessage) {
	if c.hub.auth == nil {
		return
	}
	var msg LoginMsg
	if err := json.Unmarshal(data, &msg); err != nil {
		return
	}
	id, token, err := c.hub.auth.Login(msg.Username, msg.Password, c.remoteAddr)
	if err != nil {
		c.SendJSON(Envelope{T: MsgError, Data: ErrorMsg{Msg: err.Error()}})
		return
	}
	c.authPlayerID = id
	c.authUsername = msg.Username
	c.SendJSON(Envelope{T: MsgAuthOK, Data: AuthOKMsg{
		Token:    token,
		Username: msg.Username,
		PlayerID: id,
	}})
}

func (c *Client) handleAuth(data json.RawMessage) {
	if c.hub.auth == nil {
		return
	}
	var msg AuthMsg
	if err := json.Unmarshal(data, &msg); err != nil {
		return
	}
	id, username, err := c.hub.auth.ValidateToken(msg.Token)
	if err != nil {
		c.SendJSON(Envelope{T: MsgError, Data: ErrorMsg{Msg: "invalid token"}})
		return
	}
	c.authPlayerID = id
	c.authUsername = username
	c.SendJSON(Envelope{T: MsgAuthOK, Data: AuthOKMsg{
		Token:    msg.Token,
		Username: username,
		PlayerID: id,
	}})
}

func (c *Client) handleProfile() {
	if c.hub.db == nil || c.authPlayerID == 0 {
		c.SendJSON(Envelope{T: MsgError, Data: ErrorMsg{Msg: "not authenticated"}})
		return
	}
	stats, err := c.hub.db.GetStats(c.authPlayerID)
	if err != nil || stats == nil {
		c.SendJSON(Envelope{T: MsgError, Data: ErrorMsg{Msg: "profile not found"}})
		return
	}
	c.SendJSON(Envelope{T: MsgProfileData, Data: ProfileDataMsg{
		Username: c.authUsername,
		Level:    stats.Level,
		XP:       stats.XP,
		Kills:    stats.Kills,
		Deaths:   stats.Deaths,
		Wins:     stats.Wins,
		Losses:   stats.Losses,
		Playtime: stats.Playtime,
	}})
}
