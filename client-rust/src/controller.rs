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
    firing: bool,
    boosting: bool,
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
        firing: false,
        boosting: false,
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

    let ws = WebSocket::new(&url).unwrap();

    // on open
    let ctrl_open = ctrl.clone();
    let on_open = Closure::wrap(Box::new(move || {
        let mut c = ctrl_open.borrow_mut();
        c.connected = true;
        update_status("Attaching...");
        let sid = c.sid.clone();
        let pid = c.pid.clone();
        if let Some(ref ws) = c.ws {
            let msg = serde_json::json!({"t": "control", "d": {"sid": sid, "pid": pid}});
            let _ = ws.send_with_str(&msg.to_string());
        }
    }) as Box<dyn FnMut()>);

    // on message — handle both binary (msgpack state) and text (JSON control messages)
    let ctrl_msg = ctrl.clone();
    let on_message = Closure::wrap(Box::new(move |e: MessageEvent| {
        let data = e.data();
        if let Some(ab) = data.dyn_ref::<js_sys::ArrayBuffer>() {
            let arr = js_sys::Uint8Array::new(ab);
            let bytes = arr.to_vec();
            if let Ok(gs) = rmp_serde::from_slice::<crate::protocol::GameStateMsg>(&bytes) {
                handle_state(&ctrl_msg, gs);
            }
        } else if let Some(text) = data.as_string() {
            if let Ok(env) = serde_json::from_str::<crate::protocol::Envelope>(&text) {
                handle_message(&ctrl_msg, env);
            }
        }
    }) as Box<dyn FnMut(MessageEvent)>);

    // on close
    let ctrl_close = ctrl.clone();
    let on_close = Closure::wrap(Box::new(move |_: CloseEvent| {
        {
            let mut c = ctrl_close.borrow_mut();
            c.connected = false;
            c.attached = false;
        }
        update_status("Disconnected. Reconnecting...");
        let ctrl_reconnect = ctrl_close.clone();
        let _ = gloo_timers::callback::Timeout::new(RECONNECT_DELAY, move || {
            connect_ws(&ctrl_reconnect);
        });
    }) as Box<dyn FnMut(CloseEvent)>);

    let on_error = Closure::wrap(Box::new(move |_: ErrorEvent| {}) as Box<dyn FnMut(ErrorEvent)>);

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
    let data = env.d.unwrap_or(serde_json::Value::Null);
    match env.t.as_str() {
        "control_ok" => {
            ctrl.borrow_mut().attached = true;
            update_status("Connected");
            start_input_loop(ctrl);
        }
        "error" => {
            if let Ok(e) = serde_json::from_value::<crate::protocol::ErrorMsg>(data) {
                update_status(&format!("Error: {}", e.msg));
            }
        }
        _ => {}
    }
}

fn update_status(text: &str) {
    let document = web_sys::window().unwrap().document().unwrap();
    if let Some(el) = document.get_element_by_id("ctrlStatus") {
        el.set_text_content(Some(text));
    }
}

