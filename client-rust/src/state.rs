use std::cell::RefCell;
use std::collections::HashMap;
use std::rc::Rc;

use crate::protocol::{PlayerState, ProjectileState, MobState, AsteroidState, PickupState, HealZoneState, PlayerMatchResult, TeamPlayerInfo, LeaderboardEntry, FriendInfo, StoreItem};

#[derive(Debug, Clone, PartialEq)]
pub enum Phase {
    Lobby,
    MatchLobby,
    Countdown,
    Playing,
    Dead,
    Result,
}

#[derive(Debug, Clone, Copy, PartialEq)]
pub enum GameMode {
    FFA = 0,
    TDM = 1,
    CTF = 2,
    WaveSurvival = 3,
}

impl GameMode {
    pub fn from_i32(v: i32) -> Self {
        match v {
            1 => GameMode::TDM,
            2 => GameMode::CTF,
            3 => GameMode::WaveSurvival,
            _ => GameMode::FFA,
        }
    }

    pub fn name(&self) -> &'static str {
        match self {
            GameMode::FFA => "Free-For-All",
            GameMode::TDM => "Team Deathmatch",
            GameMode::CTF => "Capture the Flag",
            GameMode::WaveSurvival => "Wave Survival",
        }
    }
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
pub struct DamageNumber {
    pub x: f64,
    pub y: f64,
    pub text: String,
    pub color: &'static str,
    pub life: f64,
    pub max_life: f64,
    pub vy: f64,
    pub offset_x: f64,
}

#[derive(Debug, Clone)]
pub struct HitMarker {
    pub life: f64,
    pub max_life: f64,
}

#[derive(Debug, Clone)]
pub struct MobSpeech {
    pub mob_id: String,
    pub text: String,
    pub time: f64,  // timestamp when created (ms)
}

#[derive(Debug, Clone)]
pub struct TouchJoystick {
    pub start_x: f64,
    pub start_y: f64,
    pub current_x: f64,
    pub current_y: f64,
}

#[derive(Debug, Clone)]
pub struct XPNotification {
    pub xp_gained: i32,
    pub level: i32,
    pub prev_level: i32,
    pub leveled_up: bool,
}

#[derive(Debug, Clone)]
pub struct ChatMessage {
    pub from: String,
    pub text: String,
    pub team: bool,
    pub time: f64,
}

pub struct AchievementNotification {
    pub name: String,
    pub description: String,
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
    pub heal_zones: Vec<HealZoneState>,
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
    pub firing: bool,
    pub boosting: bool,
    pub ability_pressed: bool,
    pub shift_pressed: bool,
    pub hyperspace_t: f64, // 0.0 = normal stars, 1.0 = full hyperspace
    pub hyperspace_locked_r: Option<f64>, // rotation locked when shift pressed

    // UI state
    pub phase: Phase,
    pub kill_feed: Vec<KillFeedEntry>,
    pub death_info: Option<DeathInfo>,

    // Match state
    pub game_mode: GameMode,
    pub match_phase: i32,       // 0=lobby, 1=countdown, 2=playing, 3=result
    pub match_time_left: f64,
    pub countdown_time: f64,
    pub my_team: i32,           // 0=none, 1=red, 2=blue
    pub team_red_score: i32,
    pub team_blue_score: i32,
    pub is_ready: bool,
    pub team_red: Vec<TeamPlayerInfo>,
    pub team_blue: Vec<TeamPlayerInfo>,
    pub team_unassigned: Vec<TeamPlayerInfo>,
    pub lobby_player_count: i32,
    pub lobby_min_players: i32,
    pub match_result: Option<(i32, Vec<PlayerMatchResult>, f64)>, // (winner_team, players, duration)

    // Auth
    pub auth_token: Option<String>,
    pub auth_username: Option<String>,
    pub auth_player_id: i64,
    pub auth_level: i32,
    pub auth_xp: i32,
    pub auth_xp_next: i32,     // XP needed for next level
    pub auth_kills: i32,
    pub auth_deaths: i32,
    pub auth_wins: i32,
    pub auth_losses: i32,

    // XP notification (after match)
    pub xp_notification: Option<XPNotification>,
    pub xp_notification_time: f64,

    // Leaderboard
    pub leaderboard: Vec<LeaderboardEntry>,

    // Achievement notifications (queue, show one at a time)
    pub achievement_queue: Vec<AchievementNotification>,
    pub achievement_show_time: f64,

    // Store & Credits
    pub auth_credits: i32,
    pub store_items: Vec<StoreItem>,
    pub owned_skins: Vec<String>,
    pub equipped_skin: String,
    pub equipped_trail: String,
    pub store_open: bool,

    // Friends
    pub friends: Vec<FriendInfo>,
    pub friend_requests: Vec<FriendInfo>,

    // Chat
    pub chat_messages: Vec<ChatMessage>,
    pub chat_open: bool,

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

    // Screen shake
    pub shake_x: f64,
    pub shake_y: f64,
    pub shake_intensity: f64,
    pub shake_decay: f64,

    // Damage numbers (world-space floating text)
    pub damage_numbers: Vec<DamageNumber>,

    // Hit markers (screen-space, brief flash when own shot connects)
    pub hit_markers: Vec<HitMarker>,

    // Mob speech bubbles
    pub mob_speech: Vec<MobSpeech>,

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
            heal_zones: Vec::new(),
            tick: 0,

            screen_w: 0.0,
            screen_h: 0.0,

            cam_x: 0.0,
            cam_y: 0.0,
            cam_zoom: 1.0,

            mouse_x: 0.0,
            mouse_y: 0.0,
            firing: false,
            boosting: false,
            ability_pressed: false,
            shift_pressed: false,
            hyperspace_t: 0.0,
            hyperspace_locked_r: None,

            phase: Phase::Lobby,
            kill_feed: Vec::new(),
            death_info: None,

            game_mode: GameMode::FFA,
            match_phase: 0,
            match_time_left: 0.0,
            countdown_time: 0.0,
            my_team: 0,
            team_red_score: 0,
            team_blue_score: 0,
            is_ready: false,
            team_red: Vec::new(),
            team_blue: Vec::new(),
            team_unassigned: Vec::new(),
            lobby_player_count: 0,
            lobby_min_players: 0,
            match_result: None,

            auth_token: None,
            auth_username: None,
            auth_player_id: 0,
            auth_level: 1,
            auth_xp: 0,
            auth_xp_next: 100,
            auth_kills: 0,
            auth_deaths: 0,
            auth_wins: 0,
            auth_losses: 0,

            xp_notification: None,
            xp_notification_time: 0.0,

            leaderboard: Vec::new(),

            achievement_queue: Vec::new(),
            achievement_show_time: 0.0,

            auth_credits: 0,
            store_items: Vec::new(),
            owned_skins: Vec::new(),
            equipped_skin: String::new(),
            equipped_trail: String::new(),
            store_open: false,

            friends: Vec::new(),
            friend_requests: Vec::new(),

            chat_messages: Vec::new(),
            chat_open: false,

            controller_attached: false,

            is_mobile: false,
            touch_joystick: None,

            debug_hitboxes: false,

            particles: Vec::with_capacity(200),
            explosions: Vec::with_capacity(10),

            shake_x: 0.0,
            shake_y: 0.0,
            shake_intensity: 0.0,
            shake_decay: 0.0,

            damage_numbers: Vec::with_capacity(30),
            hit_markers: Vec::with_capacity(5),
            mob_speech: Vec::with_capacity(8),

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
