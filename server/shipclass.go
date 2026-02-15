package main

// ShipClass identifies the class of ship
type ShipClass int

const (
	ClassFighter ShipClass = 0
	ClassTank    ShipClass = 1
	ClassScout   ShipClass = 2
	ClassSupport ShipClass = 3
)

// ShipClassDef holds the stats for a ship class
type ShipClassDef struct {
	MaxHP      int
	Accel      float64
	MaxSpeed   float64
	BoostMul   float64
	FireCD     float64
	ProjDamage int
	ProjSpeed  float64
	ProjCount  int     // number of projectiles per shot
	ProjSpread float64 // spread angle in radians (for shotgun)
	Radius     float64
	TurnSpeed  float64
}

var ShipClasses = [4]ShipClassDef{
	// Fighter: balanced, standard stats
	{
		MaxHP: 100, Accel: 600, MaxSpeed: 350, BoostMul: 1.6,
		FireCD: 0.15, ProjDamage: 20, ProjSpeed: 800,
		ProjCount: 1, ProjSpread: 0, Radius: 20, TurnSpeed: 8.0,
	},
	// Tank: slow, tanky, shotgun spread
	{
		MaxHP: 200, Accel: 350, MaxSpeed: 220, BoostMul: 1.4,
		FireCD: 0.4, ProjDamage: 15, ProjSpeed: 700,
		ProjCount: 5, ProjSpread: 0.3, Radius: 25, TurnSpeed: 6.0,
	},
	// Scout: fast, fragile, rapid fire
	{
		MaxHP: 60, Accel: 800, MaxSpeed: 480, BoostMul: 1.8,
		FireCD: 0.1, ProjDamage: 12, ProjSpeed: 900,
		ProjCount: 1, ProjSpread: 0, Radius: 16, TurnSpeed: 10.0,
	},
	// Support: medium, heal ability
	{
		MaxHP: 120, Accel: 500, MaxSpeed: 300, BoostMul: 1.5,
		FireCD: 0.2, ProjDamage: 15, ProjSpeed: 800,
		ProjCount: 1, ProjSpread: 0, Radius: 20, TurnSpeed: 8.0,
	},
}

// GetClassDef returns the definition for a ship class
func GetClassDef(class ShipClass) ShipClassDef {
	if class < 0 || int(class) >= len(ShipClasses) {
		return ShipClasses[ClassFighter]
	}
	return ShipClasses[class]
}
