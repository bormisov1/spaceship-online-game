package main

const (
	PickupRadius  = 15.0
	PickupHeal    = 20
	PickupTimeout = 30.0
)

// Pickup is a health orb that heals on contact
type Pickup struct {
	ID    string
	X, Y  float64
	Life  float64
	Alive bool
}

// NewPickup spawns a pickup at a random position away from edges
func NewPickup() *Pickup {
	return &Pickup{
		ID:    GenerateID(4),
		X:     50 + randFloat()*3900,
		Y:     50 + randFloat()*3900,
		Life:  PickupTimeout,
		Alive: true,
	}
}

// Update ticks down the pickup lifetime
func (p *Pickup) Update(dt float64) {
	if !p.Alive {
		return
	}
	p.Life -= dt
	if p.Life <= 0 {
		p.Alive = false
	}
}

// ToState converts to protocol state
func (p *Pickup) ToState() PickupState {
	return PickupState{
		ID: p.ID,
		X:  p.X,
		Y:  p.Y,
	}
}
