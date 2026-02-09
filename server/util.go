package main

import (
	"crypto/rand"
	"encoding/hex"
	"math"
)

// GenerateID returns a random hex string of the given byte length
func GenerateID(byteLen int) string {
	b := make([]byte, byteLen)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// Clamp restricts v to [min, max]
func Clamp(v, min, max float64) float64 {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

// Distance returns the distance between two points
func Distance(x1, y1, x2, y2 float64) float64 {
	dx := x2 - x1
	dy := y2 - y1
	return math.Sqrt(dx*dx + dy*dy)
}

// NormalizeAngle wraps angle to [-PI, PI]
func NormalizeAngle(a float64) float64 {
	for a > math.Pi {
		a -= 2 * math.Pi
	}
	for a < -math.Pi {
		a += 2 * math.Pi
	}
	return a
}

// LerpAngle interpolates between two angles taking the short path
func LerpAngle(from, to, t float64) float64 {
	diff := NormalizeAngle(to - from)
	return from + diff*t
}
