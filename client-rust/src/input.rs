use wasm_bindgen::prelude::*;
use wasm_bindgen::JsCast;
use web_sys::{MouseEvent, KeyboardEvent, TouchEvent};
use crate::state::{SharedState, Phase, TouchJoystick};
use crate::network::SharedNetwork;

const BOOST_COLUMN_HALF: f64 = 50.0;

pub fn setup_input(state: SharedState, _net: SharedNetwork) {
    let window = web_sys::window().unwrap();
    let document = window.document().unwrap();

    // Detect mobile
    {
        let nav = window.navigator();
        let is_mobile = nav.max_touch_points() > 0;
        state.borrow_mut().is_mobile = is_mobile;
        if is_mobile {
            let mut s = state.borrow_mut();
            s.mouse_x = s.screen_w / 2.0;
            s.mouse_y = s.screen_h / 2.0;
        }
    }

    let canvas = match document.get_element_by_id("gameCanvas") {
        Some(c) => c,
        None => return,
    };

    // Mouse move
    let state_mm = state.clone();
    let is_mobile = state.borrow().is_mobile;
    let mousemove = Closure::wrap(Box::new(move |e: MouseEvent| {
        if is_mobile { return; }
        let mut s = state_mm.borrow_mut();
        s.mouse_x = e.client_x() as f64;
        s.mouse_y = e.client_y() as f64;
    }) as Box<dyn FnMut(MouseEvent)>);
    let _ = canvas.add_event_listener_with_callback("mousemove", mousemove.as_ref().unchecked_ref());
    mousemove.forget();

    // Mouse down
    let state_md = state.clone();
    let mousedown = Closure::wrap(Box::new(move |e: MouseEvent| {
        if is_mobile { return; }
        let s = state_md.borrow();
        if s.phase != Phase::Playing { return; }
        drop(s);
        if e.button() == 0 {
            state_md.borrow_mut().firing = true;
        }
    }) as Box<dyn FnMut(MouseEvent)>);
    let _ = canvas.add_event_listener_with_callback("mousedown", mousedown.as_ref().unchecked_ref());
    mousedown.forget();

    // Mouse up
    let state_mu = state.clone();
    let mouseup = Closure::wrap(Box::new(move |e: MouseEvent| {
        if is_mobile { return; }
        if e.button() == 0 {
            state_mu.borrow_mut().firing = false;
        }
    }) as Box<dyn FnMut(MouseEvent)>);
    let _ = canvas.add_event_listener_with_callback("mouseup", mouseup.as_ref().unchecked_ref());
    mouseup.forget();

    // Context menu
    let contextmenu = Closure::wrap(Box::new(move |e: web_sys::Event| {
        e.prevent_default();
    }) as Box<dyn FnMut(web_sys::Event)>);
    let _ = canvas.add_event_listener_with_callback("contextmenu", contextmenu.as_ref().unchecked_ref());
    contextmenu.forget();

    // Key down
    let state_kd = state.clone();
    let keydown = Closure::wrap(Box::new(move |e: KeyboardEvent| {
        let s = state_kd.borrow();
        if s.phase != Phase::Playing { return; }
        drop(s);
        match e.key().as_str() {
            "w" | "W" => state_kd.borrow_mut().firing = true,
            "Shift" => {
                let mut s = state_kd.borrow_mut();
                s.boosting = true;
                s.shift_pressed = true;
                // Lock rotation at moment shift is pressed
                if s.hyperspace_locked_r.is_none() {
                    let locked_r = s.my_id.as_ref()
                        .and_then(|id| s.players.get(id))
                        .map(|p| p.r);
                    s.hyperspace_locked_r = locked_r;
                }
            }
            "q" | "Q" | " " => {
                state_kd.borrow_mut().ability_pressed = true;
            }
            "d" | "D" => {
                let mut s = state_kd.borrow_mut();
                s.debug_hitboxes = !s.debug_hitboxes;
            }
            _ => {}
        }
    }) as Box<dyn FnMut(KeyboardEvent)>);
    let _ = document.add_event_listener_with_callback("keydown", keydown.as_ref().unchecked_ref());
    keydown.forget();

    // Key up
    let state_ku = state.clone();
    let keyup = Closure::wrap(Box::new(move |e: KeyboardEvent| {
        match e.key().as_str() {
            "w" | "W" => state_ku.borrow_mut().firing = false,
            "Shift" => {
                let mut s = state_ku.borrow_mut();
                s.boosting = false;
                s.shift_pressed = false;
                s.hyperspace_locked_r = None;
            }
            "q" | "Q" | " " => {
                state_ku.borrow_mut().ability_pressed = false;
            }
            _ => {}
        }
    }) as Box<dyn FnMut(KeyboardEvent)>);
    let _ = document.add_event_listener_with_callback("keyup", keyup.as_ref().unchecked_ref());
    keyup.forget();

    // Touch input (mobile)
    if is_mobile {
        setup_touch_input(state.clone(), &canvas);

        // Prevent document-level scroll
        let prevent = Closure::wrap(Box::new(move |e: web_sys::Event| {
            e.prevent_default();
        }) as Box<dyn FnMut(web_sys::Event)>);
        let opts = web_sys::AddEventListenerOptions::new();
        let _ = document.add_event_listener_with_callback_and_add_event_listener_options(
            "touchmove", prevent.as_ref().unchecked_ref(), &opts,
        );
        prevent.forget();
    }
}

