package main

import "sync"

const (
	maxConnsPerIP = 5
	maxTotalConns = 1000
)

// Hub manages all connected clients and routes them to sessions
type Hub struct {
	mu         sync.RWMutex
	clients    map[*Client]bool
	register   chan *Client
	unregister chan *Client
	sessions   *SessionManager
	// Connection limiting (mutex-protected, accessed from HTTP handlers)
	connMu     sync.Mutex
	ipConns    map[string]int
	totalConns int
}

// NewHub creates a new Hub
func NewHub() *Hub {
	h := &Hub{
		clients:    make(map[*Client]bool),
		register:   make(chan *Client, 64),
		unregister: make(chan *Client, 64),
		sessions:   NewSessionManager(),
		ipConns:    make(map[string]int),
	}
	return h
}

func (h *Hub) CanAccept(ip string) bool {
	h.connMu.Lock()
	defer h.connMu.Unlock()
	if h.totalConns >= maxTotalConns {
		return false
	}
	if h.ipConns[ip] >= maxConnsPerIP {
		return false
	}
	return true
}

func (h *Hub) TrackConnect(ip string) {
	h.connMu.Lock()
	defer h.connMu.Unlock()
	h.ipConns[ip]++
	h.totalConns++
}

func (h *Hub) TrackDisconnect(ip string) {
	h.connMu.Lock()
	defer h.connMu.Unlock()
	h.ipConns[ip]--
	if h.ipConns[ip] <= 0 {
		delete(h.ipConns, ip)
	}
	h.totalConns--
}

// Run processes register/unregister events
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
			}
			h.mu.Unlock()
			// Remove from session if in one
			if client.sessionID != "" {
				h.sessions.RemovePlayer(client.sessionID, client.playerID)
			}
		}
	}
}

// ClientCount returns the number of connected clients
func (h *Hub) ClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}
