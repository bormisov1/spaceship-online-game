package main

import "math"

const (
	AsteroidRadius   = 50.0
	AsteroidMinSpeed = 60.0
	AsteroidMaxSpeed = 150.0
	AsteroidSpinMin  = 0.5
	AsteroidSpinMax  = 2.0
)

// Asteroid flies in a straight line across the map
type Asteroid struct {
	ID       string
	X, Y     float64
	VX, VY   float64
	Rotation float64
	Spin     float64
	Alive    bool
}

// NewAsteroid spawns an asteroid at a random edge heading inward
func NewAsteroid() *Asteroid {
	id := GenerateID(4)
	a := &Asteroid{
		ID:    id,
		Alive: true,
	}

	// Random speed
	speed := AsteroidMinSpeed + randFloat()*(AsteroidMaxSpeed-AsteroidMinSpeed)

	// Random spin
	a.Spin = AsteroidSpinMin + randFloat()*(AsteroidSpinMax-AsteroidSpinMin)
	if randFloat() < 0.5 {
		a.Spin = -a.Spin
	}

	// Pick random edge and aim inward
	edge := int(randFloat() * 4)
	switch edge {
	case 0: // left
		a.X = -AsteroidRadius
		a.Y = randFloat() * WorldHeight
		// Aim toward right half
		targetX := WorldWidth/2 + randFloat()*WorldWidth/2
		targetY := randFloat() * WorldHeight
		angle := math.Atan2(targetY-a.Y, targetX-a.X)
		a.VX = math.Cos(angle) * speed
		a.VY = math.Sin(angle) * speed
	case 1: // right
		a.X = WorldWidth + AsteroidRadius
		a.Y = randFloat() * WorldHeight
		targetX := randFloat() * WorldWidth / 2
		targetY := randFloat() * WorldHeight
		angle := math.Atan2(targetY-a.Y, targetX-a.X)
		a.VX = math.Cos(angle) * speed
		a.VY = math.Sin(angle) * speed
	case 2: // top
		a.X = randFloat() * WorldWidth
		a.Y = -AsteroidRadius
		targetX := randFloat() * WorldWidth
		targetY := WorldHeight/2 + randFloat()*WorldHeight/2
		angle := math.Atan2(targetY-a.Y, targetX-a.X)
		a.VX = math.Cos(angle) * speed
		a.VY = math.Sin(angle) * speed
	default: // bottom
		a.X = randFloat() * WorldWidth
		a.Y = WorldHeight + AsteroidRadius
		targetX := randFloat() * WorldWidth
		targetY := randFloat() * WorldHeight / 2
		angle := math.Atan2(targetY-a.Y, targetX-a.X)
		a.VX = math.Cos(angle) * speed
		a.VY = math.Sin(angle) * speed
	}

	a.Rotation = randFloat() * math.Pi * 2
	return a
}

// Update moves the asteroid and checks if it's off-map
func (a *Asteroid) Update(dt float64) {
	if !a.Alive {
		return
	}

	a.X += a.VX * dt
	a.Y += a.VY * dt
	a.Rotation += a.Spin * dt

	// Mark dead if fully off-map (no wrapping)
	margin := AsteroidRadius * 2
	if a.X < -margin || a.X > WorldWidth+margin ||
		a.Y < -margin || a.Y > WorldHeight+margin {
		a.Alive = false
	}
}

// ToState converts to protocol state
func (a *Asteroid) ToState() AsteroidState {
	return AsteroidState{
		ID: a.ID,
		X:  round1(a.X),
		Y:  round1(a.Y),
		R:  math.Round(a.Rotation*100) / 100,
	}
}
