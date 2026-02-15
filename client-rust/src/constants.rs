// Game constants (must match server)
pub const WORLD_W: f64 = 4000.0;
pub const WORLD_H: f64 = 4000.0;
pub const PLAYER_RADIUS: f64 = 20.0;
pub const PROJECTILE_RADIUS: f64 = 4.0;
// Rendering
pub const SHIP_SIZE: f64 = 60.0;

// Network
pub const INPUT_RATE: u32 = 20; // Hz
pub const RECONNECT_DELAY: u32 = 2000; // ms

// Colors per ship type
pub struct ShipColor {
    pub main: &'static str,
}

pub const SHIP_COLORS: [ShipColor; 4] = [
    ShipColor { main: "#ff4444" }, // Red
    ShipColor { main: "#4488ff" }, // Blue
    ShipColor { main: "#44ff44" }, // Green
    ShipColor { main: "#ffff44" }, // Yellow
];

// New entity sizes (must match server)
pub const MOB_RADIUS: f64 = 20.0;
pub const ASTEROID_RADIUS: f64 = 40.0;
pub const ASTEROID_RENDER_SIZE: f64 = 120.0;
pub const PICKUP_RADIUS: f64 = 15.0;
pub const PICKUP_RENDER_SIZE: f64 = 30.0;

// Laser colors per ship type
pub const LASER_COLORS: [&str; 4] = [
    "#ff2222",
    "#2288ff",
    "#22ff22",
    "#ffff22",
];
