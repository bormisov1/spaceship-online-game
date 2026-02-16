use wasm_bindgen::prelude::*;
use wasm_bindgen::JsCast;
use web_sys::HtmlCanvasElement;
use crate::state::SharedState;

pub fn resize(state: &SharedState) {
    let window = web_sys::window().unwrap();
    let w = window.inner_width().unwrap().as_f64().unwrap();
    let mut h = window.inner_height().unwrap().as_f64().unwrap();

    // On desktop, subtract donation banner height (28px) so content doesn't overlap it
    let document = window.document().unwrap();
    if let Some(banner) = document.query_selector(".donation-banner").ok().flatten() {
        if let Some(el) = banner.dyn_ref::<web_sys::HtmlElement>() {
            let display = window.get_computed_style(el)
                .ok().flatten()
                .and_then(|s| s.get_property_value("display").ok())
                .unwrap_or_default();
            if display != "none" {
                h -= 28.0;
            }
        }
    }

    if let Some(canvas) = document.get_element_by_id("gameCanvas") {
        let canvas: HtmlCanvasElement = canvas.unchecked_into();
        canvas.set_width(w as u32);
        canvas.set_height(h as u32);
    }
    if let Some(canvas) = document.get_element_by_id("bgCanvas") {
        let canvas: HtmlCanvasElement = canvas.unchecked_into();
        canvas.set_width(w as u32);
        canvas.set_height(h as u32);
    }

    let mut s = state.borrow_mut();
    s.screen_w = w;
    s.screen_h = h;

    let min_dim = w.min(h);
    s.cam_zoom = (min_dim / 700.0).min(1.0);

    if s.is_mobile && s.touch_joystick.is_none() {
        s.mouse_x = w / 2.0;
        s.mouse_y = h / 2.0;
    }
}

pub fn setup_resize_handler(state: SharedState) {
    let closure = Closure::wrap(Box::new(move |_: web_sys::Event| {
        resize(&state);
    }) as Box<dyn FnMut(web_sys::Event)>);

    let window = web_sys::window().unwrap();
    let _ = window.add_event_listener_with_callback("resize", closure.as_ref().unchecked_ref());
    closure.forget();
}

pub fn setup_fullscreen() {
    let document = web_sys::window().unwrap().document().unwrap();
    let btn = match document.get_element_by_id("fullscreenBtn") {
        Some(b) => b,
        None => return,
    };

    let closure = Closure::wrap(Box::new(move |_: web_sys::Event| {
        let document = web_sys::window().unwrap().document().unwrap();
        let elem = document.document_element().unwrap();

        // Check if fullscreen
        let is_fs = js_sys::Reflect::get(&document, &"fullscreenElement".into())
            .ok()
            .map(|v| !v.is_null() && !v.is_undefined())
            .unwrap_or(false);

        if !is_fs {
            let _ = js_sys::Reflect::get(&elem, &"requestFullscreen".into())
                .ok()
                .and_then(|f| f.dyn_into::<js_sys::Function>().ok())
                .map(|f| { let _ = f.call0(&elem); });
        } else {
            let _ = js_sys::Reflect::get(&document, &"exitFullscreen".into())
                .ok()
                .and_then(|f| f.dyn_into::<js_sys::Function>().ok())
                .map(|f| { let _ = f.call0(&document); });
        }
    }) as Box<dyn FnMut(web_sys::Event)>);

    let _ = btn.add_event_listener_with_callback("click", closure.as_ref().unchecked_ref());
    closure.forget();
}

pub fn setup_controller_btn(state: SharedState) {
    let document = web_sys::window().unwrap().document().unwrap();

    let btn = match document.get_element_by_id("controllerBtn") {
        Some(b) => b,
        None => return,
    };

    // Show on desktop only
    let is_mobile = {
        let window = web_sys::window().unwrap();
        let nav = window.navigator();
        nav.max_touch_points() > 0
    };

    if !is_mobile {
        let _ = btn.dyn_ref::<web_sys::HtmlElement>().map(|e| {
            let _ = e.style().set_property("display", "flex");
        });
    }

    let state_clone = state.clone();
    let btn_click = Closure::wrap(Box::new(move |_: web_sys::Event| {
        let s = state_clone.borrow();
        let (my_id, session_id) = (s.my_id.clone(), s.session_id.clone());
        drop(s);

        if let (Some(my_id), Some(session_id)) = (my_id, session_id) {
            let window = web_sys::window().unwrap();
            let origin = window.location().origin().unwrap_or_default();
            let controller_url = format!("{}/{}?c={}", origin, session_id, my_id);

            let document = window.document().unwrap();
            if let Some(qr_img) = document.get_element_by_id("qrImg") {
                let _ = qr_img.set_attribute("src", &format!("/api/qr?data={}", js_sys::encode_uri_component(&controller_url)));
            }
            if let Some(qr_url) = document.get_element_by_id("qrUrl") {
                qr_url.set_text_content(Some(&controller_url));
            }
            if let Some(overlay) = document.get_element_by_id("controllerOverlay") {
                let _ = overlay.class_list().add_1("visible");
            }
        }
    }) as Box<dyn FnMut(web_sys::Event)>);
    let _ = btn.add_event_listener_with_callback("click", btn_click.as_ref().unchecked_ref());
    btn_click.forget();

    // Close button
    if let Some(close_btn) = document.get_element_by_id("qrClose") {
        let close_click = Closure::wrap(Box::new(move |_: web_sys::Event| {
            let document = web_sys::window().unwrap().document().unwrap();
            if let Some(overlay) = document.get_element_by_id("controllerOverlay") {
                let _ = overlay.class_list().remove_1("visible");
            }
        }) as Box<dyn FnMut(web_sys::Event)>);
        let _ = close_btn.add_event_listener_with_callback("click", close_click.as_ref().unchecked_ref());
        close_click.forget();
    }

    // Escape to close
    let esc_closure = Closure::wrap(Box::new(move |e: web_sys::KeyboardEvent| {
        if e.key() == "Escape" {
            let document = web_sys::window().unwrap().document().unwrap();
            if let Some(overlay) = document.get_element_by_id("controllerOverlay") {
                let _ = overlay.class_list().remove_1("visible");
            }
        }
    }) as Box<dyn FnMut(web_sys::KeyboardEvent)>);
    let _ = document.add_event_listener_with_callback("keydown", esc_closure.as_ref().unchecked_ref());
    esc_closure.forget();
}

pub fn get_canvas_context(id: &str) -> Option<web_sys::CanvasRenderingContext2d> {
    let document = web_sys::window()?.document()?;
    let canvas = document.get_element_by_id(id)?;
    let canvas: HtmlCanvasElement = canvas.unchecked_into();
    canvas
        .get_context("2d")
        .ok()?
        .map(|c| c.unchecked_into())
}
