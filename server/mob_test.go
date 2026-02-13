package main

import (
	"math"
	"testing"
)

func TestMobEdgeSpawn(t *testing.T) {
	for i := 0; i < 20; i++ {
		m := NewMob()
		onEdge := m.X == 0 || m.X == WorldWidth || m.Y == 0 || m.Y == WorldHeight
		if !onEdge {
			t.Errorf("mob should spawn on edge, got (%f, %f)", m.X, m.Y)
		}
		if !m.Alive {
			t.Error("mob should be alive on spawn")
		}
		if m.HP != MobMaxHP {
			t.Errorf("mob HP should be %d, got %d", MobMaxHP, m.HP)
		}
	}
}

func TestMobTakeDamage(t *testing.T) {
	m := NewMob()

	died := m.TakeDamage(20)
	if died {
		t.Error("mob should not die from 20 damage")
	}
	if m.HP != 40 {
		t.Errorf("expected HP 40, got %d", m.HP)
	}

	died = m.TakeDamage(20)
	if died {
		t.Error("mob should not die from 40 total damage")
	}

	died = m.TakeDamage(20)
	if !died {
		t.Error("mob should die from 60 total damage")
	}
	if m.Alive {
		t.Error("mob should not be alive after death")
	}
}

func TestMobTakeDamageWhenDead(t *testing.T) {
	m := NewMob()
	m.Alive = false
	died := m.TakeDamage(100)
	if died {
		t.Error("dead mob should not report dying again")
	}
}

func TestMobAISteersTowardPlayer(t *testing.T) {
	m := NewMob()
	m.X = 2000
	m.Y = 2000
	m.VX = 0
	m.VY = 0
	m.Rotation = 0

	players := map[string]*Player{
		"p1": {
			ID: "p1", X: 2200, Y: 2000, Alive: true,
		},
	}

	// Run a few updates
	for i := 0; i < 60; i++ {
		m.Update(1.0/60.0, players)
	}

	// Mob should have moved toward the player (rightward)
	if m.X <= 2000 {
		t.Errorf("mob should have moved right toward player, X=%f", m.X)
	}
}

func TestMobAISteersTowardCenter(t *testing.T) {
	m := NewMob()
	m.X = 100
	m.Y = 100
	m.VX = 0
	m.VY = 0
	m.Rotation = 0

	players := make(map[string]*Player) // no players

	for i := 0; i < 120; i++ {
		m.Update(1.0/60.0, players)
	}

	// Should move toward center (2000, 2000)
	dist := math.Sqrt((m.X-2000)*(m.X-2000) + (m.Y-2000)*(m.Y-2000))
	initialDist := math.Sqrt((100-2000)*(100-2000) + (100-2000)*(100-2000))
	if dist >= initialDist {
		t.Errorf("mob should have moved closer to center, dist=%f initial=%f", dist, initialDist)
	}
}
