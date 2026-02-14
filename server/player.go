package main

import (
	"crypto/rand"
	"math"
)

const (
	PlayerRadius     = 20.0
	PlayerMaxHP      = 100
	PlayerAccel      = 600.0  // pixels/s²
	PlayerMaxSpeed   = 350.0  // pixels/s
	PlayerFriction   = 0.97   // velocity multiplier per tick
	PlayerBoostMul   = 1.6    // boost speed multiplier
	FireCooldown     = 0.15   // seconds between shots
	RespawnTime      = 3.0    // seconds before respawn
	WorldWidth       = 4000.0
	WorldHeight      = 4000.0
	TurnSpeed        = 8.0    // radians/s max turn rate
)

// Player represents a player in the game
type Player struct {
	ID       string
	Name     string
	X, Y     float64
	VX, VY   float64
	Rotation float64
	HP       int
	MaxHP    int
	ShipType int
	Score    int
	Alive    bool
	FireCD   float64 // fire cooldown remaining
	RespawnT float64 // respawn timer remaining
	TargetR  float64 // target rotation (toward mouse)
	Firing   bool
	Boosting bool
	TargetX   float64 // mouse world X (for distance calc)
	TargetY   float64 // mouse world Y (for distance calc)
	SlowThresh float64 // distance threshold for speed modulation
}

// NewPlayer creates a new player at a random position
func NewPlayer(id, name string, shipType int) *Player {
	return &Player{
		ID:       id,
		Name:     name,
		X:        WorldWidth/4 + randFloat()*WorldWidth/2,
		Y:        WorldHeight/4 + randFloat()*WorldHeight/2,
		HP:       PlayerMaxHP,
		MaxHP:    PlayerMaxHP,
		ShipType: shipType,
		Alive:    true,
	}
}

// Update moves the player one tick (dt in seconds)
func (p *Player) Update(dt float64) {
	if !p.Alive {
		p.RespawnT -= dt
		if p.RespawnT <= 0 {
			p.Respawn()
		}
		return
	}

	// Rotate toward target
	diff := NormalizeAngle(p.TargetR - p.Rotation)
	maxTurn := TurnSpeed * dt
	if diff > maxTurn {
		diff = maxTurn
	} else if diff < -maxTurn {
		diff = -maxTurn
	}
	p.Rotation += diff

	// Accelerate in facing direction
	accel := PlayerAccel * dt
	if p.Boosting {
		accel *= PlayerBoostMul
	}

	// Distance-based speed modulation: slow down as pointer approaches ship
	dist := math.Sqrt((p.TargetX-p.X)*(p.TargetX-p.X) + (p.TargetY-p.Y)*(p.TargetY-p.Y))
	thresh := p.SlowThresh
	if thresh < 20 {
		thresh = 20
	}
	const deadZone = 50.0
	var speedFactor float64 = 1.0
	if dist <= deadZone {
		accel = 0
		speedFactor = 0
	} else if dist < thresh {
		speedFactor = (dist - deadZone) / (thresh - deadZone)
		accel *= speedFactor
	}

	p.VX += math.Cos(p.Rotation) * accel
	p.VY += math.Sin(p.Rotation) * accel

	// Apply friction — use heavy braking when pointer is near the ship
	// so the ship actually stops instead of coasting forever
	friction := PlayerFriction
	if speedFactor < 1.0 {
		// Blend between brake (0.95) and normal friction based on speedFactor
		friction = 0.95 + speedFactor*(PlayerFriction-0.95)
	}
	p.VX *= friction
	p.VY *= friction

	// Clamp speed
	maxSpd := PlayerMaxSpeed
	if p.Boosting {
		maxSpd *= PlayerBoostMul
	}
	speed := math.Sqrt(p.VX*p.VX + p.VY*p.VY)
	if speed > maxSpd {
		scale := maxSpd / speed
		p.VX *= scale
		p.VY *= scale
	}

	// Move
	p.X += p.VX * dt
	p.Y += p.VY * dt

	// Wrap around world edges
	if p.X < 0 {
		p.X += WorldWidth
	} else if p.X > WorldWidth {
		p.X -= WorldWidth
	}
	if p.Y < 0 {
		p.Y += WorldHeight
	} else if p.Y > WorldHeight {
		p.Y -= WorldHeight
	}

	// Cooldown
	if p.FireCD > 0 {
		p.FireCD -= dt
	}
}

// Respawn resets the player after death
func (p *Player) Respawn() {
	p.X = WorldWidth/4 + randFloat()*WorldWidth/2
	p.Y = WorldHeight/4 + randFloat()*WorldHeight/2
	p.VX = 0
	p.VY = 0
	p.HP = PlayerMaxHP
	p.Alive = true
	p.FireCD = 0
	p.RespawnT = 0
}

// TakeDamage reduces HP and returns true if player died
func (p *Player) TakeDamage(dmg int) bool {
	if !p.Alive {
		return false
	}
	p.HP -= dmg
	if p.HP <= 0 {
		p.HP = 0
		p.Alive = false
		p.RespawnT = RespawnTime
		return true
	}
	return false
}

// CanFire returns true if the player can fire a projectile
func (p *Player) CanFire() bool {
	return p.Alive && p.Firing && p.FireCD <= 0
}

// ToState converts to protocol state
func (p *Player) ToState() PlayerState {
	return PlayerState{
		ID:    p.ID,
		Name:  p.Name,
		X:     p.X,
		Y:     p.Y,
		R:     p.Rotation,
		VX:    p.VX,
		VY:    p.VY,
		HP:    p.HP,
		MaxHP: p.MaxHP,
		Ship:  p.ShipType,
		Score: p.Score,
		Alive: p.Alive,
		Boost: p.Boosting,
	}
}

// randFloat returns a random float64 in [0, 1) using crypto/rand
// For game use, we use a simple approach
var randSrc uint64

func randFloat() float64 {
	// Simple xorshift for non-crypto random
	randSrc ^= randSrc << 13
	randSrc ^= randSrc >> 7
	randSrc ^= randSrc << 17
	if randSrc == 0 {
		randSrc = 1
	}
	return float64(randSrc%10000) / 10000.0
}

func init() {
	// Seed from crypto/rand
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	for i, v := range b {
		randSrc |= uint64(v) << (uint(i) * 8)
	}
	if randSrc == 0 {
		randSrc = 1
	}
}
