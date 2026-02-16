package main

import "math"

// CheckCollision checks if two circles overlap
func CheckCollision(x1, y1, r1, x2, y2, r2 float64) bool {
	dx := x2 - x1
	dy := y2 - y1
	dist2 := dx*dx + dy*dy
	radSum := r1 + r2
	return dist2 <= radSum*radSum
}

// TriangleHitbox defines 3 vertices relative to center at rotation=0.
type TriangleHitbox struct {
	// Vertices relative to center (unrotated)
	X0, Y0 float64
	X1, Y1 float64
	X2, Y2 float64
}

// Star Destroyer triangle hitbox — rotation=0 faces RIGHT (+X).
// Vertices sized to match the 300-unit rendered sprite.
var SDTriangleHitbox = TriangleHitbox{
	X0: 140, Y0: 0,     // nose (front tip, +X = forward)
	X1: -130, Y1: -130, // stern top-left
	X2: -130, Y2: 130,  // stern bottom-left
}

// transformedTriangle returns the world-space vertices of a triangle hitbox
// given the entity's position and rotation.
func transformedTriangle(cx, cy, rot float64, tri TriangleHitbox) (ax, ay, bx, by, pcx, pcy float64) {
	cosR := math.Cos(rot)
	sinR := math.Sin(rot)
	ax = cx + tri.X0*cosR - tri.Y0*sinR
	ay = cy + tri.X0*sinR + tri.Y0*cosR
	bx = cx + tri.X1*cosR - tri.Y1*sinR
	by = cy + tri.X1*sinR + tri.Y1*cosR
	pcx = cx + tri.X2*cosR - tri.Y2*sinR
	pcy = cy + tri.X2*sinR + tri.Y2*cosR
	return
}

// cross2D returns the 2D cross product of vectors (bx-ax,by-ay) and (cx-ax,cy-ay).
func cross2D(ax, ay, bx, by, cx, cy float64) float64 {
	return (bx-ax)*(cy-ay) - (by-ay)*(cx-ax)
}

// pointInTriangle checks if point (px,py) is inside triangle (ax,ay)-(bx,by)-(cx,cy).
func pointInTriangle(px, py, ax, ay, bx, by, cx, cy float64) bool {
	d1 := cross2D(ax, ay, bx, by, px, py)
	d2 := cross2D(bx, by, cx, cy, px, py)
	d3 := cross2D(cx, cy, ax, ay, px, py)
	hasNeg := (d1 < 0) || (d2 < 0) || (d3 < 0)
	hasPos := (d1 > 0) || (d2 > 0) || (d3 > 0)
	return !(hasNeg && hasPos)
}

// segmentCircleIntersect checks if a line segment (x1,y1)-(x2,y2) intersects a circle at (cx,cy) with radius r.
func segmentCircleIntersect(x1, y1, x2, y2, cx, cy, r float64) bool {
	dx := x2 - x1
	dy := y2 - y1
	fx := x1 - cx
	fy := y1 - cy
	a := dx*dx + dy*dy
	b := 2 * (fx*dx + fy*dy)
	c := fx*fx + fy*fy - r*r
	discriminant := b*b - 4*a*c
	if discriminant < 0 {
		return false
	}
	discriminant = math.Sqrt(discriminant)
	t1 := (-b - discriminant) / (2 * a)
	t2 := (-b + discriminant) / (2 * a)
	return (t1 >= 0 && t1 <= 1) || (t2 >= 0 && t2 <= 1) || (t1 <= 0 && t2 >= 1)
}

// CheckTriangleCircleCollision checks if a rotated triangle hitbox collides with a circle.
// (tx,ty) is the triangle entity center, trot is its rotation.
// (cx,cy,cr) is the circle center and radius.
func CheckTriangleCircleCollision(tx, ty, trot float64, tri TriangleHitbox, cx, cy, cr float64) bool {
	ax, ay, bx, by, pcx, pcy := transformedTriangle(tx, ty, trot, tri)

	// 1. Check if circle center is inside triangle
	if pointInTriangle(cx, cy, ax, ay, bx, by, pcx, pcy) {
		return true
	}

	// 2. Check if circle intersects any triangle edge
	if segmentCircleIntersect(ax, ay, bx, by, cx, cy, cr) {
		return true
	}
	if segmentCircleIntersect(bx, by, pcx, pcy, cx, cy, cr) {
		return true
	}
	if segmentCircleIntersect(pcx, pcy, ax, ay, cx, cy, cr) {
		return true
	}

	return false
}

// CheckTrianglePointCollision checks if a point is inside a rotated triangle hitbox.
func CheckTrianglePointCollision(tx, ty, trot float64, tri TriangleHitbox, px, py float64) bool {
	ax, ay, bx, by, cx, cy := transformedTriangle(tx, ty, trot, tri)
	return pointInTriangle(px, py, ax, ay, bx, by, cx, cy)
}

// CheckMobCollision checks collision between a mob and a circle, using triangle hitbox for Star Destroyers.
func CheckMobCollision(mob *Mob, cx, cy, cr float64) bool {
	if mob.ShipType == 3 {
		return CheckTriangleCircleCollision(mob.X, mob.Y, mob.Rotation, SDTriangleHitbox, cx, cy, cr)
	}
	return CheckCollision(mob.X, mob.Y, mob.Radius, cx, cy, cr)
}
