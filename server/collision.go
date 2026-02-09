package main

// CheckCollision checks if two circles overlap
func CheckCollision(x1, y1, r1, x2, y2, r2 float64) bool {
	dx := x2 - x1
	dy := y2 - y1
	dist2 := dx*dx + dy*dy
	radSum := r1 + r2
	return dist2 <= radSum*radSum
}

// CheckCollisionWrap checks collision with world wrapping
func CheckCollisionWrap(x1, y1, r1, x2, y2, r2 float64) bool {
	// Check direct
	if CheckCollision(x1, y1, r1, x2, y2, r2) {
		return true
	}

	// Check wrapped positions
	dx := x2 - x1
	dy := y2 - y1

	if dx > WorldWidth/2 {
		dx -= WorldWidth
	} else if dx < -WorldWidth/2 {
		dx += WorldWidth
	}
	if dy > WorldHeight/2 {
		dy -= WorldHeight
	} else if dy < -WorldHeight/2 {
		dy += WorldHeight
	}

	dist2 := dx*dx + dy*dy
	radSum := r1 + r2
	return dist2 <= radSum*radSum
}
