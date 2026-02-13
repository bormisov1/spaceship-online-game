package main

import (
	"testing"
)

func TestAsteroidStraightLine(t *testing.T) {
	a := NewAsteroid()
	startX, startY := a.X, a.Y
	vx, vy := a.VX, a.VY

	a.Update(1.0)

	expectedX := startX + vx
	expectedY := startY + vy

	if abs(a.X-expectedX) > 0.01 || abs(a.Y-expectedY) > 0.01 {
		t.Errorf("asteroid should move in straight line: expected (%f,%f) got (%f,%f)",
			expectedX, expectedY, a.X, a.Y)
	}
}

func TestAsteroidDespawnsOffMap(t *testing.T) {
	a := NewAsteroid()
	// Place far off the right edge moving further right
	a.X = WorldWidth + AsteroidRadius*3
	a.Y = WorldHeight / 2
	a.VX = 100
	a.VY = 0

	a.Update(1.0)

	if a.Alive {
		t.Error("asteroid should be dead when off-map")
	}
}

func TestAsteroidStaysAliveOnMap(t *testing.T) {
	a := NewAsteroid()
	a.X = WorldWidth / 2
	a.Y = WorldHeight / 2
	a.VX = 50
	a.VY = 0

	a.Update(1.0)

	if !a.Alive {
		t.Error("asteroid should still be alive when on map")
	}
}

func TestAsteroidSpins(t *testing.T) {
	a := NewAsteroid()
	a.Spin = 1.0
	startR := a.Rotation

	a.Update(1.0)

	if a.Rotation == startR {
		t.Error("asteroid rotation should change when spinning")
	}
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
