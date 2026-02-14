use std::cell::RefCell;
use std::rc::Rc;
use wasm_bindgen::prelude::*;
use wasm_bindgen::JsCast;
use web_sys::{WebSocket, MessageEvent, CloseEvent, ErrorEvent};
use leptos::prelude::Set;
use crate::state::{SharedState, Phase};
use crate::protocol::*;
use crate::effects;

pub struct Network {
    ws: Option<WebSocket>,
    pub state: SharedState,
    phase_signal: leptos::prelude::RwSignal<Phase>,
    sessions_signal: leptos::prelude::RwSignal<Vec<SessionInfo>>,
    checked_signal: leptos::prelude::RwSignal<Option<CheckedMsg>>,
    // Store closures to prevent them from being dropped
    _on_open: Option<Closure<dyn FnMut()>>,
    _on_message: Option<Closure<dyn FnMut(MessageEvent)>>,
    _on_close: Option<Closure<dyn FnMut(CloseEvent)>>,
    _on_error: Option<Closure<dyn FnMut(ErrorEvent)>>,
}

pub type SharedNetwork = Rc<RefCell<Network>>;

impl Network {
    pub fn new(
        state: SharedState,
        phase_signal: leptos::prelude::RwSignal<Phase>,
        sessions_signal: leptos::prelude::RwSignal<Vec<SessionInfo>>,
        checked_signal: leptos::prelude::RwSignal<Option<CheckedMsg>>,
    ) -> SharedNetwork {
        let net = Rc::new(RefCell::new(Network {
            ws: None,
            state,
            phase_signal,
            sessions_signal,
            checked_signal,
            _on_open: None,
            _on_message: None,
            _on_close: None,
            _on_error: None,
        }));
        net
    }

    pub fn connect(net: &SharedNetwork) {
        let window = web_sys::window().unwrap();
        let location = window.location();
        let protocol = location.protocol().unwrap_or_default();
        let host = location.host().unwrap_or_default();
        let ws_proto = if protocol == "https:" { "wss:" } else { "ws:" };
        let url = format!("{}//{}/ws", ws_proto, host);

        let ws = WebSocket::new(&url).unwrap();

        // on open
        let state_clone = net.borrow().state.clone();
        let net_clone = net.clone();
        let on_open = Closure::wrap(Box::new(move || {
            state_clone.borrow_mut().connected = true;
            web_sys::console::log_1(&"WebSocket connected".into());
            // Check URL session if present
            let url_sid = state_clone.borrow().url_session_id.clone();
            if let Some(sid) = url_sid {
                Network::send_raw(&net_clone, "check", &serde_json::json!({"sid": sid}));
            }
        }) as Box<dyn FnMut()>);

        // on message
        let state_clone = net.borrow().state.clone();
        let phase_signal = net.borrow().phase_signal;
        let sessions_signal = net.borrow().sessions_signal;
        let checked_signal = net.borrow().checked_signal;
        let on_message = Closure::wrap(Box::new(move |e: MessageEvent| {
            if let Some(text) = e.data().as_string() {
                if let Ok(env) = serde_json::from_str::<Envelope>(&text) {
                    handle_message(&state_clone, phase_signal, sessions_signal, checked_signal, env);
                }
            }
        }) as Box<dyn FnMut(MessageEvent)>);

        // on close
        let state_clone = net.borrow().state.clone();
        let net_clone = net.clone();
        let on_close = Closure::wrap(Box::new(move |_: CloseEvent| {
            state_clone.borrow_mut().connected = false;
            web_sys::console::log_1(&"WebSocket closed, reconnecting...".into());
            let net_clone2 = net_clone.clone();
            let _ = gloo_timers::callback::Timeout::new(crate::constants::RECONNECT_DELAY, move || {
                Network::connect(&net_clone2);
            });
        }) as Box<dyn FnMut(CloseEvent)>);

        // on error
        let on_error = Closure::wrap(Box::new(move |e: ErrorEvent| {
            web_sys::console::error_1(&format!("WebSocket error: {:?}", e.message()).into());
        }) as Box<dyn FnMut(ErrorEvent)>);

        ws.set_onopen(Some(on_open.as_ref().unchecked_ref()));
        ws.set_onmessage(Some(on_message.as_ref().unchecked_ref()));
        ws.set_onclose(Some(on_close.as_ref().unchecked_ref()));
        ws.set_onerror(Some(on_error.as_ref().unchecked_ref()));

        let mut net_mut = net.borrow_mut();
        net_mut.ws = Some(ws);
        net_mut._on_open = Some(on_open);
        net_mut._on_message = Some(on_message);
        net_mut._on_close = Some(on_close);
        net_mut._on_error = Some(on_error);
    }

