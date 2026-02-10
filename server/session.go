package main

import (
	"sync"
)

const maxSessions = 100

// Session represents a game session that players can join
type Session struct {
	ID   string
	Name string
	Game *Game
}

// SessionManager handles creation and lookup of sessions
type SessionManager struct {
	mu       sync.RWMutex
	sessions map[string]*Session
}

// NewSessionManager creates a new SessionManager
func NewSessionManager() *SessionManager {
	return &SessionManager{
		sessions: make(map[string]*Session),
	}
}

// CreateSession creates a new game session. Returns nil if limit reached.
func (sm *SessionManager) CreateSession(name string) *Session {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if len(sm.sessions) >= maxSessions {
		return nil
	}

	id := GenerateUUID()
	game := NewGame()
	sess := &Session{
		ID:   id,
		Name: name,
		Game: game,
	}
	sm.sessions[id] = sess
	go game.Run()
	return sess
}

// GetSession returns a session by ID
func (sm *SessionManager) GetSession(id string) *Session {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.sessions[id]
}

// RemovePlayer removes a player from a session
func (sm *SessionManager) RemovePlayer(sessionID, playerID string) {
	sm.mu.RLock()
	sess, ok := sm.sessions[sessionID]
	sm.mu.RUnlock()
	if !ok {
		return
	}
	sess.Game.RemovePlayer(playerID)

	// Clean up empty sessions
	if sess.Game.PlayerCount() == 0 {
		sess.Game.Stop()
		sm.mu.Lock()
		delete(sm.sessions, sessionID)
		sm.mu.Unlock()
	}
}

// ListSessions returns info about all active sessions
func (sm *SessionManager) ListSessions() []SessionInfo {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	list := make([]SessionInfo, 0, len(sm.sessions))
	for _, sess := range sm.sessions {
		list = append(list, SessionInfo{
			ID:      sess.ID,
			Name:    sess.Name,
			Players: sess.Game.PlayerCount(),
		})
	}
	return list
}
