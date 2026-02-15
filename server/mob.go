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

	// Smart AI constants
	MobOptimalRange   = 450.0 // preferred combat distance
	MobOptimalRangeSq = MobOptimalRange * MobOptimalRange
	MobDodgeRange     = 300.0 // range to detect incoming projectiles
	MobDodgeRangeSq   = MobDodgeRange * MobDodgeRange
	MobDodgeImpulse   = 120.0 // lateral velocity impulse when dodging
	MobDodgeCooldown  = 0.3   // seconds between dodge reactions
	MobStrafeFlipMin  = 1.5   // min seconds before strafe direction flip
	MobStrafeFlipMax  = 3.5   // max seconds before strafe direction flip
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

	// Smart AI state
	StrafeDir   float64 // +1 or -1 for circle strafe direction
	StrafeTimer float64 // timer until strafe direction flip
	DodgeCD     float64 // cooldown for dodge reactions

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

	// Random strafe direction
	if rand.Float64() < 0.5 {
		m.StrafeDir = 1
	} else {
		m.StrafeDir = -1
	}
	m.StrafeTimer = MobStrafeFlipMin + rand.Float64()*(MobStrafeFlipMax-MobStrafeFlipMin)
	return m
}

// Update moves the mob and steers toward nearest player or center.
// Returns true if the mob wants to fire this tick.
func (m *Mob) Update(dt float64, players map[string]*Player, projectiles map[string]*Projectile) bool {
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
	if m.DodgeCD > 0 {
		m.DodgeCD -= dt
	}

	// Find nearest alive player within detect range (also capture velocity for lead targeting)
	var targetX, targetY, targetVX, targetVY float64
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
			targetVX = p.VX
			targetVY = p.VY
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

		// --- LEAD TARGETING: aim at predicted position ---
		dist := math.Sqrt(bestDist)
		timeToHit := dist / ProjectileSpeed
		leadX := targetX + targetVX*timeToHit
		leadY := targetY + targetVY*timeToHit

		// Rotate toward lead position (for aiming/shooting)
		desiredR := math.Atan2(leadY-m.Y, leadX-m.X)
		diff := NormalizeAngle(desiredR - m.Rotation)
		maxTurn := MobTurnSpeed * dt
		if diff > maxTurn {
			diff = maxTurn
		} else if diff < -maxTurn {
			diff = -maxTurn
		}
		m.Rotation += diff

		// --- OPTIMAL DISTANCE + CIRCLE STRAFE: compute movement direction ---
		angleToTarget := math.Atan2(targetY-m.Y, targetX-m.X)
		// radial: +1 = approach, -1 = retreat
		radial := Clamp((dist-MobOptimalRange)/(MobOptimalRange*0.5), -1, 1)
		// tangential: strafe more when near optimal range
		tangential := m.StrafeDir * (1.0 - math.Abs(radial)*0.7)
		moveX := math.Cos(angleToTarget)*radial + math.Cos(angleToTarget+math.Pi/2)*tangential
		moveY := math.Sin(angleToTarget)*radial + math.Sin(angleToTarget+math.Pi/2)*tangential
		moveAngle := math.Atan2(moveY, moveX)

		// Flip strafe direction periodically
		m.StrafeTimer -= dt
		if m.StrafeTimer <= 0 {
			m.StrafeDir = -m.StrafeDir
			m.StrafeTimer = MobStrafeFlipMin + rand.Float64()*(MobStrafeFlipMax-MobStrafeFlipMin)
		}

		// Accelerate in movement direction (decoupled from aim)
		accel := MobAccel * dt
		m.VX += math.Cos(moveAngle) * accel
		m.VY += math.Sin(moveAngle) * accel
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

		// Accelerate in facing direction when wandering
		accel := MobAccel * dt
		m.VX += math.Cos(m.Rotation) * accel
		m.VY += math.Sin(m.Rotation) * accel
	}

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

	// --- DODGE INCOMING PROJECTILES ---
	if m.DodgeCD <= 0 {
		for _, proj := range projectiles {
			if !proj.Alive || proj.OwnerID == m.ID {
				continue
			}
			dx := m.X - proj.X
			dy := m.Y - proj.Y
			d2 := dx*dx + dy*dy
			if d2 > MobDodgeRangeSq {
				continue
			}
			// Is the projectile heading toward us?
			dot := dx*proj.VX + dy*proj.VY
			if dot <= 0 {
				continue
			}
			// Will it pass close enough to hit?
			projSpeed2 := proj.VX*proj.VX + proj.VY*proj.VY
			if projSpeed2 < 1 {
				continue
			}
			t := dot / projSpeed2
			closestX := proj.X + proj.VX*t - m.X
			closestY := proj.Y + proj.VY*t - m.Y
			perpDist2 := closestX*closestX + closestY*closestY
			hitZone := MobRadius + ProjectileRadius + 30
			if perpDist2 < hitZone*hitZone {
				// Dodge perpendicular to projectile direction
				perpX := -proj.VY
				perpY := proj.VX
				perpLen := math.Sqrt(perpX*perpX + perpY*perpY)
				if perpLen > 0 {
					perpX /= perpLen
					perpY /= perpLen
				}
				// Dodge away from the projectile path
				cross := dx*proj.VY - dy*proj.VX
				if cross < 0 {
					perpX = -perpX
					perpY = -perpY
				}
				m.VX += perpX * MobDodgeImpulse
				m.VY += perpY * MobDodgeImpulse
				m.DodgeCD = MobDodgeCooldown
				break
			}
		}
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