    pub fn send_raw(net: &SharedNetwork, msg_type: &str, data: &serde_json::Value) {
        let net_ref = net.borrow();
        if let Some(ws) = &net_ref.ws {
            if ws.ready_state() == 1 {
                let env = serde_json::json!({"t": msg_type, "d": data});
                let _ = ws.send_with_str(&env.to_string());
            }
        }
    }

    pub fn send_input(net: &SharedNetwork) {
        let state = net.borrow().state.clone();
        let s = state.borrow();
        if s.phase != Phase::Playing || s.my_id.is_none() {
            return;
        }
        if s.controller_attached {
            return;
        }

        let zoom = s.cam_zoom;
        let mut mx = (s.mouse_x - s.screen_w / 2.0) / zoom + s.cam_x;
        let mut my = (s.mouse_y - s.screen_h / 2.0) / zoom + s.cam_y;

        // Mobile auto-aim (only when joystick is actively being used)
        if s.is_mobile {
            let jdx = s.mouse_x - s.screen_w / 2.0;
            let jdy = s.mouse_y - s.screen_h / 2.0;
            let jdist = (jdx * jdx + jdy * jdy).sqrt();

            if jdist > 5.0 {
                if let Some(my_id) = &s.my_id {
                    if let Some(me) = s.players.get(my_id) {
                        if me.a {
                            let aim_angle = jdy.atan2(jdx);

                            let orbit_r: f64 = 360.0;
                            let detect_r: f64 = 150.0;
                            let orbit_x = me.x + aim_angle.cos() * orbit_r;
                            let orbit_y = me.y + aim_angle.sin() * orbit_r;

                            let mut best_dist = detect_r * detect_r;
                            let mut best_target: Option<(f64, f64)> = None;

                            for (id, p) in &s.players {
                                if Some(id) == s.my_id.as_ref() || !p.a { continue; }
                                let dx = p.x - orbit_x;
                                let dy = p.y - orbit_y;
                                let d2 = dx * dx + dy * dy;
                                if d2 <= best_dist {
                                    best_dist = d2;
                                    best_target = Some((p.x, p.y));
                                }
                            }
                            for (_, m) in &s.mobs {
                                if !m.a { continue; }
                                let dx = m.x - orbit_x;
                                let dy = m.y - orbit_y;
                                let d2 = dx * dx + dy * dy;
                                if d2 <= best_dist {
                                    best_dist = d2;
                                    best_target = Some((m.x, m.y));
                                }
                            }

                            if let Some((tx, ty)) = best_target {
                                mx = tx;
                                my = ty;
                            }
                        }
                    }
                }
            }
        }

        let thresh = s.screen_w.min(s.screen_h) / (8.0 * zoom);

        // During hyperspace (shift), lock steering to rotation captured at shift press
        if s.shift_pressed {
            if let Some(locked_r) = s.hyperspace_locked_r {
                if let Some(my_id) = &s.my_id {
                    if let Some(me) = s.players.get(my_id) {
                        mx = me.x + locked_r.cos() * 1000.0;
                        my = me.y + locked_r.sin() * 1000.0;
                    }
                }
            }
        }
        drop(s);

        let input = ClientInput { mx, my, fire: state.borrow().firing, boost: state.borrow().boosting, thresh };
        Network::send_raw(net, "input", &serde_json::to_value(&input).unwrap());
    }

    pub fn list_sessions(net: &SharedNetwork) {
        Network::send_raw(net, "list", &serde_json::json!({}));
    }

    pub fn create_session(net: &SharedNetwork, name: &str, session_name: &str) {
        Network::send_raw(net, "create", &serde_json::json!({"name": name, "sname": session_name}));
    }

    pub fn join_session(net: &SharedNetwork, name: &str, session_id: &str) {
        Network::send_raw(net, "join", &serde_json::json!({"name": name, "sid": session_id}));
    }

    pub fn check_session(net: &SharedNetwork, sid: &str) {
        Network::send_raw(net, "check", &serde_json::json!({"sid": sid}));
    }

    pub fn send_leave(net: &SharedNetwork) {
        Network::send_raw(net, "leave", &serde_json::json!({}));
    }
}

