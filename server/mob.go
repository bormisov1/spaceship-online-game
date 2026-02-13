package main

import "math"

const (
	MobRadius         = 20.0
	MobMaxHP          = 60
	MobSpeed          = 180.0
	MobDetectRange    = 1200.0
	MobShootRange     = 900.0  // start shooting when this close
	MobRepelRadius    = 50.0
	MobRepelForce     = 120.0 // gentle nudge, allows head-on collisions
	MobExplodeRelV    = 250.0
	MobAccel          = 200.0
	MobFriction       = 0.96
	MobTurnSpeed      = 4.0
	MobShipType       = 3
	MobKillScore      = 5
	MobCollisionDmg   = 30
	MobBurstSize      = 5
	MobBurstFireRate  = 0.15  // seconds between shots in a burst
	MobBurstCooldown  = 5.0   // seconds between bursts
)

// Mob is an AI-controlled enemy ship
type Mob struct {
	ID        string
	X, Y      float64
	VX, VY    float64
	Rotation  float64
	HP        int
	MaxHP     int
	Alive     bool
	BurstLeft int     // shots remaining in current burst
	FireCD    float64 // cooldown between individual shots
	BurstCD   float64 // cooldown between bursts
}

// NewMob spawns a mob at a random map edge
func NewMob() *Mob {
	id := GenerateID(4)
	m := &Mob{
		ID:    id,
		HP:    MobMaxHP,
		MaxHP: MobMaxHP,
		Alive: true,
	}

	// Pick a random edge: 0=left, 1=right, 2=top, 3=bottom
	edge := int(randFloat() * 4)
	switch edge {
	case 0: // left
		m.X = 0
		m.Y = randFloat() * WorldHeight
	case 1: // right
		m.X = WorldWidth
		m.Y = randFloat() * WorldHeight
	case 2: // top
		m.X = randFloat() * WorldWidth
		m.Y = 0
	default: // bottom
		m.X = randFloat() * WorldWidth
		m.Y = WorldHeight
	}

	// Face toward center
	m.Rotation = math.Atan2(WorldHeight/2-m.Y, WorldWidth/2-m.X)
	return m
}

// Update moves the mob and steers toward nearest player or center.
// Returns true if the mob wants to fire this tick.
func (m *Mob) Update(dt float64, players map[string]*Player) bool {
	if !m.Alive {
		return false
	}

	// Tick cooldowns
	if m.FireCD > 0 {
		m.FireCD -= dt
	}
	if m.BurstCD > 0 {
		m.BurstCD -= dt
	}

	// Find nearest alive player within detect range
	var targetX, targetY float64
	bestDist := math.MaxFloat64
	found := false

	for _, p := range players {
		if !p.Alive {
			continue
		}
		d := Distance(m.X, m.Y, p.X, p.Y)
		if d < MobDetectRange && d < bestDist {
			bestDist = d
			targetX = p.X
			targetY = p.Y
			found = true
		}
	}

	if !found {
		// Steer toward center
		targetX = WorldWidth / 2
		targetY = WorldHeight / 2
	}

	// Rotate toward target
	desiredR := math.Atan2(targetY-m.Y, targetX-m.X)
	diff := NormalizeAngle(desiredR - m.Rotation)
	maxTurn := MobTurnSpeed * dt
	if diff > maxTurn {
		diff = maxTurn
	} else if diff < -maxTurn {
		diff = -maxTurn
	}
	m.Rotation += diff

	// Accelerate in facing direction
	accel := MobAccel * dt
	m.VX += math.Cos(m.Rotation) * accel
	m.VY += math.Sin(m.Rotation) * accel

	// Friction
	m.VX *= MobFriction
	m.VY *= MobFriction

	// Clamp speed
	speed := math.Sqrt(m.VX*m.VX + m.VY*m.VY)
	if speed > MobSpeed {
		scale := MobSpeed / speed
		m.VX *= scale
		m.VY *= scale
	}

	// Move
	m.X += m.VX * dt
	m.Y += m.VY * dt

	// Wrap around world edges
	if m.X < 0 {
		m.X += WorldWidth
	} else if m.X > WorldWidth {
		m.X -= WorldWidth
	}
	if m.Y < 0 {
		m.Y += WorldHeight
	} else if m.Y > WorldHeight {
		m.Y -= WorldHeight
	}

	// Burst fire logic
	wantFire := false
	if found && bestDist < MobShootRange {
		if m.BurstLeft > 0 && m.FireCD <= 0 {
			// Continue burst
			wantFire = true
			m.BurstLeft--
			m.FireCD = MobBurstFireRate
			if m.BurstLeft == 0 {
				m.BurstCD = MobBurstCooldown
			}
		} else if m.BurstLeft == 0 && m.BurstCD <= 0 {
			// Start new burst
			m.BurstLeft = MobBurstSize
			wantFire = true
			m.BurstLeft--
			m.FireCD = MobBurstFireRate
			if m.BurstLeft == 0 {
				m.BurstCD = MobBurstCooldown
			}
		}
	}

	return wantFire
}

// TakeDamage reduces HP and returns true if mob died
func (m *Mob) TakeDamage(dmg int) bool {
	if !m.Alive {
		return false
	}
	m.HP -= dmg
	if m.HP <= 0 {
		m.HP = 0
		m.Alive = false
		return true
	}
	return false
}

// ToState converts to protocol state
func (m *Mob) ToState() MobState {
	return MobState{
		ID:    m.ID,
		X:     m.X,
		Y:     m.Y,
		R:     m.Rotation,
		VX:    m.VX,
		VY:    m.VY,
		HP:    m.HP,
		MaxHP: m.MaxHP,
		Alive: m.Alive,
	}
}