fn setup_touch_handlers(ctrl: &SharedCtrl) {
    let document = web_sys::window().unwrap().document().unwrap();
    // Wait a bit for DOM to be ready
    let ctrl_clone = ctrl.clone();
    let _ = gloo_timers::callback::Timeout::new(100, move || {
        let document = web_sys::window().unwrap().document().unwrap();
        if let Some(pad) = document.get_element_by_id("ctrlPad") {
            let mut opts = web_sys::AddEventListenerOptions::new();
            opts.set_passive(false);

            // Touch start
            let ctrl_ts = ctrl_clone.clone();
            let ts = Closure::wrap(Box::new(move |e: TouchEvent| {
                e.prevent_default();
                let c = ctrl_ts.borrow();
                let half_w = c.screen_w / 2.0;
                let center_left = half_w - BOOST_COLUMN_HALF;
                let center_right = half_w + BOOST_COLUMN_HALF;
                drop(c);

                let changed = e.changed_touches();
                for i in 0..changed.length() {
                    if let Some(touch) = changed.get(i) {
                        let cx = touch.client_x() as f64;
                        let mut c = ctrl_ts.borrow_mut();
                        if cx < center_left {
                            c.joystick_dx = 0.0;
                            c.joystick_dy = 0.0;
                        } else if cx > center_right {
                            c.firing = true;
                            update_fire_indicator(true);
                        } else {
                            c.boosting = true;
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
                        let cx = touch.client_x() as f64;
                        let c = ctrl_tm.borrow();
                        let half_w = c.screen_w / 2.0;
                        let center_left = half_w - BOOST_COLUMN_HALF;
                        drop(c);
                        if cx < center_left {
                            // Approximate: use center of left zone as start
                            let center_x = center_left / 2.0;
                            let center_y = ctrl_tm.borrow().screen_h / 2.0;
                            let mut c = ctrl_tm.borrow_mut();
                            c.joystick_dx = cx - center_x;
                            c.joystick_dy = touch.client_y() as f64 - center_y;
                            update_knob(c.joystick_dx, c.joystick_dy);
                        }
                    }
                }
            }) as Box<dyn FnMut(TouchEvent)>);
            let _ = pad.add_event_listener_with_callback_and_add_event_listener_options(
                "touchmove", tm.as_ref().unchecked_ref(), &opts,
            );
            tm.forget();

            // Touch end
            let ctrl_te = ctrl_clone.clone();
            let te = Closure::wrap(Box::new(move |e: TouchEvent| {
                e.prevent_default();
                let changed = e.changed_touches();
                for i in 0..changed.length() {
                    if let Some(touch) = changed.get(i) {
                        let cx = touch.client_x() as f64;
                        let c = ctrl_te.borrow();
                        let half_w = c.screen_w / 2.0;
                        let center_left = half_w - BOOST_COLUMN_HALF;
                        let center_right = half_w + BOOST_COLUMN_HALF;
                        drop(c);
                        if cx < center_left {
                            let mut c = ctrl_te.borrow_mut();
                            c.joystick_dx = 0.0;
                            c.joystick_dy = 0.0;
                            update_knob(0.0, 0.0);
                        } else if cx > center_right {
                            ctrl_te.borrow_mut().firing = false;
                            update_fire_indicator(false);
                        } else {
                            ctrl_te.borrow_mut().boosting = false;
                            update_boost_indicator(false);
                        }
                    }
                }
            }) as Box<dyn FnMut(TouchEvent)>);
            let _ = pad.add_event_listener_with_callback_and_add_event_listener_options(
                "touchend", te.as_ref().unchecked_ref(), &opts,
            );
            let te2 = Closure::wrap(Box::new(move |e: TouchEvent| {
                e.prevent_default();
            }) as Box<dyn FnMut(TouchEvent)>);
            let _ = pad.add_event_listener_with_callback_and_add_event_listener_options(
                "touchcancel", te2.as_ref().unchecked_ref(), &opts,
            );
            te.forget();
            te2.forget();
        }
    });
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
    let interval = gloo_timers::callback::Interval::new(1000 / INPUT_RATE, move || {
        send_input(&ctrl_clone);
    });
    std::mem::forget(interval);
}

fn send_input(ctrl: &SharedCtrl) {
    let c = ctrl.borrow();
    if !c.connected || !c.attached { return; }

    let dist = (c.joystick_dx * c.joystick_dx + c.joystick_dy * c.joystick_dy).sqrt();
    let aim_angle = if dist > DEAD_ZONE {
        c.joystick_dy.atan2(c.joystick_dx)
    } else {
        c.player_r
    };

    let orbit_x = c.player_x + aim_angle.cos() * AIM_ORBIT_R;
    let orbit_y = c.player_y + aim_angle.sin() * AIM_ORBIT_R;

    // Auto-aim
    let mut locked = false;
    let mut lock_id = c.lock_target_id.clone();
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

    let (mx, my) = if locked {
        (target_x, target_y)
    } else if dist > DEAD_ZONE {
        (c.player_x + c.joystick_dx * JOYSTICK_SCALE, c.player_y + c.joystick_dy * JOYSTICK_SCALE)
    } else {
        (c.player_x, c.player_y)
    };

    let firing = c.firing;
    let boosting = c.boosting;
    let ws = c.ws.clone();
    drop(c);

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
