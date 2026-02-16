package main

import (
	"math"
	"testing"
)

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

func TestPointInTriangle(t *testing.T) {
	// Simple right triangle: (0,0), (10,0), (0,10)
	if !pointInTriangle(2, 2, 0, 0, 10, 0, 0, 10) {
		t.Error("point inside triangle should be detected")
	}
	if !pointInTriangle(0, 0, 0, 0, 10, 0, 0, 10) {
		t.Error("point on vertex should be inside")
	}
	if pointInTriangle(10, 10, 0, 0, 10, 0, 0, 10) {
		t.Error("point outside triangle should not be detected")
	}
	if pointInTriangle(-1, -1, 0, 0, 10, 0, 0, 10) {
		t.Error("point far outside should not be detected")
	}
}

func TestSegmentCircleIntersect(t *testing.T) {
	// Segment through circle center
	if !segmentCircleIntersect(-10, 0, 10, 0, 0, 0, 5) {
		t.Error("segment through circle should intersect")
	}
	// Segment touching circle edge
	if !segmentCircleIntersect(-10, 5, 10, 5, 0, 0, 5) {
		t.Error("segment tangent to circle should intersect")
	}
	// Segment missing circle
	if segmentCircleIntersect(-10, 10, 10, 10, 0, 0, 5) {
		t.Error("segment above circle should not intersect")
	}
	// Segment entirely inside circle
	if !segmentCircleIntersect(-1, 0, 1, 0, 0, 0, 5) {
		t.Error("segment inside circle should intersect")
	}
}

func TestTriangleCircleCollision(t *testing.T) {
	// Unrotated SD triangle at origin: nose at (140,0), stern at (-130,-130) and (-130,130)
	tri := SDTriangleHitbox

	// Circle at center of triangle — should collide
	if !CheckTriangleCircleCollision(0, 0, 0, tri, 0, 0, 10) {
		t.Error("circle at triangle center should collide")
	}

	// Circle at nose — should collide
	if !CheckTriangleCircleCollision(0, 0, 0, tri, 135, 0, 10) {
		t.Error("circle near nose should collide")
	}

	// Circle far from triangle — should not collide
	if CheckTriangleCircleCollision(0, 0, 0, tri, 300, 0, 10) {
		t.Error("circle far away should not collide")
	}

	// Circle just outside the stern edge — should not collide
	if CheckTriangleCircleCollision(0, 0, 0, tri, -130, 145, 5) {
		t.Error("circle just outside stern should not collide")
	}

	// Circle touching the stern edge — should collide
	if !CheckTriangleCircleCollision(0, 0, 0, tri, -130, 125, 5) {
		t.Error("circle touching stern edge should collide")
	}

	// Rotated 90 degrees (pi/2): nose now at (0,140), stern at (130,-130) and (-130,-130)
	rot := math.Pi / 2
	if !CheckTriangleCircleCollision(0, 0, rot, tri, 0, 135, 10) {
		t.Error("circle near rotated nose should collide")
	}
	if CheckTriangleCircleCollision(0, 0, rot, tri, 0, 160, 5) {
		t.Error("circle beyond rotated nose should not collide")
	}

	// Triangle at non-origin position
	if !CheckTriangleCircleCollision(500, 500, 0, tri, 500, 500, 10) {
		t.Error("circle at translated triangle center should collide")
	}
	if !CheckTriangleCircleCollision(500, 500, 0, tri, 640, 500, 10) {
		t.Error("circle near translated nose should collide")
	}
}

func TestTrianglePointCollision(t *testing.T) {
	tri := SDTriangleHitbox

	// Point at center
	if !CheckTrianglePointCollision(0, 0, 0, tri, 0, 0) {
		t.Error("point at center should be inside")
	}

	// Point at nose
	if !CheckTrianglePointCollision(0, 0, 0, tri, 140, 0) {
		t.Error("point at nose vertex should be inside")
	}

	// Point outside beyond nose
	if CheckTrianglePointCollision(0, 0, 0, tri, 150, 0) {
		t.Error("point beyond nose should be outside")
	}

	// Point inside near stern
	if !CheckTrianglePointCollision(0, 0, 0, tri, -120, 0) {
		t.Error("point near stern center should be inside")
	}

	// Point outside above
	if CheckTrianglePointCollision(0, 0, 0, tri, 0, 100) {
		t.Error("point above triangle should be outside")
	}
}

func TestCheckMobCollision(t *testing.T) {
	// Star Destroyer (ShipType 3) uses triangle hitbox
	sd := &Mob{
		X: 0, Y: 0, Rotation: 0,
		ShipType: 3, Radius: SDRadius,
	}
	// Circle at center — should collide
	if !CheckMobCollision(sd, 0, 0, 10) {
		t.Error("SD: circle at center should collide")
	}
	// Circle far away — should not
	if CheckMobCollision(sd, 300, 0, 10) {
		t.Error("SD: circle far away should not collide")
	}
	// Circle outside the triangle but near stern — at (0, 140) the triangle half-height at x=0 is ~67
	if CheckMobCollision(sd, 0, 140, 5) {
		t.Error("SD: circle outside triangle should NOT collide")
	}

	// TIE fighter (ShipType 4) uses circle hitbox
	tie := &Mob{
		X: 0, Y: 0, Rotation: 0,
		ShipType: 4, Radius: TieRadius,
	}
	if !CheckMobCollision(tie, 20, 0, 10) {
		t.Error("TIE: circle within radius should collide")
	}
	if CheckMobCollision(tie, 50, 0, 10) {
		t.Error("TIE: circle outside radius should not collide")
	}
}
