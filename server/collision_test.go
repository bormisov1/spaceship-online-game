package main

import "testing"

func TestCheckCollision(t *testing.T) {
	// Overlapping circles
	if !CheckCollision(0, 0, 10, 15, 0, 10) {
		t.Error("circles should collide (overlapping)")
	}

	// Touching circles
	if !CheckCollision(0, 0, 10, 20, 0, 10) {
		t.Error("circles should collide (touching)")
	}

	// Non-overlapping circles
	if CheckCollision(0, 0, 10, 25, 0, 10) {
		t.Error("circles should not collide")
	}

	// Same position
	if !CheckCollision(5, 5, 1, 5, 5, 1) {
		t.Error("same position should collide")
	}
}

func TestCheckCollisionWrap(t *testing.T) {
	// Circles near world edges that are actually close
	if !CheckCollisionWrap(10, 10, 20, WorldWidth-5, 10, 20) {
		t.Error("should collide across X wrap")
	}

	if !CheckCollisionWrap(10, 10, 20, 10, WorldHeight-5, 20) {
		t.Error("should collide across Y wrap")
	}

	// Far apart circles
	if CheckCollisionWrap(100, 100, 10, WorldWidth/2, WorldHeight/2, 10) {
		t.Error("should not collide when far apart")
	}
}
