package main

// HealZone is an area-of-effect heal placed by the Support class ability
type HealZone struct {
	ID      string
	X, Y    float64
	Radius  float64
	OwnerID string
	TeamID  int
	Life    float64
	Rate    float64 // HP/s healed to allies in range
}

// NewHealZone creates a heal zone at the given position
func NewHealZone(x, y float64, ownerID string, team int) *HealZone {
	return &HealZone{
		ID:      GenerateID(4),
		X:       x,
		Y:       y,
		Radius:  HealAuraRadius,
		OwnerID: ownerID,
		TeamID:  team,
		Life:    HealAuraDuration,
		Rate:    HealAuraRate,
	}
}

// Update ticks the heal zone lifetime, returns false when expired
func (hz *HealZone) Update(dt float64) bool {
	hz.Life -= dt
	return hz.Life > 0
}
