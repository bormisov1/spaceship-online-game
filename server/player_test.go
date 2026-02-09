package main

import (
	"math"
	"testing"
)

func TestNewPlayer(t *testing.T) {
	p := NewPlayer("test1", "TestPilot", 2)
	if p.ID != "test1" {
		t.Errorf("expected ID test1, got %s", p.ID)
	}
	if p.Name != "TestPilot" {
		t.Errorf("expected name TestPilot, got %s", p.Name)
	}
	if p.ShipType != 2 {
		t.Errorf("expected ship type 2, got %d", p.ShipType)
	}
	if p.HP != PlayerMaxHP {
		t.Errorf("expected HP %d, got %d", PlayerMaxHP, p.HP)
	}
	if !p.Alive {
		t.Error("expected player to be alive")
	}
}

func TestPlayerUpdate(t *testing.T) {
	p := &Player{
		ID:    "test",
		X:     100,
		Y:     100,
		Alive: true,
		HP:    PlayerMaxHP,
		MaxHP: PlayerMaxHP,
	}
	p.TargetR = 0 // facing right
	p.Update(1.0 / 60.0)

	// Player should have moved slightly
	if p.VX == 0 && p.VY == 0 {
		t.Error("expected velocity change after update")
	}
}

func TestPlayerTakeDamage(t *testing.T) {
	p := &Player{
		ID:    "test",
		Alive: true,
		HP:    100,
		MaxHP: 100,
	}

	died := p.TakeDamage(30)
	if died {
		t.Error("should not have died from 30 damage")
	}
	if p.HP != 70 {
		t.Errorf("expected HP 70, got %d", p.HP)
	}

	died = p.TakeDamage(80)
	if !died {
		t.Error("should have died from 80 more damage")
	}
	if p.Alive {
		t.Error("expected player to be dead")
	}
	if p.HP != 0 {
		t.Errorf("expected HP 0, got %d", p.HP)
	}
}

func TestPlayerRespawn(t *testing.T) {
	p := &Player{
		ID:    "test",
		Alive: false,
		HP:    0,
		MaxHP: PlayerMaxHP,
	}
	p.Respawn()
	if !p.Alive {
		t.Error("expected player to be alive after respawn")
	}
	if p.HP != PlayerMaxHP {
		t.Errorf("expected full HP, got %d", p.HP)
	}
}

func TestPlayerWorldWrap(t *testing.T) {
	p := &Player{
		ID:    "test",
		X:     WorldWidth - 1,
		Y:     WorldHeight - 1,
		VX:    100,
		VY:    100,
		Alive: true,
		HP:    100,
		MaxHP: 100,
	}
	// Move with large dt to go past boundary
	p.Update(0.5)
	if p.X >= WorldWidth || p.X < 0 {
		t.Errorf("X should wrap, got %f", p.X)
	}
	if p.Y >= WorldHeight || p.Y < 0 {
		t.Errorf("Y should wrap, got %f", p.Y)
	}
}

func TestPlayerCanFire(t *testing.T) {
	p := &Player{
		ID:     "test",
		Alive:  true,
		Firing: true,
		FireCD: 0,
		HP:     100,
	}
	if !p.CanFire() {
		t.Error("should be able to fire")
	}

	p.FireCD = 0.1
	if p.CanFire() {
		t.Error("should not fire during cooldown")
	}

	p.FireCD = 0
	p.Alive = false
	if p.CanFire() {
		t.Error("dead player should not fire")
	}
}

func TestPlayerBoostAccel(t *testing.T) {
	// Non-boosting player
	p1 := &Player{
		ID: "test1", X: 100, Y: 100, Alive: true, HP: 100, MaxHP: 100,
		TargetX: 500, TargetY: 100, SlowThresh: 200,
	}
	p1.TargetR = 0
	p1.Update(1.0 / 60.0)
	normalVX := p1.VX

	// Boosting player
	p2 := &Player{
		ID: "test2", X: 100, Y: 100, Alive: true, HP: 100, MaxHP: 100,
		Boosting: true, TargetX: 500, TargetY: 100, SlowThresh: 200,
	}
	p2.TargetR = 0
	p2.Update(1.0 / 60.0)
	boostedVX := p2.VX

	if boostedVX <= normalVX {
		t.Errorf("boosted VX (%f) should be greater than normal VX (%f)", boostedVX, normalVX)
	}
	// Boost should multiply acceleration by PlayerBoostMul
	ratio := boostedVX / normalVX
	if math.Abs(ratio-PlayerBoostMul) > 0.01 {
		t.Errorf("expected boost ratio ~%f, got %f", PlayerBoostMul, ratio)
	}
}

func TestSpeedModulationDeadZone(t *testing.T) {
	// Pointer within dead zone (<=50px) — no acceleration
	p := &Player{
		ID: "test", X: 100, Y: 100, Alive: true, HP: 100, MaxHP: 100,
		TargetX: 130, TargetY: 100, SlowThresh: 200,
	}
	p.TargetR = 0
	p.Update(1.0 / 60.0)
	if p.VX != 0 || p.VY != 0 {
		t.Errorf("expected no velocity in dead zone, got VX=%f VY=%f", p.VX, p.VY)
	}
}

func TestSpeedModulationPartial(t *testing.T) {
	thresh := 200.0

	// Pointer at half threshold distance — partial accel
	halfDist := (thresh + 20) / 2 // midpoint between deadZone and thresh
	pHalf := &Player{
		ID: "half", X: 100, Y: 100, Alive: true, HP: 100, MaxHP: 100,
		TargetX: 100 + halfDist, TargetY: 100, SlowThresh: thresh,
	}
	pHalf.TargetR = 0
	pHalf.Update(1.0 / 60.0)

	// Pointer far away — full accel
	pFull := &Player{
		ID: "full", X: 100, Y: 100, Alive: true, HP: 100, MaxHP: 100,
		TargetX: 100 + thresh + 100, TargetY: 100, SlowThresh: thresh,
	}
	pFull.TargetR = 0
	pFull.Update(1.0 / 60.0)

	if pHalf.VX >= pFull.VX {
		t.Errorf("partial speed VX (%f) should be less than full speed VX (%f)", pHalf.VX, pFull.VX)
	}
	if pHalf.VX <= 0 {
		t.Errorf("partial speed VX should be positive, got %f", pHalf.VX)
	}
}

func TestPlayerToState(t *testing.T) {
	p := &Player{
		ID:       "test",
		Name:     "Pilot",
		X:        100,
		Y:        200,
		Rotation: math.Pi / 4,
		VX:       10,
		VY:       20,
		HP:       80,
		MaxHP:    100,
		ShipType: 1,
		Score:    5,
		Alive:    true,
	}
	s := p.ToState()
	if s.ID != "test" || s.Name != "Pilot" || s.X != 100 || s.Y != 200 {
		t.Error("state mismatch")
	}
	if s.HP != 80 || s.MaxHP != 100 || s.Ship != 1 || s.Score != 5 {
		t.Error("state field mismatch")
	}
}
