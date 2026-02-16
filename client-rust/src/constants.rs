// Game constants (must match server)
pub const WORLD_W: f64 = 4000.0;
pub const WORLD_H: f64 = 4000.0;
pub const PLAYER_RADIUS: f64 = 25.0;
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

pub const SHIP_COLORS: [ShipColor; 6] = [
    ShipColor { main: "#ff4444" }, // Rebel 1
    ShipColor { main: "#4488ff" }, // Rebel 2
    ShipColor { main: "#44ff44" }, // Rebel 3
    ShipColor { main: "#88ff88" }, // Star Destroyer
    ShipColor { main: "#aaaaff" }, // TIE 1
    ShipColor { main: "#aaaaff" }, // TIE 2
];

// New entity sizes (must match server)
pub const MOB_RADIUS: f64 = 25.0;       // TIE fighter radius
pub const SD_MOB_RADIUS: f64 = 185.0;   // Star Destroyer radius (broad-phase)
pub const ASTEROID_RADIUS: f64 = 50.0;
pub const ASTEROID_RENDER_SIZE: f64 = 120.0;
pub const PICKUP_RADIUS: f64 = 15.0;
pub const PICKUP_RENDER_SIZE: f64 = 30.0;

// Team colors
pub const TEAM_RED_COLOR: &str = "#ff4444";
pub const TEAM_BLUE_COLOR: &str = "#4488ff";

// Laser colors per ship type
pub const LASER_COLORS: [&str; 6] = [
    "#ff2222",
    "#2288ff",
    "#22ff22",
    "#22ff22", // Star Destroyer
    "#44ff44", // TIE 1
    "#44ff44", // TIE 2
];
