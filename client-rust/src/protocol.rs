use serde::{Deserialize, Serialize};

// Envelope wraps all messages
#[derive(Serialize, Deserialize, Debug, Clone)]
pub struct Envelope {
    pub t: String,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub d: Option<serde_json::Value>,
}

// Client -> Server input
#[derive(Serialize, Debug)]
pub struct ClientInput {
    pub mx: f64,
    pub my: f64,
    pub fire: bool,
    pub boost: bool,
    pub thresh: f64,
}

// Server -> Client: welcome
#[derive(Deserialize, Debug, Clone)]
pub struct WelcomeMsg {
    pub id: String,
    pub s: i32,
}

// Server -> Client: joined
#[derive(Deserialize, Debug, Clone)]
pub struct JoinedMsg {
    pub sid: String,
}

// Server -> Client: created
#[derive(Deserialize, Debug, Clone)]
pub struct CreatedMsg {
    pub sid: String,
}

// Server -> Client: player state
#[derive(Deserialize, Debug, Clone)]
pub struct PlayerState {
    pub id: String,
    pub n: String,
    pub x: f64,
    pub y: f64,
    pub r: f64,
    pub vx: f64,
    pub vy: f64,
    pub hp: i32,
    pub mhp: i32,
    pub s: i32,
    pub sc: i32,
    pub a: bool,
}

// Server -> Client: projectile state
#[derive(Deserialize, Debug, Clone)]
pub struct ProjectileState {
    pub id: String,
    pub x: f64,
    pub y: f64,
    pub r: f64,
    pub o: String,
}

// Server -> Client: mob state
#[derive(Deserialize, Debug, Clone)]
pub struct MobState {
    pub id: String,
    pub x: f64,
    pub y: f64,
    pub r: f64,
    pub vx: f64,
    pub vy: f64,
    pub hp: i32,
    pub mhp: i32,
    pub a: bool,
}

// Server -> Client: asteroid state
#[derive(Deserialize, Debug, Clone)]
pub struct AsteroidState {
    pub id: String,
    pub x: f64,
    pub y: f64,
    pub r: f64,
}

// Server -> Client: pickup state
#[derive(Deserialize, Debug, Clone)]
pub struct PickupState {
    pub id: String,
    pub x: f64,
    pub y: f64,
}

// Server -> Client: full game state
#[derive(Deserialize, Debug, Clone)]
pub struct GameStateMsg {
    pub p: Vec<PlayerState>,
    pub pr: Vec<ProjectileState>,
    #[serde(default)]
    pub m: Vec<MobState>,
    #[serde(default)]
    pub a: Vec<AsteroidState>,
    #[serde(default)]
    pub pk: Vec<PickupState>,
    pub tick: u64,
}

// Server -> Client: kill notification
#[derive(Deserialize, Debug, Clone)]
pub struct KillMsg {
    pub kid: String,
    pub kn: String,
    pub vid: String,
    pub vn: String,
}

// Server -> Client: death notification
#[derive(Deserialize, Debug, Clone)]
pub struct DeathMsg {
    pub kid: String,
    pub kn: String,
}

// Server -> Client: session list
#[derive(Deserialize, Debug, Clone)]
pub struct SessionInfo {
    pub id: String,
    pub name: String,
    pub players: i32,
}

// Server -> Client: session check response
#[derive(Deserialize, Debug, Clone)]
pub struct CheckedMsg {
    pub sid: String,
    pub exists: bool,
    #[serde(default)]
    pub name: String,
    #[serde(default)]
    pub players: i32,
}

// Server -> Client: error
#[derive(Deserialize, Debug, Clone)]
pub struct ErrorMsg {
    pub msg: String,
}
