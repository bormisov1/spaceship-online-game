use std::cell::RefCell;
use std::collections::HashMap;
use std::rc::Rc;

use crate::protocol::{PlayerState, ProjectileState, MobState, AsteroidState, PickupState};

#[derive(Debug, Clone, PartialEq)]
pub enum Phase {
    Lobby,
    Playing,
    Dead,
}

#[derive(Debug, Clone)]
pub struct KillFeedEntry {
    pub killer: String,
    pub victim: String,
    pub time: f64,
}

#[derive(Debug, Clone)]
pub struct DeathInfo {
    pub killer_name: String,
}

#[derive(Debug, Clone)]
pub struct Particle {
    pub x: f64,
    pub y: f64,
    pub vx: f64,
    pub vy: f64,
    pub life: f64,
    pub max_life: f64,
    pub size: f64,
    pub color: String,
    pub kind: ParticleKind,
}

#[derive(Debug, Clone, PartialEq)]
pub enum ParticleKind {
    Engine,
    Explosion,
}

#[derive(Debug, Clone)]
pub struct Explosion {
    pub x: f64,
    pub y: f64,
    pub radius: f64,
    pub max_radius: f64,
    pub life: f64,
    pub max_life: f64,
}

#[derive(Debug, Clone)]
pub struct TouchJoystick {
    pub start_x: f64,
    pub start_y: f64,
    pub current_x: f64,
    pub current_y: f64,
}

pub struct GameState {
    // Connection
    pub connected: bool,
    pub my_id: Option<String>,
    pub my_ship: i32,
    pub session_id: Option<String>,
    pub url_session_id: Option<String>,
    pub pending_name: Option<String>, // name saved before create, for auto-join

    // Game state from server
    pub players: HashMap<String, PlayerState>,
    pub projectiles: HashMap<String, ProjectileState>,
    pub mobs: HashMap<String, MobState>,
    pub asteroids: HashMap<String, AsteroidState>,
    pub pickups: HashMap<String, PickupState>,
    pub tick: u64,

    // Screen
    pub screen_w: f64,
    pub screen_h: f64,

    // Camera
    pub cam_x: f64,
    pub cam_y: f64,
    pub cam_zoom: f64,

    // Input
    pub mouse_x: f64,
    pub mouse_y: f64,
    pub mouse_world_x: f64,
    pub mouse_world_y: f64,
    pub firing: bool,
    pub boosting: bool,
    pub shift_pressed: bool,
    pub hyperspace_t: f64, // 0.0 = normal stars, 1.0 = full hyperspace
    pub hyperspace_locked_r: Option<f64>, // rotation locked when shift pressed

    // UI state
    pub phase: Phase,
    pub kill_feed: Vec<KillFeedEntry>,
    pub death_info: Option<DeathInfo>,

    // Controller
    pub controller_attached: bool,

    // Mobile
    pub is_mobile: bool,
    pub touch_joystick: Option<TouchJoystick>,

    // Debug
    pub debug_hitboxes: bool,

    // Effects
    pub particles: Vec<Particle>,
    pub explosions: Vec<Explosion>,

    // Interpolation: previous state for lerping between server updates
    pub prev_players: HashMap<String, PlayerState>,
    pub prev_mobs: HashMap<String, MobState>,
    pub prev_cam_x: f64,
    pub prev_cam_y: f64,
    pub interp_last_update: f64, // timestamp of last state update (ms)
    pub interp_interval: f64,    // estimated interval between updates (ms)
}

impl GameState {
    pub fn new() -> Self {
        Self {
            connected: false,
            my_id: None,
            my_ship: 0,
            session_id: None,
            url_session_id: None,
            pending_name: None,

            players: HashMap::new(),
            projectiles: HashMap::new(),
            mobs: HashMap::new(),
            asteroids: HashMap::new(),
            pickups: HashMap::new(),
            tick: 0,

            screen_w: 0.0,
            screen_h: 0.0,

            cam_x: 0.0,
            cam_y: 0.0,
            cam_zoom: 1.0,

            mouse_x: 0.0,
            mouse_y: 0.0,
            mouse_world_x: 0.0,
            mouse_world_y: 0.0,
            firing: false,
            boosting: false,
            shift_pressed: false,
            hyperspace_t: 0.0,
            hyperspace_locked_r: None,

            phase: Phase::Lobby,
            kill_feed: Vec::new(),
            death_info: None,

            controller_attached: false,

            is_mobile: false,
            touch_joystick: None,

            debug_hitboxes: false,

            particles: Vec::with_capacity(200),
            explosions: Vec::with_capacity(10),

            prev_players: HashMap::new(),
            prev_mobs: HashMap::new(),
            prev_cam_x: 0.0,
            prev_cam_y: 0.0,
            interp_last_update: 0.0,
            interp_interval: 33.33, // ~30 Hz default
        }
    }
}

pub type SharedState = Rc<RefCell<GameState>>;

pub fn new_shared_state() -> SharedState {
    Rc::new(RefCell::new(GameState::new()))
}
