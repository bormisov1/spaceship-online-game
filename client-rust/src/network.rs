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
    expired_signal: leptos::prelude::RwSignal<bool>,
    auth_signal: leptos::prelude::RwSignal<Option<String>>,
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
        expired_signal: leptos::prelude::RwSignal<bool>,
        auth_signal: leptos::prelude::RwSignal<Option<String>>,
    ) -> SharedNetwork {
        let net = Rc::new(RefCell::new(Network {
            ws: None,
            state,
            phase_signal,
            sessions_signal,
            checked_signal,
            expired_signal,
            auth_signal,
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
        ws.set_binary_type(web_sys::BinaryType::Arraybuffer);

        // on open
        let state_clone = net.borrow().state.clone();
        let net_clone = net.clone();
        let on_open = Closure::wrap(Box::new(move || {
            state_clone.borrow_mut().connected = true;
            web_sys::console::log_1(&"WebSocket connected".into());

            // Auto-authenticate with stored token (don't restore username yet — wait for auth_ok)
            if let Ok(Some(storage)) = web_sys::window().unwrap().local_storage() {
                if let Ok(Some(token)) = storage.get_item("auth_token") {
                    if !token.is_empty() {
                        Network::send_auth_token(&net_clone, &token);
                    }
                }
            }

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
        let expired_signal = net.borrow().expired_signal;
        let auth_signal = net.borrow().auth_signal;
        let net_for_msg = net.clone();
        let on_message = Closure::wrap(Box::new(move |e: MessageEvent| {
            let data = e.data();
            // Binary message = msgpack-encoded GameState
            if let Some(ab) = data.dyn_ref::<js_sys::ArrayBuffer>() {
                let arr = js_sys::Uint8Array::new(ab);
                let bytes = arr.to_vec();
                if let Ok(gs) = rmp_serde::from_slice::<GameStateMsg>(&bytes) {
                    handle_state(&state_clone, &phase_signal, gs);
                }
            } else if let Some(text) = data.as_string() {
                if let Ok(env) = serde_json::from_str::<Envelope>(&text) {
                    handle_message(&state_clone, &net_for_msg, phase_signal, sessions_signal, checked_signal, expired_signal, auth_signal, env);
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

    pub fn send_binary(net: &SharedNetwork, data: &[u8]) {
        let net_ref = net.borrow();
        if let Some(ws) = &net_ref.ws {
            if ws.ready_state() == 1 {
                let _ = ws.send_with_u8_array(data);
            }
        }
    }

    pub fn send_input(net: &SharedNetwork) {
        let state = net.borrow().state.clone();
        let s = state.borrow();
        let dominated_by_playing = matches!(s.phase, Phase::Playing | Phase::Dead | Phase::Countdown);
        if !dominated_by_playing || s.my_id.is_none() {
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

        let s2 = state.borrow();
        let fire = s2.firing;
        let boost = s2.boosting;
        let ability = s2.ability_pressed;
        drop(s2);

        // Binary input: 8 bytes [0x01, mx_hi, mx_lo, my_hi, my_lo, flags, thresh_hi, thresh_lo]
        let mx_i = mx.round() as i16;
        let my_i = my.round() as i16;
        let thresh_i = thresh.round().max(0.0).min(65535.0) as u16;
        let flags: u8 = (if fire { 0x01 } else { 0 }) | (if boost { 0x02 } else { 0 }) | (if ability { 0x04 } else { 0 });
        let buf: [u8; 8] = [
            0x01,
            (mx_i as u16 >> 8) as u8, mx_i as u8,
            (my_i as u16 >> 8) as u8, my_i as u8,
            flags,
            (thresh_i >> 8) as u8, thresh_i as u8,
        ];
        Network::send_binary(net, &buf);
    }

    pub fn list_sessions(net: &SharedNetwork) {
        Network::send_raw(net, "list", &serde_json::json!({}));
    }

    pub fn create_session(net: &SharedNetwork, name: &str, session_name: &str, mode: i32) {
        Network::send_raw(net, "create", &serde_json::json!({"name": name, "sname": session_name, "mode": mode}));
    }

    pub fn join_session(net: &SharedNetwork, name: &str, session_id: &str) {
        Network::send_raw(net, "join", &serde_json::json!({"name": name, "sid": session_id}));
    }

    pub fn send_leave(net: &SharedNetwork) {
        Network::send_raw(net, "leave", &serde_json::json!({}));
    }

    pub fn send_ready(net: &SharedNetwork) {
        Network::send_raw(net, "ready", &serde_json::json!({}));
    }

    pub fn send_team_pick(net: &SharedNetwork, team: i32) {
        Network::send_raw(net, "team_pick", &serde_json::json!({"team": team}));
    }

    pub fn send_rematch(net: &SharedNetwork) {
        Network::send_raw(net, "rematch", &serde_json::json!({}));
    }

    pub fn send_register(net: &SharedNetwork, username: &str, password: &str) {
        Network::send_raw(net, "register", &serde_json::json!({"username": username, "password": password}));
    }

    pub fn send_login(net: &SharedNetwork, username: &str, password: &str) {
        Network::send_raw(net, "login", &serde_json::json!({"username": username, "password": password}));
    }

    pub fn send_auth_token(net: &SharedNetwork, token: &str) {
        Network::send_raw(net, "auth", &serde_json::json!({"token": token}));
    }

    pub fn send_profile_request(net: &SharedNetwork) {
        Network::send_raw(net, "profile", &serde_json::json!({}));
    }

    pub fn send_leaderboard_request(net: &SharedNetwork) {
        Network::send_raw(net, "leaderboard", &serde_json::json!({}));
    }

    pub fn send_friend_add(net: &SharedNetwork, username: &str) {
        Network::send_raw(net, "friend_add", &serde_json::json!({"username": username}));
    }

    pub fn send_friend_accept(net: &SharedNetwork, username: &str) {
        Network::send_raw(net, "friend_accept", &serde_json::json!({"username": username}));
    }

    pub fn send_friend_decline(net: &SharedNetwork, username: &str) {
        Network::send_raw(net, "friend_decline", &serde_json::json!({"username": username}));
    }

    pub fn send_friend_list(net: &SharedNetwork) {
        Network::send_raw(net, "friend_list", &serde_json::json!({}));
    }

    pub fn send_chat(net: &SharedNetwork, text: &str, team: bool) {
        Network::send_raw(net, "chat", &serde_json::json!({"text": text, "team": team}));
    }

    pub fn send_store_request(net: &SharedNetwork) {
        Network::send_raw(net, "store", &serde_json::json!({}));
    }

    pub fn send_buy(net: &SharedNetwork, item_id: &str) {
        Network::send_raw(net, "buy", &serde_json::json!({"item_id": item_id}));
    }

    pub fn send_equip(net: &SharedNetwork, skin_id: &str, trail_id: &str) {
        Network::send_raw(net, "equip", &serde_json::json!({"skin_id": skin_id, "trail_id": trail_id}));
    }

    pub fn send_daily_login(net: &SharedNetwork) {
        Network::send_raw(net, "daily_login", &serde_json::json!({}));
    }
}

fn handle_message(
    state: &SharedState,
    net: &SharedNetwork,
    phase_signal: leptos::prelude::RwSignal<Phase>,
    sessions_signal: leptos::prelude::RwSignal<Vec<SessionInfo>>,
    checked_signal: leptos::prelude::RwSignal<Option<CheckedMsg>>,
    expired_signal: leptos::prelude::RwSignal<bool>,
    auth_signal: leptos::prelude::RwSignal<Option<String>>,
    env: Envelope,
) {
    let data = env.d.unwrap_or(serde_json::Value::Null);
    match env.t.as_str() {
        "state" => {
            // Legacy JSON state handling (binary msgpack is preferred path)
            if let Ok(gs) = serde_json::from_value::<GameStateMsg>(data) {
                handle_state(state, &phase_signal, gs);
            }
        }
        "welcome" => {
            if let Ok(w) = serde_json::from_value::<WelcomeMsg>(data) {
                let mut s = state.borrow_mut();
                s.my_id = Some(w.id);
                s.my_ship = w.s;
                // Default to Playing; server will send match_phase to override if needed
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
                    Some(&format!("{}{}", crate::app::base_path(), j.sid)),
                );
            }
        }
        "created" => {
            if let Ok(c) = serde_json::from_value::<CreatedMsg>(data) {
                // Update URL without full reload, then auto-join
                let window = web_sys::window().unwrap();
                let _ = window.history().unwrap().push_state_with_url(
                    &wasm_bindgen::JsValue::NULL, "", Some(&format!("{}{}", crate::app::base_path(), c.sid)),
                );
                let name = state.borrow_mut().pending_name.take()
                    .unwrap_or_else(|| "Pilot".to_string());
                state.borrow_mut().url_session_id = Some(c.sid.clone());
                Network::join_session(net, &name, &c.sid);
            }
        }
        "sessions" => {
            if let Ok(sessions) = serde_json::from_value::<Vec<SessionInfo>>(data) {
                sessions_signal.set(sessions);
            }
        }
        "hit" => {
            if let Ok(h) = serde_json::from_value::<HitMsg>(data) {
                let mut s = state.borrow_mut();
                let my_id = s.my_id.clone();

                // Damage number at hit position
                effects::add_damage_number(&mut s, h.x, h.y, h.dmg, false);

                // Screen shake — bigger for victim
                let shake_amount = (h.dmg as f64 / 10.0).min(6.0);
                if my_id.as_deref() == Some(&h.vid) {
                    effects::trigger_shake(&mut s, shake_amount * 1.5);
                } else {
                    effects::trigger_shake(&mut s, shake_amount * 0.5);
                }

                // Hit marker if I'm the attacker
                if my_id.as_deref() == Some(&h.aid) {
                    effects::add_hit_marker(&mut s);
                }
            }
        }
        "mob_say" => {
            if let Ok(ms) = serde_json::from_value::<MobSayMsg>(data) {
                let mut s = state.borrow_mut();
                effects::add_mob_speech(&mut s, ms.mid, ms.text);
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

                // Screen shake on kills
                let my_id = s.my_id.clone();
                if my_id.as_deref() == Some(k.kid.as_str()) {
                    effects::trigger_shake(&mut s, 8.0); // I got a kill
                } else if my_id.as_deref() == Some(k.vid.as_str()) {
                    effects::trigger_shake(&mut s, 12.0); // I died
                } else {
                    effects::trigger_shake(&mut s, 3.0); // nearby kill
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
                if !c.exists {
                    // Session expired — clear URL session and redirect to lobby
                    state.borrow_mut().url_session_id = None;
                    let window = web_sys::window().unwrap();
                    let _ = window.history().unwrap().replace_state_with_url(
                        &wasm_bindgen::JsValue::NULL, "", Some(crate::app::base_path()),
                    );
                    expired_signal.set(true);
                } else {
                    checked_signal.set(Some(c));
                }
            }
        }
        "match_phase" => {
            if let Ok(mp) = serde_json::from_value::<MatchPhaseMsg>(data) {
                let mut s = state.borrow_mut();
                s.game_mode = crate::state::GameMode::from_i32(mp.mode);
                s.match_phase = mp.phase;
                s.countdown_time = mp.countdown;
                match mp.phase {
                    0 => {
                        // PhaseLobby
                        s.is_ready = false;
                        s.match_result = None;
                        if matches!(s.game_mode, crate::state::GameMode::FFA) {
                            // FFA: return to main lobby (session selection)
                            s.session_id = None;
                            s.my_id = None;
                            s.controller_attached = false;
                            s.phase = Phase::Lobby;
                            phase_signal.set(Phase::Lobby);
                            // Reset URL to base path
                            let window = web_sys::window().unwrap();
                            let _ = window.history().unwrap().push_state_with_url(
                                &wasm_bindgen::JsValue::NULL, "", Some(crate::app::base_path()),
                            );
                            drop(s);
                            Network::send_leave(net);
                        } else {
                            // Team modes: stay in match lobby for rematching
                            s.phase = Phase::MatchLobby;
                            phase_signal.set(Phase::MatchLobby);
                        }
                    }
                    1 => {
                        // PhaseCountdown
                        s.phase = Phase::Countdown;
                        phase_signal.set(Phase::Countdown);
                    }
                    2 => {
                        // PhasePlaying
                        s.phase = Phase::Playing;
                        phase_signal.set(Phase::Playing);
                    }
                    3 => {
                        // PhaseResult
                        s.phase = Phase::Result;
                        phase_signal.set(Phase::Result);
                    }
                    _ => {}
                }
            }
        }
        "match_result" => {
            if let Ok(mr) = serde_json::from_value::<MatchResultMsg>(data) {
                let mut s = state.borrow_mut();
                s.match_result = Some((mr.winner_team, mr.players, mr.duration));
                s.phase = Phase::Result;
                phase_signal.set(Phase::Result);
            }
        }
        "team_update" => {
            if let Ok(tu) = serde_json::from_value::<TeamUpdateMsg>(data) {
                let mut s = state.borrow_mut();
                s.team_red = tu.red;
                s.team_blue = tu.blue;
            }
        }
        "ctrl_on" => {
            state.borrow_mut().controller_attached = true;
            // Dismiss QR overlay on desktop
            if let Some(overlay) = web_sys::window().unwrap().document().unwrap()
                .get_element_by_id("controllerOverlay") {
                let _ = overlay.class_list().remove_1("visible");
            }
        }
        "ctrl_off" => {
            state.borrow_mut().controller_attached = false;
        }
        "auth_ok" => {
            if let Ok(a) = serde_json::from_value::<AuthOKMsg>(data) {
                let mut s = state.borrow_mut();
                s.auth_token = Some(a.token.clone());
                s.auth_username = Some(a.username.clone());
                s.auth_player_id = a.pid;
                // Store token in localStorage
                if let Ok(Some(storage)) = web_sys::window().unwrap().local_storage() {
                    let _ = storage.set_item("auth_token", &a.token);
                    let _ = storage.set_item("auth_username", &a.username);
                }
                drop(s);
                // Update auth signal for reactive UI
                auth_signal.set(Some(a.username.clone()));
                // Request profile data and claim daily login
                Network::send_profile_request(net);
                Network::send_daily_login(net);
            }
        }
        "profile_data" => {
            if let Ok(p) = serde_json::from_value::<ProfileDataMsg>(data) {
                let mut s = state.borrow_mut();
                s.auth_level = p.level;
                s.auth_xp = p.xp;
                s.auth_xp_next = p.xp_next;
                s.auth_kills = p.kills;
                s.auth_deaths = p.deaths;
                s.auth_wins = p.wins;
                s.auth_losses = p.losses;
                s.auth_credits = p.credits;
            }
        }
        "xp_update" => {
            if let Ok(xu) = serde_json::from_value::<XPUpdateMsg>(data) {
                let mut s = state.borrow_mut();
                s.auth_xp = xu.total_xp;
                s.auth_level = xu.level;
                s.auth_xp_next = xu.xp_next;
                s.xp_notification = Some(crate::state::XPNotification {
                    xp_gained: xu.xp_gained,
                    level: xu.level,
                    prev_level: xu.prev_level,
                    leveled_up: xu.leveled_up,
                });
                s.xp_notification_time = web_sys::window().unwrap().performance().unwrap().now();
            }
        }
        "leaderboard_res" => {
            if let Ok(lb) = serde_json::from_value::<LeaderboardMsg>(data) {
                state.borrow_mut().leaderboard = lb.entries;
            }
        }
        "achievement" => {
            if let Ok(ach) = serde_json::from_value::<AchievementMsg>(data) {
                let mut s = state.borrow_mut();
                s.achievement_queue.push(crate::state::AchievementNotification {
                    name: ach.name,
                    description: ach.desc,
                });
                if s.achievement_show_time == 0.0 {
                    s.achievement_show_time = web_sys::window().unwrap().performance().unwrap().now();
                }
            }
        }
        "friend_list_res" => {
            if let Ok(fl) = serde_json::from_value::<FriendListMsg>(data) {
                let mut s = state.borrow_mut();
                s.friends = fl.friends;
                s.friend_requests = fl.requests;
            }
        }
        "friend_notify" => {
            if let Ok(n) = serde_json::from_value::<FriendNotifyMsg>(data) {
                web_sys::console::log_1(&format!("Friend {}: {}", n.notify_type, n.username).into());
                // Refresh friend list
                Network::send_friend_list(net);
            }
        }
        "store_res" => {
            if let Ok(sr) = serde_json::from_value::<crate::protocol::StoreResMsg>(data) {
                let mut s = state.borrow_mut();
                s.store_items = sr.items;
                s.owned_skins = sr.owned;
                s.auth_credits = sr.credits;
                s.equipped_skin = sr.skin;
                s.equipped_trail = sr.trail;
            }
        }
        "buy_res" => {
            if let Ok(br) = serde_json::from_value::<crate::protocol::BuyResMsg>(data) {
                let mut s = state.borrow_mut();
                if br.success {
                    s.owned_skins.push(br.item_id);
                    s.auth_credits = br.credits;
                }
            }
        }
        "inventory_res" => {
            if let Ok(ir) = serde_json::from_value::<crate::protocol::InventoryResMsg>(data) {
                let mut s = state.borrow_mut();
                s.owned_skins = ir.owned;
                s.equipped_skin = ir.skin;
                s.equipped_trail = ir.trail;
                s.auth_credits = ir.credits;
            }
        }
        "credits_update" => {
            if let Ok(cu) = serde_json::from_value::<crate::protocol::CreditsUpdateMsg>(data) {
                state.borrow_mut().auth_credits = cu.credits;
            }
        }
        "daily_login_res" => {
            // Handled by credits_update that follows
        }
        "chat_msg" => {
            if let Ok(msg) = serde_json::from_value::<ChatMsg>(data) {
                let mut s = state.borrow_mut();
                s.chat_messages.push(crate::state::ChatMessage {
                    from: msg.from,
                    text: msg.text,
                    team: msg.team,
                    time: web_sys::window().unwrap().performance().unwrap().now(),
                });
                // Keep max 50 messages
                if s.chat_messages.len() > 50 {
                    s.chat_messages.remove(0);
                }
            }
        }
        "error" => {
            if let Ok(e) = serde_json::from_value::<ErrorMsg>(data) {
                web_sys::console::error_1(&format!("Server error: {}", e.msg).into());
                // Invalid token — clear stale auth from state and storage
                if e.msg == "invalid token" {
                    let mut s = state.borrow_mut();
                    s.auth_token = None;
                    s.auth_username = None;
                    s.auth_player_id = 0;
                    drop(s);
                    auth_signal.set(None);
                    if let Ok(Some(storage)) = web_sys::window().unwrap().local_storage() {
                        let _ = storage.remove_item("auth_token");
                        let _ = storage.remove_item("auth_username");
                    }
                }
                // Show error in auth error element if it exists
                if let Some(el) = web_sys::window().unwrap().document().unwrap()
                    .get_element_by_id("authError") {
                    el.set_text_content(Some(&e.msg));
                }
            }
        }
        _ => {}
    }
}

fn handle_state(state: &SharedState, phase_signal: &leptos::prelude::RwSignal<Phase>, gs: GameStateMsg) {
    let mut s = state.borrow_mut();

    // Save current→prev for interpolation (swap reuses allocations)
    let mut tmp_players = std::mem::take(&mut s.prev_players);
    tmp_players.clear();
    std::mem::swap(&mut tmp_players, &mut s.players);
    s.prev_players = tmp_players;

    let mut tmp_mobs = std::mem::take(&mut s.prev_mobs);
    tmp_mobs.clear();
    std::mem::swap(&mut tmp_mobs, &mut s.mobs);
    s.prev_mobs = tmp_mobs;

    s.prev_cam_x = s.cam_x;
    s.prev_cam_y = s.cam_y;

    // Record timing for interpolation
    let now = web_sys::window().unwrap().performance().unwrap().now();
    if s.interp_last_update > 0.0 {
        let elapsed = now - s.interp_last_update;
        // Smooth the interval estimate
        if elapsed > 10.0 && elapsed < 200.0 {
            s.interp_interval = s.interp_interval * 0.8 + elapsed * 0.2;
        }
    }
    s.interp_last_update = now;

    // Update current state, merging delta-compressed velocity
    s.players.clear();
    for mut p in gs.p {
        // If velocity was omitted (delta compression), carry forward from prev
        if p.vx.is_none() || p.vy.is_none() {
            if let Some(prev) = s.prev_players.get(&p.id) {
                if p.vx.is_none() { p.vx = prev.vx; }
                if p.vy.is_none() { p.vy = prev.vy; }
            }
        }
        s.players.insert(p.id.clone(), p);
    }

    s.projectiles.clear();
    for pr in gs.pr {
        s.projectiles.insert(pr.id.clone(), pr);
    }

    s.mobs.clear();
    for mut m in gs.m {
        if m.vx.is_none() || m.vy.is_none() {
            if let Some(prev) = s.prev_mobs.get(&m.id) {
                if m.vx.is_none() { m.vx = prev.vx; }
                if m.vy.is_none() { m.vy = prev.vy; }
            }
        }
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

    s.heal_zones = gs.hz;
    s.tick = gs.tick;
    s.match_phase = gs.mp;
    s.match_time_left = gs.tl;
    s.team_red_score = gs.trs;
    s.team_blue_score = gs.tbs;

    // Update camera + sync controller boost state
    if let Some(my_id) = &s.my_id {
        if let Some(me) = s.players.get(my_id) {
            let me_x = me.x;
            let me_y = me.y;
            let me_alive = me.a;
            let me_boosting = me.b;
            s.cam_x = me_x;
            s.cam_y = me_y;

            // When controller is attached, sync boost visual from server state
            if s.controller_attached {
                s.boosting = me_boosting;
                s.shift_pressed = me_boosting;
            }

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
