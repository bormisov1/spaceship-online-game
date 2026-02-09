package main

import (
	"math"
	"testing"
)

func TestNewProjectile(t *testing.T) {
	owner := &Player{
		ID:       "owner1",
		X:        500,
		Y:        500,
		Rotation: 0, // facing right
		VX:       0,
		VY:       0,
	}
	proj := NewProjectile(owner)
	if proj.OwnerID != "owner1" {
		t.Errorf("expected owner owner1, got %s", proj.OwnerID)
	}
	if !proj.Alive {
		t.Error("projectile should be alive")
	}
	if proj.Life != ProjectileLifetime {
		t.Errorf("expected lifetime %f, got %f", ProjectileLifetime, proj.Life)
	}
	// Should be spawned ahead of player
	if proj.X <= owner.X {
		t.Error("projectile should spawn ahead of player")
	}
	// Velocity should be roughly ProjectileSpeed in X direction
	if math.Abs(proj.VX-ProjectileSpeed) > 1 {
		t.Errorf("expected VX ~%f, got %f", ProjectileSpeed, proj.VX)
	}
}

func TestProjectileUpdate(t *testing.T) {
	proj := &Projectile{
		ID:    "proj1",
		X:     100,
		Y:     100,
		VX:    ProjectileSpeed,
		VY:    0,
		Life:  ProjectileLifetime,
		Alive: true,
	}
	dt := 1.0 / 60.0
	proj.Update(dt)
	expectedX := 100 + ProjectileSpeed*dt
	if math.Abs(proj.X-expectedX) > 0.01 {
		t.Errorf("expected X ~%f, got %f", expectedX, proj.X)
	}
	if proj.Life >= ProjectileLifetime {
		t.Error("life should decrease")
	}
}

func TestProjectileExpiry(t *testing.T) {
	proj := &Projectile{
		ID:    "proj1",
		X:     100,
		Y:     100,
		VX:    0,
		VY:    0,
		Life:  0.01,
		Alive: true,
	}
	proj.Update(0.02) // exceed lifetime
	if proj.Alive {
		t.Error("projectile should be dead after lifetime expires")
	}
}

func TestProjectileWorldWrap(t *testing.T) {
	proj := &Projectile{
		ID:    "proj1",
		X:     WorldWidth - 1,
		Y:     WorldHeight - 1,
		VX:    100,
		VY:    100,
		Life:  2.0,
		Alive: true,
	}
	proj.Update(0.5)
	if proj.X >= WorldWidth || proj.X < 0 {
		t.Errorf("X should wrap, got %f", proj.X)
	}
}

func TestProjectileToState(t *testing.T) {
	proj := &Projectile{
		ID:       "proj1",
		OwnerID:  "owner1",
		X:        100,
		Y:        200,
		Rotation: 1.5,
		Alive:    true,
	}
	s := proj.ToState()
	if s.ID != "proj1" || s.Owner != "owner1" || s.X != 100 || s.Y != 200 {
		t.Error("state mismatch")
	}
}
