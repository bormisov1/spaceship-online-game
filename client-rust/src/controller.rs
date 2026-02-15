use wasm_bindgen::prelude::*;
use wasm_bindgen::JsCast;
use web_sys::{WebSocket, MessageEvent, CloseEvent, ErrorEvent, TouchEvent};
use std::cell::RefCell;
use std::rc::Rc;
use crate::constants::{INPUT_RATE, RECONNECT_DELAY};

const JOYSTICK_SCALE: f64 = 3.0;
const DEAD_ZONE: f64 = 8.0;
const AIM_ORBIT_R: f64 = 360.0;
const AIM_DETECT_R: f64 = 150.0;

const BOOST_COLUMN_HALF: f64 = 50.0;
const DEBUG_MAX_LINES: usize = 30;

fn debug_log(msg: &str) {
    let document = web_sys::window().unwrap().document().unwrap();
    if let Some(el) = document.get_element_by_id("ctrlDebug") {
        let prev = el.text_content().unwrap_or_default();
        let mut lines: Vec<&str> = prev.lines().collect();
        lines.push(msg);
        if lines.len() > DEBUG_MAX_LINES {
            lines.drain(0..lines.len() - DEBUG_MAX_LINES);
        }
        el.set_text_content(Some(&lines.join("\n")));
        // Auto-scroll to bottom
        let html_el: &web_sys::HtmlElement = el.unchecked_ref();
        html_el.set_scroll_top(html_el.scroll_height());
    }
}

struct ControllerState {
    ws: Option<WebSocket>,
    sid: String,
    pid: String,
    connected: bool,
    attached: bool,
    player_x: f64,
    player_y: f64,
    player_r: f64,
    screen_w: f64,
    screen_h: f64,
    enemies: Vec<Enemy>,
    lock_target_id: Option<String>,
    joystick_dx: f64,
    joystick_dy: f64,
    joystick_touch_id: Option<i32>,
    joystick_start_x: f64,
    joystick_start_y: f64,
    fire_touch_id: Option<i32>,
    firing: bool,
    boost_touch_id: Option<i32>,
    boosting: bool,
    boost_locked_r: Option<f64>,
    // Store closures
    _on_open: Option<Closure<dyn FnMut()>>,
    _on_message: Option<Closure<dyn FnMut(MessageEvent)>>,
    _on_close: Option<Closure<dyn FnMut(CloseEvent)>>,
    _on_error: Option<Closure<dyn FnMut(ErrorEvent)>>,
}

struct Enemy {
    id: String,
    x: f64,
    y: f64,
}

type SharedCtrl = Rc<RefCell<ControllerState>>;

pub fn init_controller(session_id: &str, player_id: &str) {
    // Debug log will work after DOM is ready, log to web_sys console for init
    web_sys::console::log_1(&format!("init_controller sid={} pid={}", session_id, player_id).into());

    let ctrl = Rc::new(RefCell::new(ControllerState {
        ws: None,
        sid: session_id.to_string(),
        pid: player_id.to_string(),
        connected: false,
        attached: false,
        player_x: 0.0,
        player_y: 0.0,
        player_r: 0.0,
        screen_w: 0.0,
        screen_h: 0.0,
        enemies: Vec::new(),
        lock_target_id: None,
        joystick_dx: 0.0,
        joystick_dy: 0.0,
        joystick_touch_id: None,
        joystick_start_x: 0.0,
        joystick_start_y: 0.0,
        fire_touch_id: None,
        firing: false,
        boost_touch_id: None,
        boosting: false,
        boost_locked_r: None,
        _on_open: None,
        _on_message: None,
        _on_close: None,
        _on_error: None,
    }));

    check_orientation(&ctrl);

    // Orientation change handler
    let ctrl_orient = ctrl.clone();
    let orient_closure = Closure::wrap(Box::new(move |_: web_sys::Event| {
        check_orientation(&ctrl_orient);
    }) as Box<dyn FnMut(web_sys::Event)>);
    let window = web_sys::window().unwrap();
    let _ = window.add_event_listener_with_callback("resize", orient_closure.as_ref().unchecked_ref());
    orient_closure.forget();

    // Touch handlers
    setup_touch_handlers(&ctrl);

    // Connect
    connect_ws(&ctrl);
}

