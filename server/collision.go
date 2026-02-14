package main

// CheckCollision checks if two circles overlap
func CheckCollision(x1, y1, r1, x2, y2, r2 float64) bool {
	dx := x2 - x1
	dy := y2 - y1
	dist2 := dx*dx + dy*dy
	radSum := r1 + r2
	return dist2 <= radSum*radSum
}

