package main

import (
	"sync"
)

// Hub manages all connected clients and routes them to sessions
type Hub struct {
	mu         sync.RWMutex
	clients    map[*Client]bool
	register   chan *Client
	unregister chan *Client
	sessions   *SessionManager
}

// NewHub creates a new Hub
func NewHub() *Hub {
	h := &Hub{
		clients:    make(map[*Client]bool),
		register:   make(chan *Client, 64),
		unregister: make(chan *Client, 64),
		sessions:   NewSessionManager(),
	}
	return h
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