fn check_orientation(ctrl: &SharedCtrl) {
    let window = web_sys::window().unwrap();
    let w = window.inner_width().unwrap().as_f64().unwrap();
    let h = window.inner_height().unwrap().as_f64().unwrap();

    ctrl.borrow_mut().screen_w = w;
    ctrl.borrow_mut().screen_h = h;

    let landscape = w > h;
    let document = window.document().unwrap();

    if let Some(rotate_msg) = document.get_element_by_id("ctrlRotateMsg") {
        let el: &web_sys::HtmlElement = rotate_msg.unchecked_ref();
        let _ = el.style().set_property("display", if landscape { "none" } else { "flex" });
    }
    if let Some(pad) = document.get_element_by_id("ctrlPad") {
        let el: &web_sys::HtmlElement = pad.unchecked_ref();
        let _ = el.style().set_property("display", if landscape { "block" } else { "none" });
    }
}

fn connect_ws(ctrl: &SharedCtrl) {
    let window = web_sys::window().unwrap();
    let location = window.location();
    let protocol = location.protocol().unwrap_or_default();
    let host = location.host().unwrap_or_default();
    let ws_proto = if protocol == "https:" { "wss:" } else { "ws:" };
    let url = format!("{}//{}/ws", ws_proto, host);

    debug_log(&format!("WS connecting to {}", url));

    let ws = WebSocket::new(&url).unwrap();
    ws.set_binary_type(web_sys::BinaryType::Arraybuffer);

    // on open
    let ctrl_open = ctrl.clone();
    let on_open = Closure::wrap(Box::new(move || {
        let mut c = ctrl_open.borrow_mut();
        c.connected = true;
        update_status("Attaching...");
        let sid = c.sid.clone();
        let pid = c.pid.clone();
        debug_log(&format!("WS open, sending control sid={} pid={}", sid, pid));
        if let Some(ref ws) = c.ws {
            let msg = serde_json::json!({"t": "control", "d": {"sid": sid, "pid": pid}});
            let _ = ws.send_with_str(&msg.to_string());
        }
    }) as Box<dyn FnMut()>);

    // on message — handle both binary (msgpack state) and text (JSON control messages)
    let ctrl_msg = ctrl.clone();
    let msg_count: Rc<RefCell<u32>> = Rc::new(RefCell::new(0));
    let on_message = Closure::wrap(Box::new(move |e: MessageEvent| {
        let data = e.data();
        if let Some(ab) = data.dyn_ref::<js_sys::ArrayBuffer>() {
            let arr = js_sys::Uint8Array::new(ab);
            let bytes = arr.to_vec();
            let mut cnt = msg_count.borrow_mut();
            *cnt += 1;
            if *cnt <= 3 {
                debug_log(&format!("bin msg #{} len={}", *cnt, bytes.len()));
            }
            match rmp_serde::from_slice::<crate::protocol::GameStateMsg>(&bytes) {
                Ok(gs) => {
                    if *cnt <= 3 {
                        let c = ctrl_msg.borrow();
                        debug_log(&format!("  state: {} players, pid match={}", gs.p.len(),
                            gs.p.iter().any(|p| p.id == c.pid)));
                    }
                    handle_state(&ctrl_msg, gs);
                }
                Err(err) => {
                    debug_log(&format!("  msgpack ERR: {}", err));
                }
            }
        } else if let Some(text) = data.as_string() {
            debug_log(&format!("txt msg: {}", &text[..text.len().min(120)]));
            if let Ok(env) = serde_json::from_str::<crate::protocol::Envelope>(&text) {
                handle_message(&ctrl_msg, env);
            }
        } else {
            debug_log("msg: unknown type (not arraybuffer, not string)");
        }
    }) as Box<dyn FnMut(MessageEvent)>);

    // on close
    let ctrl_close = ctrl.clone();
    let on_close = Closure::wrap(Box::new(move |e: CloseEvent| {
        debug_log(&format!("WS closed code={} reason={}", e.code(), e.reason()));
        {
            let mut c = ctrl_close.borrow_mut();
            c.connected = false;
            c.attached = false;
        }
        update_status("Disconnected. Reconnecting...");
        let ctrl_reconnect = ctrl_close.clone();
        gloo_timers::callback::Timeout::new(RECONNECT_DELAY, move || {
            connect_ws(&ctrl_reconnect);
        }).forget();
    }) as Box<dyn FnMut(CloseEvent)>);

    let on_error = Closure::wrap(Box::new(move |_: ErrorEvent| {
        debug_log("WS error event");
    }) as Box<dyn FnMut(ErrorEvent)>);

    ws.set_onopen(Some(on_open.as_ref().unchecked_ref()));
    ws.set_onmessage(Some(on_message.as_ref().unchecked_ref()));
    ws.set_onclose(Some(on_close.as_ref().unchecked_ref()));
    ws.set_onerror(Some(on_error.as_ref().unchecked_ref()));

    let mut c = ctrl.borrow_mut();
    c.ws = Some(ws);
    c._on_open = Some(on_open);
    c._on_message = Some(on_message);
    c._on_close = Some(on_close);
    c._on_error = Some(on_error);
}

