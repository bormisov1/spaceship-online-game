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
	hub        *Hub
	conn       *websocket.Conn
	send       chan []byte
	playerID   string
	sessionID  string
	remoteAddr string
	msgCount   int
	msgResetAt time.Time
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
		_, message, err := c.conn.ReadMessage()
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

		c.handleMessage(message)
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
			err := c.conn.WriteMessage(websocket.TextMessage, message)
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
	defer func() { recover() }()
	select {
	case c.send <- data:
	default:
		// Client too slow, drop message
	}
}

// handleMessage routes incoming messages
func (c *Client) handleMessage(raw []byte) {
	var env Envelope
	if err := json.Unmarshal(raw, &env); err != nil {
		log.Printf("unmarshal error: %v", err)
		return
	}

	switch env.T {
	case MsgList:
		c.handleList()
	case MsgCreate:
		c.handleCreate(raw)
	case MsgJoin:
		c.handleJoin(raw)
	case MsgInput:
		c.handleInput(raw)
	case MsgLeave:
		c.handleLeave()
	case MsgCheck:
		c.handleCheck(raw)
	}
}

func (c *Client) handleList() {
	sessions := c.hub.sessions.ListSessions()
	c.SendJSON(Envelope{T: MsgSessions, Data: sessions})
}

func (c *Client) handleCreate(raw []byte) {
	var msg struct {
		T string    `json:"t"`
		D CreateMsg `json:"d"`
	}
	if err := json.Unmarshal(raw, &msg); err != nil {
		return
	}
	name := msg.D.Name
	if name == "" {
		name = "Pilot"
	}
	if len(name) > maxNameLen {
		name = name[:maxNameLen]
	}
	sname := msg.D.SessionName
	if sname == "" {
		sname = "Battle Arena"
	}
	if len(sname) > 30 {
		sname = sname[:30]
	}

	sess := c.hub.sessions.CreateSession(sname)
	if sess == nil {
		c.SendJSON(Envelope{T: MsgError, Data: ErrorMsg{Msg: "too many active sessions"}})
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

	sess.Game.SetClient(player.ID, c)

	c.SendJSON(Envelope{T: MsgJoined, Data: map[string]string{"sid": sess.ID}})
	c.SendJSON(Envelope{T: MsgWelcome, Data: WelcomeMsg{ID: player.ID, Ship: player.ShipType}})
}

func (c *Client) handleJoin(raw []byte) {
	var msg struct {
		T string  `json:"t"`
		D JoinMsg `json:"d"`
	}
	if err := json.Unmarshal(raw, &msg); err != nil {
		return
	}
	name := msg.D.Name
	if name == "" {
		name = "Pilot"
	}
	if len(name) > maxNameLen {
		name = name[:maxNameLen]
	}

	sess := c.hub.sessions.GetSession(msg.D.SessionID)
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

	sess.Game.SetClient(player.ID, c)

	c.SendJSON(Envelope{T: MsgJoined, Data: map[string]string{"sid": sess.ID}})
	c.SendJSON(Envelope{T: MsgWelcome, Data: WelcomeMsg{ID: player.ID, Ship: player.ShipType}})
}

func (c *Client) handleInput(raw []byte) {
	if c.sessionID == "" || c.playerID == "" {
		return
	}
	var msg struct {
		T string      `json:"t"`
		D ClientInput `json:"d"`
	}
	if err := json.Unmarshal(raw, &msg); err != nil {
		return
	}
	sess := c.hub.sessions.GetSession(c.sessionID)
	if sess == nil {
		return
	}
	sess.Game.HandleInput(c.playerID, msg.D)
}

func (c *Client) handleCheck(raw []byte) {
	var msg struct {
		T string   `json:"t"`
		D CheckMsg `json:"d"`
	}
	if err := json.Unmarshal(raw, &msg); err != nil {
		return
	}
	sess := c.hub.sessions.GetSession(msg.D.SID)
	if sess == nil {
		c.SendJSON(Envelope{T: MsgChecked, Data: CheckedMsg{SID: msg.D.SID, Exists: false}})
		return
	}
	c.SendJSON(Envelope{T: MsgChecked, Data: CheckedMsg{
		SID:     msg.D.SID,
		Exists:  true,
		Name:    sess.Name,
		Players: sess.Game.PlayerCount(),
	}})
}

func (c *Client) handleLeave() {
	if c.sessionID != "" {
		c.hub.sessions.RemovePlayer(c.sessionID, c.playerID)
		c.sessionID = ""
		c.playerID = ""
	}
}
