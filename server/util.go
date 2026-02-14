package main

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"math"
)

// GenerateID returns a random hex string of the given byte length
func GenerateID(byteLen int) string {
	b := make([]byte, byteLen)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// GenerateUUID returns a random UUID v4 string
func GenerateUUID() string {
	b := make([]byte, 16)
	rand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40 // version 4
	b[8] = (b[8] & 0x3f) | 0x80 // variant 10
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
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

// round1 rounds a float64 to 1 decimal place to reduce JSON payload size
func round1(x float64) float64 {
	return math.Round(x*10) / 10
}

// LerpAngle interpolates between two angles taking the short path
func LerpAngle(from, to, t float64) float64 {
	diff := NormalizeAngle(to - from)
	return from + diff*t
}
