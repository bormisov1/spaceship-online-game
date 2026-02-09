package main

import "testing"

func TestApplyDamage(t *testing.T) {
	p := &Player{
		ID:    "test",
		Alive: true,
		HP:    100,
		MaxHP: 100,
	}

	died := ApplyDamage(p, 50)
	if died {
		t.Error("should not die from 50 damage")
	}
	if p.HP != 50 {
		t.Errorf("expected HP 50, got %d", p.HP)
	}

	died = ApplyDamage(p, 60)
	if !died {
		t.Error("should die from 60 more damage")
	}
}

func TestApplyDamageToDeadPlayer(t *testing.T) {
	p := &Player{
		ID:    "test",
		Alive: false,
		HP:    0,
		MaxHP: 100,
	}
	died := ApplyDamage(p, 50)
	if died {
		t.Error("dead player should not die again")
	}
}

func TestRespawnPlayer(t *testing.T) {
	p := &Player{
		ID:       "test",
		Alive:    false,
		HP:       0,
		MaxHP:    PlayerMaxHP,
		VX:       100,
		VY:       100,
		RespawnT: 3.0,
	}
	RespawnPlayer(p)
	if !p.Alive {
		t.Error("should be alive after respawn")
	}
	if p.HP != PlayerMaxHP {
		t.Errorf("expected full HP, got %d", p.HP)
	}
	if p.VX != 0 || p.VY != 0 {
		t.Error("velocity should be zero after respawn")
	}
}