fn setup_touch_input(state: SharedState, canvas: &web_sys::Element) {
    const JOYSTICK_SCALE: f64 = 2.5;

    // Touch start
    let state_ts = state.clone();
    let touchstart = Closure::wrap(Box::new(move |e: TouchEvent| {
        e.prevent_default();
        let s = state_ts.borrow();
        if s.phase != Phase::Playing { return; }
        let screen_w = s.screen_w;
        drop(s);

        let half_w = screen_w / 2.0;
        let center_left = half_w - BOOST_COLUMN_HALF;
        let center_right = half_w + BOOST_COLUMN_HALF;

        let changed = e.changed_touches();
        for i in 0..changed.length() {
            if let Some(touch) = changed.get(i) {
                let cx = touch.client_x() as f64;
                let cy = touch.client_y() as f64;

                // Center column = boost (invisible)
                if cx >= center_left && cx <= center_right {
                    let mut s = state_ts.borrow_mut();
                    s.boosting = true;
                    s.shift_pressed = true;
                    if s.hyperspace_locked_r.is_none() {
                        let locked_r = s.my_id.as_ref()
                            .and_then(|id| s.players.get(id))
                            .map(|p| p.r);
                        s.hyperspace_locked_r = locked_r;
                    }
                    continue;
                }

                let mut s = state_ts.borrow_mut();
                if cx < center_left && s.touch_joystick.is_none() {
                    s.touch_joystick = Some(TouchJoystick {
                        start_x: cx,
                        start_y: cy,
                        current_x: cx,
                        current_y: cy,
                    });
                    s.mouse_x = s.screen_w / 2.0;
                    s.mouse_y = s.screen_h / 2.0;
                } else if cx > center_right && !s.firing {
                    s.firing = true;
                }
            }
        }
    }) as Box<dyn FnMut(TouchEvent)>);

    let opts = web_sys::AddEventListenerOptions::new();
    let _ = canvas.add_event_listener_with_callback_and_add_event_listener_options(
        "touchstart", touchstart.as_ref().unchecked_ref(), &opts,
    );
    touchstart.forget();

    // Touch move
    let state_tm = state.clone();
    let touchmove = Closure::wrap(Box::new(move |e: TouchEvent| {
        e.prevent_default();
        let changed = e.changed_touches();
        for i in 0..changed.length() {
            if let Some(touch) = changed.get(i) {
                let mut s = state_tm.borrow_mut();
                if let Some(ref mut tj) = s.touch_joystick {
                    let cx = touch.client_x() as f64;
                    let cy = touch.client_y() as f64;
                    // Check if this touch is near the joystick start
                    tj.current_x = cx;
                    tj.current_y = cy;
                    let dx = cx - tj.start_x;
                    let dy = cy - tj.start_y;
                    s.mouse_x = s.screen_w / 2.0 + dx * JOYSTICK_SCALE;
                    s.mouse_y = s.screen_h / 2.0 + dy * JOYSTICK_SCALE;
                }
            }
        }
    }) as Box<dyn FnMut(TouchEvent)>);
    let _ = canvas.add_event_listener_with_callback_and_add_event_listener_options(
        "touchmove", touchmove.as_ref().unchecked_ref(), &opts,
    );
    touchmove.forget();

    // Touch end / cancel
    let state_te = state.clone();
    let touchend = Closure::wrap(Box::new(move |e: TouchEvent| {
        e.prevent_default();
        let changed = e.changed_touches();
        for i in 0..changed.length() {
            if let Some(_touch) = changed.get(i) {
                let cx = _touch.client_x() as f64;

                let mut s = state_te.borrow_mut();
                let half_w = s.screen_w / 2.0;
                let center_left = half_w - BOOST_COLUMN_HALF;
                let center_right = half_w + BOOST_COLUMN_HALF;

                // Center column = release boost
                if cx >= center_left && cx <= center_right {
                    s.boosting = false;
                    s.shift_pressed = false;
                    s.hyperspace_locked_r = None;
                    continue;
                }

                // Left zone: release joystick
                if cx < center_left {
                    if s.touch_joystick.is_some() {
                        if let Some(ref tj) = s.touch_joystick {
                            if (cx - tj.start_x).abs() < 200.0 {
                                s.touch_joystick = None;
                                s.mouse_x = s.screen_w / 2.0;
                                s.mouse_y = s.screen_h / 2.0;
                            }
                        }
                    }
                }

                // Right zone: release fire
                if cx > center_right {
                    s.firing = false;
                }
            }
        }
    }) as Box<dyn FnMut(TouchEvent)>);
    let _ = canvas.add_event_listener_with_callback_and_add_event_listener_options(
        "touchend", touchend.as_ref().unchecked_ref(), &opts,
    );
    let touchend2 = Closure::wrap(Box::new(move |e: TouchEvent| {
        e.prevent_default();
    }) as Box<dyn FnMut(TouchEvent)>);
    let _ = canvas.add_event_listener_with_callback_and_add_event_listener_options(
        "touchcancel", touchend2.as_ref().unchecked_ref(), &opts,
    );
    touchend.forget();
    touchend2.forget();
}
