// Game constants (must match server)
pub const WORLD_W: f64 = 4000.0;
pub const WORLD_H: f64 = 4000.0;
pub const PLAYER_RADIUS: f64 = 20.0;
pub const PROJECTILE_RADIUS: f64 = 4.0;
pub const PLAYER_MAX_HP: i32 = 100;

// Rendering
pub const SHIP_SIZE: f64 = 60.0;
pub const STAR_COUNT: usize = 300;
pub const PARALLAX_FACTOR: f64 = 0.05;
pub const ENGINE_PARTICLES: usize = 15;

// Network
pub const INPUT_RATE: u32 = 20; // Hz
pub const RECONNECT_DELAY: u32 = 2000; // ms

// Colors per ship type
pub struct ShipColor {
    pub main: &'static str,
    pub accent: &'static str,
    pub engine: &'static str,
}

pub const SHIP_COLORS: [ShipColor; 4] = [
    ShipColor { main: "#ff4444", accent: "#ff8888", engine: "#ff6600" }, // Red
    ShipColor { main: "#4488ff", accent: "#88bbff", engine: "#00aaff" }, // Blue
    ShipColor { main: "#44ff44", accent: "#88ff88", engine: "#00ff66" }, // Green
    ShipColor { main: "#ffff44", accent: "#ffffaa", engine: "#ffaa00" }, // Yellow
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
