package main

import "math"

// AbilityType identifies the ability
type AbilityType int

const (
	AbilityMissileBarrage AbilityType = 0 // Fighter: 5 homing projectiles
	AbilityShield         AbilityType = 1 // Tank: absorb 50 damage for 3s
	AbilityBlink          AbilityType = 2 // Scout: teleport 200px forward
	AbilityHealAura       AbilityType = 3 // Support: heal nearby allies
)

// Ability cooldowns and durations
const (
	MissileBarrageCooldown = 12.0
	MissileBarrageCount    = 5
	MissileBarrageDamage   = 25
	MissileBarrageSpeed    = 500.0
	MissileBarrageLifetime = 3.0
	MissileBarrageTurnRate = 6.0

	ShieldCooldown  = 15.0
	ShieldDuration  = 3.0
	ShieldAbsorb    = 50

	BlinkCooldown = 8.0
	BlinkDistance  = 200.0

	HealAuraCooldown = 18.0
	HealAuraDuration = 5.0
	HealAuraRadius   = 150.0
	HealAuraRate     = 10.0 // HP/s per ally
)

// Ability tracks the state of a player's ability
type Ability struct {
	Type     AbilityType
	Cooldown float64 // remaining cooldown
	Active   bool    // currently active
	Timer    float64 // remaining active duration
	ShieldHP int     // remaining shield HP (Tank)
}

// AbilityForClass returns the default ability for a class
func AbilityForClass(class ShipClass) Ability {
	switch class {
	case ClassTank:
		return Ability{Type: AbilityShield}
	case ClassScout:
		return Ability{Type: AbilityBlink}
	case ClassSupport:
		return Ability{Type: AbilityHealAura}
	default:
		return Ability{Type: AbilityMissileBarrage}
	}
}

// CanActivate returns true if the ability is ready
func (a *Ability) CanActivate() bool {
	return a.Cooldown <= 0 && !a.Active
}

// Activate starts the ability and returns true on success
func (a *Ability) Activate() bool {
	if !a.CanActivate() {
		return false
	}
	switch a.Type {
	case AbilityMissileBarrage:
		a.Cooldown = MissileBarrageCooldown
		// Missiles spawned by game.go
	case AbilityShield:
		a.Active = true
		a.Timer = ShieldDuration
		a.ShieldHP = ShieldAbsorb
		a.Cooldown = ShieldCooldown
	case AbilityBlink:
		a.Cooldown = BlinkCooldown
		// Teleport handled by game.go
	case AbilityHealAura:
		a.Active = true
		a.Timer = HealAuraDuration
		a.Cooldown = HealAuraCooldown
	}
	return true
}

// Update ticks the ability cooldowns and active timers
func (a *Ability) Update(dt float64) {
	if a.Cooldown > 0 {
		a.Cooldown -= dt
		if a.Cooldown < 0 {
			a.Cooldown = 0
		}
	}
	if a.Active {
		a.Timer -= dt
		if a.Timer <= 0 {
			a.Active = false
			a.Timer = 0
			a.ShieldHP = 0
		}
	}
}

// AbsorbDamage applies shield damage absorption, returns remaining damage
func (a *Ability) AbsorbDamage(dmg int) int {
	if !a.Active || a.Type != AbilityShield || a.ShieldHP <= 0 {
		return dmg
	}
	if dmg <= a.ShieldHP {
		a.ShieldHP -= dmg
		return 0
	}
	remaining := dmg - a.ShieldHP
	a.ShieldHP = 0
	a.Active = false
	a.Timer = 0
	return remaining
}

// HomingProjectile is a missile that tracks the nearest enemy
type HomingProjectile struct {
	ID       string
	X, Y     float64
	VX, VY   float64
	Rotation float64
	OwnerID  string
	Alive    bool
	Life     float64
	Damage   int
	worldW   float64
	worldH   float64
}

// NewHomingProjectile creates a homing missile
func NewHomingProjectile(x, y, rotation float64, ownerID string, worldW, worldH float64) *HomingProjectile {
	if worldW == 0 { worldW = WorldWidth }
	if worldH == 0 { worldH = WorldHeight }
	return &HomingProjectile{
		ID:       GenerateID(4),
		X:        x,
		Y:        y,
		VX:       math.Cos(rotation) * MissileBarrageSpeed,
		VY:       math.Sin(rotation) * MissileBarrageSpeed,
		Rotation: rotation,
		OwnerID:  ownerID,
		Alive:    true,
		Life:     MissileBarrageLifetime,
		Damage:   MissileBarrageDamage,
		worldW:   worldW,
		worldH:   worldH,
	}
}

// wrapDelta returns the shortest signed delta considering world wrapping
func wrapDelta(d, size float64) float64 {
	if d > size/2 {
		d -= size
	} else if d < -size/2 {
		d += size
	}
	return d
}

// Update moves the homing projectile toward the nearest enemy
func (h *HomingProjectile) Update(dt float64, players map[string]*Player, mobs map[string]*Mob) {
	if !h.Alive {
		return
	}
	h.Life -= dt
	if h.Life <= 0 {
		h.Alive = false
		return
	}

	ww := h.worldW
	wh := h.worldH

	// Find nearest target (wrap-aware)
	var targetDX, targetDY float64
	bestDist := math.MaxFloat64
	found := false

	for _, p := range players {
		if !p.Alive || p.ID == h.OwnerID {
			continue
		}
		dx := wrapDelta(p.X-h.X, ww)
		dy := wrapDelta(p.Y-h.Y, wh)
		d2 := dx*dx + dy*dy
		if d2 < bestDist {
			bestDist = d2
			targetDX = dx
			targetDY = dy
			found = true
		}
	}
	for _, m := range mobs {
		if !m.Alive {
			continue
		}
		dx := wrapDelta(m.X-h.X, ww)
		dy := wrapDelta(m.Y-h.Y, wh)
		d2 := dx*dx + dy*dy
		if d2 < bestDist {
			bestDist = d2
			targetDX = dx
			targetDY = dy
			found = true
		}
	}

	if found {
		desired := math.Atan2(targetDY, targetDX)
		diff := NormalizeAngle(desired - h.Rotation)
		maxTurn := MissileBarrageTurnRate * dt
		if diff > maxTurn {
			diff = maxTurn
		} else if diff < -maxTurn {
			diff = -maxTurn
		}
		h.Rotation += diff
	}

	h.VX = math.Cos(h.Rotation) * MissileBarrageSpeed
	h.VY = math.Sin(h.Rotation) * MissileBarrageSpeed
	h.X += h.VX * dt
	h.Y += h.VY * dt

	// Wrap position
	if h.X < 0 { h.X += ww } else if h.X > ww { h.X -= ww }
	if h.Y < 0 { h.Y += wh } else if h.Y > wh { h.Y -= wh }
}