fn handle_state(ctrl: &SharedCtrl, gs: crate::protocol::GameStateMsg) {
    let mut c = ctrl.borrow_mut();
    let pid = c.pid.clone();
    let mut new_enemies = Vec::new();

    for p in &gs.p {
        if p.id == pid {
            c.player_x = p.x;
            c.player_y = p.y;
            c.player_r = p.r;
        } else if p.a {
            new_enemies.push(Enemy { id: format!("p_{}", p.id), x: p.x, y: p.y });
        }
    }
    for m in &gs.m {
        if m.a {
            new_enemies.push(Enemy { id: format!("m_{}", m.id), x: m.x, y: m.y });
        }
    }
    c.enemies = new_enemies;
}

fn handle_message(ctrl: &SharedCtrl, env: crate::protocol::Envelope) {
    debug_log(&format!("handle_message t={}", env.t));
    let data = env.d.unwrap_or(serde_json::Value::Null);
    match env.t.as_str() {
        "control_ok" => {
            debug_log("ATTACHED - starting input loop");
            ctrl.borrow_mut().attached = true;
            update_status("Connected");
            start_input_loop(ctrl);
        }
        "error" => {
            if let Ok(e) = serde_json::from_value::<crate::protocol::ErrorMsg>(data) {
                debug_log(&format!("server error: {}", e.msg));
                update_status(&format!("Error: {}", e.msg));
            }
        }
        _ => {
            debug_log(&format!("unhandled msg type: {}", env.t));
        }
    }
}

fn update_status(text: &str) {
    let document = web_sys::window().unwrap().document().unwrap();
    if let Some(el) = document.get_element_by_id("ctrlStatus") {
        el.set_text_content(Some(text));
    }
}