fn handle_message(
    state: &SharedState,
    phase_signal: leptos::prelude::RwSignal<Phase>,
    sessions_signal: leptos::prelude::RwSignal<Vec<SessionInfo>>,
    checked_signal: leptos::prelude::RwSignal<Option<CheckedMsg>>,
    env: Envelope,
) {
    let data = env.d.unwrap_or(serde_json::Value::Null);
    match env.t.as_str() {
        "state" => {
            if let Ok(gs) = serde_json::from_value::<GameStateMsg>(data) {
                handle_state(state, &phase_signal, gs);
            }
        }
        "welcome" => {
            if let Ok(w) = serde_json::from_value::<WelcomeMsg>(data) {
                let mut s = state.borrow_mut();
                s.my_id = Some(w.id);
                s.my_ship = w.s;
                s.phase = Phase::Playing;
                phase_signal.set(Phase::Playing);
            }
        }
        "joined" => {
            if let Ok(j) = serde_json::from_value::<JoinedMsg>(data) {
                let mut s = state.borrow_mut();
                s.session_id = Some(j.sid.clone());
                // Update URL
                let window = web_sys::window().unwrap();
                let _ = window.history().unwrap().push_state_with_url(
                    &wasm_bindgen::JsValue::NULL,
                    "",
                    Some(&format!("/rust/{}", j.sid)),
                );
            }
        }
        "created" => {
            if let Ok(c) = serde_json::from_value::<CreatedMsg>(data) {
                // Navigate to session URL
                let window = web_sys::window().unwrap();
                let _ = window.location().set_href(&format!("/rust/{}", c.sid));
            }
        }
        "sessions" => {
            if let Ok(sessions) = serde_json::from_value::<Vec<SessionInfo>>(data) {
                sessions_signal.set(sessions);
            }
        }
        "kill" => {
            if let Ok(k) = serde_json::from_value::<KillMsg>(data) {
                let mut s = state.borrow_mut();
                let now = web_sys::window().unwrap().performance().unwrap().now();
                s.kill_feed.push(crate::state::KillFeedEntry {
                    killer: k.kn,
                    victim: k.vn.clone(),
                    time: now,
                });
                if s.kill_feed.len() > 5 {
                    s.kill_feed.remove(0);
                }
                // Add explosion at victim location
                let victim_pos = s.players.get(&k.vid).map(|p| (p.x, p.y))
                    .or_else(|| s.mobs.get(&k.vid).map(|m| (m.x, m.y)));
                if let Some((vx, vy)) = victim_pos {
                    let mut particles = std::mem::take(&mut s.particles);
                    let mut explosions = std::mem::take(&mut s.explosions);
                    effects::add_explosion(&mut particles, &mut explosions, vx, vy);
                    s.particles = particles;
                    s.explosions = explosions;
                }
            }
        }
        "death" => {
            if let Ok(d) = serde_json::from_value::<DeathMsg>(data) {
                let mut s = state.borrow_mut();
                s.death_info = Some(crate::state::DeathInfo { killer_name: d.kn });
                s.phase = Phase::Dead;
                phase_signal.set(Phase::Dead);
            }
        }
        "checked" => {
            if let Ok(c) = serde_json::from_value::<CheckedMsg>(data) {
                checked_signal.set(Some(c));
            }
        }
        "ctrl_on" => {
            state.borrow_mut().controller_attached = true;
        }
        "ctrl_off" => {
            state.borrow_mut().controller_attached = false;
        }
        "error" => {
            if let Ok(e) = serde_json::from_value::<ErrorMsg>(data) {
                web_sys::console::error_1(&format!("Server error: {}", e.msg).into());
            }
        }
        _ => {}
    }
}

fn handle_state(state: &SharedState, phase_signal: &leptos::prelude::RwSignal<Phase>, gs: GameStateMsg) {
    let mut s = state.borrow_mut();
    let now = web_sys::window().unwrap().performance().unwrap().now();

    // Store previous state for interpolation
    s.prev_players = s.players.clone();
    s.prev_projectiles = s.projectiles.clone();
    s.prev_mobs = s.mobs.clone();
    s.prev_asteroids = s.asteroids.clone();
    s.prev_pickups = s.pickups.clone();
    s.last_state_time = now;

    // Update current state
    s.players.clear();
    for p in gs.p {
        s.players.insert(p.id.clone(), p);
    }

    s.projectiles.clear();
    for pr in gs.pr {
        s.projectiles.insert(pr.id.clone(), pr);
    }

    s.mobs.clear();
    for m in gs.m {
        s.mobs.insert(m.id.clone(), m);
    }

    s.asteroids.clear();
    for a in gs.a {
        s.asteroids.insert(a.id.clone(), a);
    }

    s.pickups.clear();
    for pk in gs.pk {
        s.pickups.insert(pk.id.clone(), pk);
    }

    s.tick = gs.tick;

    // Update camera
    if let Some(my_id) = &s.my_id {
        if let Some(me) = s.players.get(my_id) {
            let me_x = me.x;
            let me_y = me.y;
            let me_alive = me.a;
            drop(me);
            s.cam_x = me_x;
            s.cam_y = me_y;

            if !me_alive && s.phase == Phase::Playing {
                s.phase = Phase::Dead;
                phase_signal.set(Phase::Dead);
            } else if me_alive && s.phase == Phase::Dead {
                s.phase = Phase::Playing;
                s.death_info = None;
                phase_signal.set(Phase::Playing);
            }
        }
    }
}
