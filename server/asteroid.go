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
	worldW   float64
	worldH   float64
}

// NewAsteroid spawns an asteroid at a random edge heading inward
func NewAsteroid(worldW, worldH float64) *Asteroid {
	if worldW == 0 { worldW = WorldWidth }
	if worldH == 0 { worldH = WorldHeight }
	id := GenerateID(4)
	a := &Asteroid{
		ID:     id,
		Alive:  true,
		worldW: worldW,
		worldH: worldH,
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
		a.Y = randFloat() * worldH
		// Aim toward right half
		targetX := worldW/2 + randFloat()*worldW/2
		targetY := randFloat() * worldH
		angle := math.Atan2(targetY-a.Y, targetX-a.X)
		a.VX = math.Cos(angle) * speed
		a.VY = math.Sin(angle) * speed
	case 1: // right
		a.X = worldW + AsteroidRadius
		a.Y = randFloat() * worldH
		targetX := randFloat() * worldW / 2
		targetY := randFloat() * worldH
		angle := math.Atan2(targetY-a.Y, targetX-a.X)
		a.VX = math.Cos(angle) * speed
		a.VY = math.Sin(angle) * speed
	case 2: // top
		a.X = randFloat() * worldW
		a.Y = -AsteroidRadius
		targetX := randFloat() * worldW
		targetY := worldH/2 + randFloat()*worldH/2
		angle := math.Atan2(targetY-a.Y, targetX-a.X)
		a.VX = math.Cos(angle) * speed
		a.VY = math.Sin(angle) * speed
	default: // bottom
		a.X = randFloat() * worldW
		a.Y = worldH + AsteroidRadius
		targetX := randFloat() * worldW
		targetY := randFloat() * worldH / 2
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
	ww := a.worldW
	wh := a.worldH
	if ww == 0 { ww = WorldWidth }
	if wh == 0 { wh = WorldHeight }
	margin := AsteroidRadius * 2
	if a.X < -margin || a.X > ww+margin ||
		a.Y < -margin || a.Y > wh+margin {
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