fn setup_touch_handlers(ctrl: &SharedCtrl) {
    // Wait a bit for DOM to be ready
    let ctrl_clone = ctrl.clone();
    gloo_timers::callback::Timeout::new(100, move || {
        let document = web_sys::window().unwrap().document().unwrap();
        let has_pad = document.get_element_by_id("ctrlPad").is_some();
        debug_log(&format!("setup_touch: ctrlPad found={}", has_pad));
        if let Some(pad) = document.get_element_by_id("ctrlPad") {
            let opts = web_sys::AddEventListenerOptions::new();
            opts.set_passive(false);

            // Touch start
            let ctrl_ts = ctrl_clone.clone();
            let ts = Closure::wrap(Box::new(move |e: TouchEvent| {
                e.prevent_default();
                let c = ctrl_ts.borrow();
                let half_w = c.screen_w / 2.0;
                let center_left = half_w - BOOST_COLUMN_HALF;
                let center_right = half_w + BOOST_COLUMN_HALF;
                let has_joystick = c.joystick_touch_id.is_some();
                let has_fire = c.fire_touch_id.is_some();
                let has_boost = c.boost_touch_id.is_some();
                let player_r = c.player_r;
                drop(c);

                let changed = e.changed_touches();
                for i in 0..changed.length() {
                    if let Some(touch) = changed.get(i) {
                        let cx = touch.client_x() as f64;
                        let cy = touch.client_y() as f64;
                        let tid = touch.identifier();
                        let zone = if cx < center_left { "LEFT" } else if cx > center_right { "RIGHT" } else { "CENTER" };
                        debug_log(&format!("tstart id={} x={:.0} zone={} cl={:.0} cr={:.0}", tid, cx, zone, center_left, center_right));
                        let mut c = ctrl_ts.borrow_mut();
                        if cx < center_left && !has_joystick {
                            c.joystick_touch_id = Some(tid);
                            c.joystick_start_x = cx;
                            c.joystick_start_y = cy;
                            c.joystick_dx = 0.0;
                            c.joystick_dy = 0.0;
                        } else if cx > center_right && !has_fire {
                            c.fire_touch_id = Some(tid);
                            c.firing = true;
                            update_fire_indicator(true);
                        } else if cx >= center_left && cx <= center_right && !has_boost {
                            c.boost_touch_id = Some(tid);
                            c.boosting = true;
                            c.boost_locked_r = Some(player_r);
                            update_boost_indicator(true);
                        }
                    }
                }
            }) as Box<dyn FnMut(TouchEvent)>);
            let _ = pad.add_event_listener_with_callback_and_add_event_listener_options(
                "touchstart", ts.as_ref().unchecked_ref(), &opts,
            );
            ts.forget();

            // Touch move
            let ctrl_tm = ctrl_clone.clone();
            let tm = Closure::wrap(Box::new(move |e: TouchEvent| {
                e.prevent_default();
                let changed = e.changed_touches();
                for i in 0..changed.length() {
                    if let Some(touch) = changed.get(i) {
                        let tid = touch.identifier();
                        let c = ctrl_tm.borrow();
                        if c.joystick_touch_id == Some(tid) {
                            let dx = touch.client_x() as f64 - c.joystick_start_x;
                            let dy = touch.client_y() as f64 - c.joystick_start_y;
                            drop(c);
                            let mut c = ctrl_tm.borrow_mut();
                            c.joystick_dx = dx;
                            c.joystick_dy = dy;
                            update_knob(dx, dy);
                        }
                    }
                }
            }) as Box<dyn FnMut(TouchEvent)>);
            let _ = pad.add_event_listener_with_callback_and_add_event_listener_options(
                "touchmove", tm.as_ref().unchecked_ref(), &opts,
            );
            tm.forget();

            // Touch end / cancel — shared handler
            let make_touch_end = |ctrl_ref: SharedCtrl| {
                Closure::wrap(Box::new(move |e: TouchEvent| {
                    e.prevent_default();
                    let changed = e.changed_touches();
                    for i in 0..changed.length() {
                        if let Some(touch) = changed.get(i) {
                            let tid = touch.identifier();
                            let mut c = ctrl_ref.borrow_mut();
                            if c.joystick_touch_id == Some(tid) {
                                c.joystick_touch_id = None;
                                c.joystick_dx = 0.0;
                                c.joystick_dy = 0.0;
                                update_knob(0.0, 0.0);
                            }
                            if c.fire_touch_id == Some(tid) {
                                c.fire_touch_id = None;
                                c.firing = false;
                                update_fire_indicator(false);
                            }
                            if c.boost_touch_id == Some(tid) {
                                c.boost_touch_id = None;
                                c.boosting = false;
                                c.boost_locked_r = None;
                                update_boost_indicator(false);
                            }
                        }
                    }
                }) as Box<dyn FnMut(TouchEvent)>)
            };

            let te = make_touch_end(ctrl_clone.clone());
            let _ = pad.add_event_listener_with_callback_and_add_event_listener_options(
                "touchend", te.as_ref().unchecked_ref(), &opts,
            );
            te.forget();

            let te2 = make_touch_end(ctrl_clone.clone());
            let _ = pad.add_event_listener_with_callback_and_add_event_listener_options(
                "touchcancel", te2.as_ref().unchecked_ref(), &opts,
            );
            te2.forget();
        }
    }).forget();
}

fn update_knob(dx: f64, dy: f64) {
    let document = web_sys::window().unwrap().document().unwrap();
    if let Some(knob) = document.get_element_by_id("joystickKnob") {
        let el: &web_sys::HtmlElement = knob.unchecked_ref();
        let max_r = 45.0;
        let dist = (dx * dx + dy * dy).sqrt();
        let (dx, dy) = if dist > max_r {
            (dx / dist * max_r, dy / dist * max_r)
        } else {
            (dx, dy)
        };
        let _ = el.style().set_property(
            "transform",
            &format!("translate(calc(-50% + {}px), calc(-50% + {}px))", dx, dy),
        );
    }
}

fn update_fire_indicator(active: bool) {
    let document = web_sys::window().unwrap().document().unwrap();
    if let Some(ind) = document.get_element_by_id("fireIndicator") {
        if active {
            let _ = ind.class_list().add_1("active");
        } else {
            let _ = ind.class_list().remove_1("active");
        }
    }
}

