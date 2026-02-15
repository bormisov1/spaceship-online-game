package main

import (
	"math"
	"math/rand"
)

const (
	MobRadius         = 20.0
	MobMaxHP          = 60
	MobSpeed          = 180.0
	MobDetectRange    = 655.0
	MobShootRange     = 900.0  // start shooting when this close
	MobDetectRangeSq  = MobDetectRange * MobDetectRange
	MobShootRangeSq   = MobShootRange * MobShootRange
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
	MobWanderDrift    = 1.0   // max radians/s the wander angle changes
	MobWanderTurn     = 1.5   // how fast mob turns toward wander heading (rad/s)
	MobPhraseChance   = 0.15  // 15% chance of saying a phrase on state change
	MobLowHPThreshold = 0.25  // below 25% HP triggers "almost dying" phrase
)

// Mob phrase pools keyed by situation
var mobPhrases = map[string][]string{
	"notice": {
		"🎯 Target acquired!",
		"👀 I see you!",
		"💀 You're mine!",
		"🔥 Time to fight!",
		"⚡ Engaging!",
		"😈 Found one!",
	},
	"low_hp": {
		"😰 I'm hit bad...",
		"💔 Systems failing!",
		"🆘 Mayday mayday!",
		"😱 Not like this...",
		"🔧 Need repairs!",
		"💀 Tell my family...",
	},
	"lost": {
		"🤔 Where'd they go?",
		"👻 Lost visual...",
		"❓ Come back here!",
		"🔍 Scanning...",
		"😤 Coward!",
	},
	"fire": {
		"💥 Eat this!",
		"🔫 Pew pew pew!",
		"🎆 FIRE!",
		"☄️ Take that!",
		"😤 Die already!",
	},
	"asteroid_death": {
		"🪨 Oh no—",
		"💫 Didn't see that!",
		"😵 ROCK!",
		"🪨 Not a rock...",
	},
	"mob_crash": {
		"🤦 Watch where you're going!",
		"💥 Oops...",
		"😵 My bad!",
		"🫠 Friendly fire!",
	},
	"kill_player": {
		"😎 Got 'em!",
		"🏆 Too easy!",
		"✨ Another one down!",
		"💪 Who's next?",
	},
}

// Mob is an AI-controlled enemy ship
type Mob struct {
	ID        string
	X, Y      float64
	VX, VY    float64
	Rotation  float64
	HP        int
	MaxHP     int
	Alive       bool
	BurstLeft   int     // shots remaining in current burst
	FireCD      float64 // cooldown between individual shots
	BurstCD     float64 // cooldown between bursts
	WanderAngle float64 // desired heading when idle

	// State tracking for phrases
	WasTracking  bool   // was tracking a player last tick
	SaidLowHP    bool   // already said low-HP phrase
	PendingPhrase string // phrase to broadcast this tick
}

// pickPhrase randomly selects a phrase from a pool (with chance gate)
func pickPhrase(pool string, chance float64) string {
	if rand.Float64() > chance {
		return ""
	}
	phrases := mobPhrases[pool]
	if len(phrases) == 0 {
		return ""
	}
	return phrases[rand.Intn(len(phrases))]
}

// pickPhraseAlways selects a phrase without chance gate
func pickPhraseAlways(pool string) string {
	phrases := mobPhrases[pool]
	if len(phrases) == 0 {
		return ""
	}
	return phrases[rand.Intn(len(phrases))]
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
	m.WanderAngle = m.Rotation
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
		d2 := DistanceSq(m.X, m.Y, p.X, p.Y)
		if d2 < MobDetectRangeSq && d2 < bestDist {
			bestDist = d2
			targetX = p.X
			targetY = p.Y
			found = true
		}
	}

	// Clear pending phrase each tick
	m.PendingPhrase = ""

	if found {
		// State transition: started tracking
		if !m.WasTracking {
			m.PendingPhrase = pickPhrase("notice", MobPhraseChance)
		}
		m.WasTracking = true

		// Rotate toward target player
		desiredR := math.Atan2(targetY-m.Y, targetX-m.X)
		diff := NormalizeAngle(desiredR - m.Rotation)
		maxTurn := MobTurnSpeed * dt
		if diff > maxTurn {
			diff = maxTurn
		} else if diff < -maxTurn {
			diff = -maxTurn
		}
		m.Rotation += diff
	} else {
		// State transition: lost player
		if m.WasTracking {
			m.PendingPhrase = pickPhrase("lost", MobPhraseChance)
		}
		m.WasTracking = false

		// Wander: drift the wander angle gently, then turn toward it
		m.WanderAngle += (randFloat()*2 - 1) * MobWanderDrift * dt
		diff := NormalizeAngle(m.WanderAngle - m.Rotation)
		maxTurn := MobWanderTurn * dt
		if diff > maxTurn {
			diff = maxTurn
		} else if diff < -maxTurn {
			diff = -maxTurn
		}
		m.Rotation += diff
	}

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
	if found && bestDist < MobShootRangeSq {
		if m.BurstLeft > 0 && m.FireCD <= 0 {
			// Continue burst
			wantFire = true
			m.BurstLeft--
			m.FireCD = MobBurstFireRate
			if m.BurstLeft == 0 {
				m.BurstCD = MobBurstCooldown
			}
		} else if m.BurstLeft == 0 && m.BurstCD <= 0 {
			// Start new burst — say fire phrase
			if m.PendingPhrase == "" {
				m.PendingPhrase = pickPhrase("fire", MobPhraseChance)
			}
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
	// Low HP phrase (once)
	if !m.SaidLowHP && float64(m.HP)/float64(m.MaxHP) < MobLowHPThreshold {
		m.SaidLowHP = true
		m.PendingPhrase = pickPhraseAlways("low_hp")
	}
	return false
}

// ToState converts to protocol state
func (m *Mob) ToState() MobState {
	vx := round1(m.VX)
	vy := round1(m.VY)
	return MobState{
		ID:    m.ID,
		X:     round1(m.X),
		Y:     round1(m.Y),
		R:     round1(m.Rotation),
		VX:    &vx,
		VY:    &vy,
		HP:    m.HP,
		MaxHP: m.MaxHP,
		Alive: m.Alive,
	}
}
