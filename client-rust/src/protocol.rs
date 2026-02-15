use serde::{Deserialize, Serialize};

// Envelope wraps all messages
#[derive(Serialize, Deserialize, Debug, Clone)]
pub struct Envelope {
    pub t: String,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub d: Option<serde_json::Value>,
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

// Server -> Client: player state (vx/vy omitted when unchanged via delta compression)
#[derive(Deserialize, Debug, Clone)]
pub struct PlayerState {
    pub id: String,
    pub n: String,
    pub x: f64,
    pub y: f64,
    pub r: f64,
    pub vx: Option<f64>,
    pub vy: Option<f64>,
    pub hp: i32,
    pub mhp: i32,
    pub s: i32,
    pub sc: i32,
    pub a: bool,
    #[serde(default)]
    pub b: bool,
    #[serde(default)]
    pub tm: i32,
    #[serde(default)]
    pub cl: i32,
    #[serde(default)]
    pub acd: f64,
    #[serde(default)]
    pub aact: bool,
    #[serde(default)]
    pub sp: bool,
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

// Server -> Client: mob state (vx/vy omitted when unchanged via delta compression)
#[derive(Deserialize, Debug, Clone)]
pub struct MobState {
    pub id: String,
    pub x: f64,
    pub y: f64,
    pub r: f64,
    pub vx: Option<f64>,
    pub vy: Option<f64>,
    pub hp: i32,
    pub mhp: i32,
    #[serde(default = "default_mob_ship")]
    pub s: i32,
    pub a: bool,
}

fn default_mob_ship() -> i32 { 3 }

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
    #[serde(default)]
    pub mp: i32,
    #[serde(default)]
    pub tl: f64,
    #[serde(default)]
    pub trs: i32,
    #[serde(default)]
    pub tbs: i32,
    #[serde(default)]
    pub hz: Vec<HealZoneState>,
}

// Server -> Client: heal zone state
#[derive(Deserialize, Debug, Clone)]
pub struct HealZoneState {
    pub id: String,
    pub x: f64,
    pub y: f64,
    pub r: f64, // radius
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
    pub kn: String,
}

// Server -> Client: session list
#[derive(Deserialize, Debug, Clone)]
pub struct SessionInfo {
    pub id: String,
    pub name: String,
    pub players: i32,
    #[serde(default)]
    pub mode: i32,
    #[serde(default)]
    pub phase: i32,
}

// Server -> Client: session check response
#[derive(Deserialize, Debug, Clone)]
pub struct CheckedMsg {
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

// Server -> Client: hit event (damage dealt)
#[derive(Deserialize, Debug, Clone)]
pub struct HitMsg {
    pub x: f64,
    pub y: f64,
    pub dmg: i32,
    pub vid: String,  // victim ID
    pub aid: String,  // attacker ID
}

// Server -> Client: mob speech bubble
#[derive(Deserialize, Debug, Clone)]
pub struct MobSayMsg {
    pub mid: String,  // mob ID
    pub text: String, // phrase text (with emoji)
}

// Server -> Client: match phase changed
#[derive(Deserialize, Debug, Clone)]
pub struct MatchPhaseMsg {
    pub phase: i32,
    pub mode: i32,
    #[serde(default)]
    pub time_left: f64,
    #[serde(default)]
    pub countdown: f64,
}

// Server -> Client: match result
#[derive(Deserialize, Debug, Clone)]
pub struct MatchResultMsg {
    pub winner_team: i32,
    pub players: Vec<PlayerMatchResult>,
    pub duration: f64,
}

// Player stats in match result
#[derive(Deserialize, Debug, Clone)]
pub struct PlayerMatchResult {
    pub id: String,
    pub n: String,
    pub tm: i32,
    pub k: i32,
    pub d: i32,
    pub a: i32,
    pub sc: i32,
    #[serde(default)]
    pub mvp: bool,
}

// Server -> Client: team roster update
#[derive(Deserialize, Debug, Clone)]
pub struct TeamUpdateMsg {
    pub red: Vec<TeamPlayerInfo>,
    pub blue: Vec<TeamPlayerInfo>,
}

// Player info on a team
#[derive(Deserialize, Debug, Clone)]
pub struct TeamPlayerInfo {
    pub id: String,
    pub n: String,
    pub ready: bool,
}

// Server -> Client: auth success
#[derive(Deserialize, Debug, Clone)]
pub struct AuthOKMsg {
    pub token: String,
    pub username: String,
    pub pid: i64,
    #[serde(default)]
    pub guest: bool,
}

// Server -> Client: profile/stats data
#[derive(Deserialize, Debug, Clone)]
pub struct ProfileDataMsg {
    pub username: String,
    pub level: i32,
    pub xp: i32,
    pub kills: i32,
    pub deaths: i32,
    pub wins: i32,
    pub losses: i32,
    pub playtime: f64,
}