fn update_boost_indicator(active: bool) {
    let document = web_sys::window().unwrap().document().unwrap();
    if let Some(ind) = document.get_element_by_id("boostIndicator") {
        if active {
            let _ = ind.class_list().add_1("active");
        } else {
            let _ = ind.class_list().remove_1("active");
        }
    }
}

fn start_input_loop(ctrl: &SharedCtrl) {
    let ctrl_clone = ctrl.clone();
    let send_count: Rc<RefCell<u32>> = Rc::new(RefCell::new(0));
    let interval = gloo_timers::callback::Interval::new(1000 / INPUT_RATE, move || {
        let mut cnt = send_count.borrow_mut();
        *cnt += 1;
        let log_this = *cnt <= 3 || *cnt % 40 == 0; // log first 3, then every 2s
        drop(cnt);
        send_input(&ctrl_clone, log_this);
    });
    std::mem::forget(interval);
}

fn send_input(ctrl: &SharedCtrl, log: bool) {
    let c = ctrl.borrow();
    if !c.connected || !c.attached {
        if log { debug_log(&format!("send_input skip: conn={} att={}", c.connected, c.attached)); }
        return;
    }

    let dist = (c.joystick_dx * c.joystick_dx + c.joystick_dy * c.joystick_dy).sqrt();

    let (mx, my);
    let mut lock_id: Option<String>;

    if dist > DEAD_ZONE {
        let aim_angle = c.joystick_dy.atan2(c.joystick_dx);
        let orbit_x = c.player_x + aim_angle.cos() * AIM_ORBIT_R;
        let orbit_y = c.player_y + aim_angle.sin() * AIM_ORBIT_R;

        // Auto-aim: only when joystick is active
        let mut locked = false;
        lock_id = c.lock_target_id.clone();
        let mut target_x = 0.0;
        let mut target_y = 0.0;

        if let Some(ref tid) = lock_id {
            if let Some(t) = c.enemies.iter().find(|e| &e.id == tid) {
                let dx = t.x - orbit_x;
                let dy = t.y - orbit_y;
                if dx * dx + dy * dy <= AIM_DETECT_R * AIM_DETECT_R {
                    locked = true;
                    target_x = t.x;
                    target_y = t.y;
                }
            }
            if !locked { lock_id = None; }
        }

        if !locked {
            let mut best_dist = AIM_DETECT_R * AIM_DETECT_R;
            for e in &c.enemies {
                let dx = e.x - orbit_x;
                let dy = e.y - orbit_y;
                let d2 = dx * dx + dy * dy;
                if d2 <= best_dist {
                    best_dist = d2;
                    lock_id = Some(e.id.clone());
                    target_x = e.x;
                    target_y = e.y;
                    locked = true;
                }
            }
        }

        if locked {
            mx = target_x;
            my = target_y;
        } else {
            mx = c.player_x + c.joystick_dx * JOYSTICK_SCALE;
            my = c.player_y + c.joystick_dy * JOYSTICK_SCALE;
        }
    } else {
        // Joystick idle: maintain current heading, clear lock
        lock_id = None;
        mx = c.player_x;
        my = c.player_y;
    }

    // During boost, lock steering to the direction captured at boost start
    let (mx, my) = if c.boosting {
        if let Some(locked_r) = c.boost_locked_r {
            (c.player_x + locked_r.cos() * 1000.0, c.player_y + locked_r.sin() * 1000.0)
        } else {
            (mx, my)
        }
    } else {
        (mx, my)
    };

    let firing = c.firing;
    let boosting = c.boosting;
    let player_x = c.player_x;
    let player_y = c.player_y;
    let jdx = c.joystick_dx;
    let jdy = c.joystick_dy;
    let ws = c.ws.clone();
    drop(c);

    if log {
        debug_log(&format!("input: pos=({:.0},{:.0}) joy=({:.0},{:.0}) mx/my=({:.0},{:.0}) fire={} boost={}",
            player_x, player_y, jdx, jdy, mx, my, firing, boosting));
    }

    // Update lock target
    ctrl.borrow_mut().lock_target_id = lock_id;

    if let Some(ws) = ws {
        if ws.ready_state() == 1 {
            let msg = serde_json::json!({
                "t": "input",
                "d": { "mx": mx, "my": my, "fire": firing, "boost": boosting, "thresh": 50 }
            });
            let _ = ws.send_with_str(&msg.to_string());
        }
    }
}
