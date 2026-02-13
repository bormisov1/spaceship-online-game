package main

import (
	"sync"
	"testing"
)

// mockBroadcaster captures sent messages for testing
type mockBroadcaster struct {
	mu       sync.Mutex
	messages []interface{}
}

func (m *mockBroadcaster) SendJSON(msg interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.messages = append(m.messages, msg)
}

func TestGameAddRemovePlayer(t *testing.T) {
	g := NewGame()
	p := g.AddPlayer("TestPilot")
	if p.Name != "TestPilot" {
		t.Errorf("expected name TestPilot, got %s", p.Name)
	}
	if g.PlayerCount() != 1 {
		t.Errorf("expected 1 player, got %d", g.PlayerCount())
	}

	g.RemovePlayer(p.ID)
	if g.PlayerCount() != 0 {
		t.Errorf("expected 0 players, got %d", g.PlayerCount())
	}
}

func TestGameShipTypeRotation(t *testing.T) {
	g := NewGame()
	p1 := g.AddPlayer("A")
	p2 := g.AddPlayer("B")
	p3 := g.AddPlayer("C")
	p4 := g.AddPlayer("D")

	if p1.ShipType != 0 || p2.ShipType != 1 || p3.ShipType != 2 {
		t.Error("ship types should cycle 0-2")
	}
	if p4.ShipType != 0 {
		t.Error("ship type should wrap back to 0")
	}
}

func TestGameHandleInput(t *testing.T) {
	g := NewGame()
	p := g.AddPlayer("Test")

	input := ClientInput{
		MX:   p.X + 100,
		MY:   p.Y,
		Fire: true,
	}
	g.HandleInput(p.ID, input)

	g.mu.RLock()
	player := g.players[p.ID]
	g.mu.RUnlock()

	if !player.Firing {
		t.Error("player should be firing")
	}
}

func TestGameUpdate(t *testing.T) {
	g := NewGame()
	p1 := g.AddPlayer("Player1")
	p2 := g.AddPlayer("Player2")

	mock1 := &mockBroadcaster{}
	mock2 := &mockBroadcaster{}
	g.SetClient(p1.ID, mock1)
	g.SetClient(p2.ID, mock2)

	// Run a few ticks
	for i := 0; i < 10; i++ {
		g.update()
	}

	if g.tick != 10 {
		t.Errorf("expected tick 10, got %d", g.tick)
	}
}

func TestGameProjectileCreation(t *testing.T) {
	g := NewGame()
	p := g.AddPlayer("Shooter")
	p.Firing = true
	p.FireCD = 0

	g.update()

	g.mu.RLock()
	projCount := len(g.projectiles)
	g.mu.RUnlock()

	if projCount != 1 {
		t.Errorf("expected 1 projectile, got %d", projCount)
	}
}
