// Game constants (must match server)
export const WORLD_W = 4000;
export const WORLD_H = 4000;
export const PLAYER_RADIUS = 20;
export const PROJECTILE_RADIUS = 4;
export const PLAYER_MAX_HP = 100;

// Rendering
export const SHIP_SIZE = 60;
export const STAR_COUNT = 300;
export const PARALLAX_FACTOR = 0.05;
export const ENGINE_PARTICLES = 15;

// Network
export const INPUT_RATE = 20; // Hz
export const RECONNECT_DELAY = 2000; // ms

// Colors per ship type
export const SHIP_COLORS = [
    { main: '#ff4444', accent: '#ff8888', engine: '#ff6600' }, // Red
    { main: '#4488ff', accent: '#88bbff', engine: '#00aaff' }, // Blue
    { main: '#44ff44', accent: '#88ff88', engine: '#00ff66' }, // Green
    { main: '#ffff44', accent: '#ffffaa', engine: '#ffaa00' }, // Yellow
];

// New entity sizes (must match server)
export const MOB_RADIUS = 20;
export const ASTEROID_RADIUS = 40;
export const ASTEROID_RENDER_SIZE = 120;
export const PICKUP_RADIUS = 15;
export const PICKUP_RENDER_SIZE = 30;

// Laser colors per ship type
export const LASER_COLORS = [
    '#ff2222',
    '#2288ff',
    '#22ff22',
    '#ffff22',
];
