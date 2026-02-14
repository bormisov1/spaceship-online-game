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

