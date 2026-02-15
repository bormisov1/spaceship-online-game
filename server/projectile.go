package main

import "math"

const (
	ProjectileSpeed    = 800.0 // pixels/s
	ProjectileLifetime = 2.0   // seconds
	ProjectileRadius   = 4.0
	ProjectileDamage   = 20
	ProjectileOffset   = 30.0 // spawn distance from ship center
)

// Projectile represents a laser projectile
type Projectile struct {
	ID       string
	OwnerID  string
	X, Y     float64
	VX, VY   float64
	Rotation float64
	Life     float64
	Damage   int
	Alive    bool
}

// NewProjectile creates a projectile from a player's position and facing direction
func NewProjectile(owner *Player) *Projectile {
	id := GenerateID(3)
	vx := math.Cos(owner.Rotation) * ProjectileSpeed
	vy := math.Sin(owner.Rotation) * ProjectileSpeed
	return &Projectile{
		ID:       id,
		OwnerID:  owner.ID,
		X:        owner.X + math.Cos(owner.Rotation)*ProjectileOffset,
		Y:        owner.Y + math.Sin(owner.Rotation)*ProjectileOffset,
		VX:       vx + owner.VX*0.3, // inherit some of ship velocity
		VY:       vy + owner.VY*0.3,
		Rotation: owner.Rotation,
		Life:     ProjectileLifetime,
		Damage:   ProjectileDamage,
		Alive:    true,
	}
}

// NewMobProjectile creates a projectile from a mob's position and facing direction
func NewMobProjectile(mob *Mob) *Projectile {
	id := GenerateID(3)
	vx := math.Cos(mob.Rotation) * ProjectileSpeed
	vy := math.Sin(mob.Rotation) * ProjectileSpeed
	return &Projectile{
		ID:       id,
		OwnerID:  mob.ID,
		X:        mob.X + math.Cos(mob.Rotation)*ProjectileOffset,
		Y:        mob.Y + math.Sin(mob.Rotation)*ProjectileOffset,
		VX:       vx + mob.VX*0.3,
		VY:       vy + mob.VY*0.3,
		Rotation: mob.Rotation,
		Life:     ProjectileLifetime,
		Damage:   mob.ProjDamage,
		Alive:    true,
	}
}

// Update moves the projectile one tick
func (p *Projectile) Update(dt float64) {
	if !p.Alive {
		return
	}
	p.X += p.VX * dt
	p.Y += p.VY * dt
	p.Life -= dt

	// Wrap around world
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

	if p.Life <= 0 {
		p.Alive = false
	}
}

// ToState converts to protocol state
func (p *Projectile) ToState() ProjectileState {
	return ProjectileState{
		ID:    p.ID,
		X:     round1(p.X),
		Y:     round1(p.Y),
		R:     round1(p.Rotation),
		Owner: p.OwnerID,
	}
}
