package main

import (
	"crypto/rand"
	"math"
)

const (
	PlayerRadius   = 20.0
	PlayerMaxHP    = 100
	PlayerAccel    = 600.0  // pixels/s²
	PlayerMaxSpeed = 350.0  // pixels/s
	PlayerFriction = 0.97   // velocity multiplier per tick
	PlayerBoostMul = 1.6    // boost speed multiplier
	FireCooldown   = 0.15   // seconds between shots
	RespawnTime    = 3.0    // seconds before respawn
	WorldWidth     = 4000.0 // default world size
	WorldHeight    = 4000.0
	TurnSpeed      = 8.0 // radians/s max turn rate

	SpawnProtectionTime = 2.0 // seconds of spawn protection
	AssistWindow        = 5.0 // seconds within which damage counts as assist
)

// DamageRecord tracks recent damage from a player (for assist credit)
type DamageRecord struct {
	AttackerID string
	Time       float64
}

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
	TargetX  float64 // mouse world X (for distance calc)
	TargetY  float64 // mouse world Y (for distance calc)
	SlowThresh float64 // distance threshold for speed modulation

	// Match fields
	Team            int
	Ready           bool
	Kills           int
	Deaths          int
	Assists         int
	DamageDealt     int
	SpawnProtection float64
	RecentDamagers  []DamageRecord

	// Ship class & ability
	Class       ShipClass
	Ability     Ability
	AbilityUsed bool // input flag for ability activation

	// World bounds (set from match config)
	worldW float64
	worldH float64
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
		worldW:   WorldWidth,
		worldH:   WorldHeight,
	}
}

// NewPlayerWithWorld creates a player for a specific world size
func NewPlayerWithWorld(id, name string, shipType int, worldW, worldH float64) *Player {
	p := NewPlayer(id, name, shipType)
	p.worldW = worldW
	p.worldH = worldH
	p.X = worldW/4 + randFloat()*worldW/2
	p.Y = worldH/4 + randFloat()*worldH/2
	return p
}

// SpawnAtPosition spawns the player at a specific position (for team modes)
func (p *Player) SpawnAtPosition(x, y float64) {
	p.X = x
	p.Y = y
	p.VX = 0
	p.VY = 0
	p.HP = p.MaxHP
	p.Alive = true
	p.FireCD = 0
	p.RespawnT = 0
	p.SpawnProtection = SpawnProtectionTime
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

	// Tick spawn protection
	if p.SpawnProtection > 0 {
		p.SpawnProtection -= dt
		if p.SpawnProtection < 0 {
			p.SpawnProtection = 0
		}
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

	// Distance-based speed modulation
	dist2 := (p.TargetX-p.X)*(p.TargetX-p.X) + (p.TargetY-p.Y)*(p.TargetY-p.Y)
	thresh := p.SlowThresh
	if thresh < 20 {
		thresh = 20
	}
	const deadZone = 50.0
	var speedFactor float64 = 1.0
	if dist2 <= deadZone*deadZone {
		accel = 0
		speedFactor = 0
	} else if dist2 < thresh*thresh {
		dist := math.Sqrt(dist2)
		speedFactor = (dist - deadZone) / (thresh - deadZone)
		accel *= speedFactor
	}

	p.VX += math.Cos(p.Rotation) * accel
	p.VY += math.Sin(p.Rotation) * accel

	friction := PlayerFriction
	if speedFactor < 1.0 {
		friction = 0.95 + speedFactor*(PlayerFriction-0.95)
	}
	p.VX *= friction
	p.VY *= friction

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

	p.X += p.VX * dt
	p.Y += p.VY * dt

	ww := p.worldW
	wh := p.worldH
	if ww == 0 {
		ww = WorldWidth
	}
	if wh == 0 {
		wh = WorldHeight
	}
	if p.X < 0 {
		p.X += ww
	} else if p.X > ww {
		p.X -= ww
	}
	if p.Y < 0 {
		p.Y += wh
	} else if p.Y > wh {
		p.Y -= wh
	}

	if p.FireCD > 0 {
		p.FireCD -= dt
	}

	// Tick ability
	p.Ability.Update(dt)

	// Clean up old damage records
	if len(p.RecentDamagers) > 20 {
		p.RecentDamagers = p.RecentDamagers[len(p.RecentDamagers)-20:]
	}
}

// Respawn resets the player after death
func (p *Player) Respawn() {
	ww := p.worldW
	wh := p.worldH
	if ww == 0 {
		ww = WorldWidth
	}
	if wh == 0 {
		wh = WorldHeight
	}
	p.X = ww/4 + randFloat()*ww/2
	p.Y = wh/4 + randFloat()*wh/2
	p.VX = 0
	p.VY = 0
	p.HP = p.MaxHP
	p.Alive = true
	p.FireCD = 0
	p.RespawnT = 0
	p.SpawnProtection = SpawnProtectionTime
}

// TakeDamage reduces HP and returns true if player died
func (p *Player) TakeDamage(dmg int) bool {
	if !p.Alive {
		return false
	}
	if p.SpawnProtection > 0 {
		return false
	}
	// Shield absorption (Tank ability)
	dmg = p.Ability.AbsorbDamage(dmg)
	if dmg <= 0 {
		return false
	}
	p.HP -= dmg
	if p.HP <= 0 {
		p.HP = 0
		p.Alive = false
		p.RespawnT = RespawnTime
		p.Deaths++
		return true
	}
	return false
}

// RecordDamage tracks who damaged this player (for assist credit)
func (p *Player) RecordDamage(attackerID string, gameTime float64) {
	p.RecentDamagers = append(p.RecentDamagers, DamageRecord{
		AttackerID: attackerID,
		Time:       gameTime,
	})
}

// GetAssistIDs returns IDs of players who dealt damage within the assist window (excluding the killer)
func (p *Player) GetAssistIDs(killerID string, gameTime float64) []string {
	seen := make(map[string]bool)
	var assists []string
	for _, dr := range p.RecentDamagers {
		if dr.AttackerID == killerID {
			continue
		}
		if gameTime-dr.Time <= AssistWindow && !seen[dr.AttackerID] {
			seen[dr.AttackerID] = true
			assists = append(assists, dr.AttackerID)
		}
	}
	return assists
}

// CanFire returns true if the player can fire a projectile
func (p *Player) CanFire() bool {
	return p.Alive && p.Firing && p.FireCD <= 0
}

// ToState converts to protocol state
func (p *Player) ToState() PlayerState {
	vx := round1(p.VX)
	vy := round1(p.VY)
	acd := round1(p.Ability.Cooldown)
	return PlayerState{
		ID:      p.ID,
		Name:    p.Name,
		X:       round1(p.X),
		Y:       round1(p.Y),
		R:       round1(p.Rotation),
		VX:      &vx,
		VY:      &vy,
		HP:      p.HP,
		MaxHP:   p.MaxHP,
		Ship:    p.ShipType,
		Score:   p.Score,
		Alive:   p.Alive,
		Boost:   p.Boosting,
		Team:    p.Team,
		Class:   int(p.Class),
		AbilCD:  acd,
		AbilAct: p.Ability.Active,
		SpawnP:  p.SpawnProtection > 0,
	}
}

var randSrc uint64

func randFloat() float64 {
	randSrc ^= randSrc << 13
	randSrc ^= randSrc >> 7
	randSrc ^= randSrc << 17
	if randSrc == 0 {
		randSrc = 1
	}
	return float64(randSrc%10000) / 10000.0
}

func init() {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	for i, v := range b {
		randSrc |= uint64(v) << (uint(i) * 8)
	}
	if randSrc == 0 {
		randSrc = 1
	}
}
