use std::cell::RefCell;
use std::rc::Rc;
use wasm_bindgen::prelude::*;
use wasm_bindgen::JsCast;
use crate::state::{SharedState, Phase};
use crate::renderer;

pub fn start_game_loop(state: SharedState) {
    let f: Rc<RefCell<Option<Closure<dyn FnMut(f64)>>>> = Rc::new(RefCell::new(None));
    let g = f.clone();

    let last_time = Rc::new(RefCell::new(0.0_f64));

    *g.borrow_mut() = Some(Closure::wrap(Box::new(move |timestamp: f64| {
        let mut lt = last_time.borrow_mut();
        let dt = ((timestamp - *lt) / 1000.0).min(0.05);
        *lt = timestamp;
        drop(lt);

        {
            let s = state.borrow();
            match s.phase {
                Phase::Playing | Phase::Dead | Phase::Countdown | Phase::MatchLobby | Phase::Result => {
                    drop(s);
                    renderer::render(&state, dt);
                }
                Phase::Lobby => {
                    let w = s.screen_w;
                    let h = s.screen_h;
                    drop(s);
                    if let Some(ctx) = crate::canvas::get_canvas_context("bgCanvas") {
                        crate::hyperspace::render_hyperspace(&ctx, w, h, dt);
                    }
                }
            }
        }

        // Request next frame
        let window = web_sys::window().unwrap();
        let _ = window.request_animation_frame(
            f.borrow().as_ref().unwrap().as_ref().unchecked_ref()
        );
    }) as Box<dyn FnMut(f64)>));

    let window = web_sys::window().unwrap();
    let _ = window.request_animation_frame(
        g.borrow().as_ref().unwrap().as_ref().unchecked_ref()
    );
}
