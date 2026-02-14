use std::cell::RefCell;
use std::collections::HashMap;
use wasm_bindgen::JsCast;
use web_sys::CanvasRenderingContext2d;
use crate::state::SharedState;
use crate::constants::*;
use crate::{starfield, ships, effects, projectiles, mobs, asteroids, pickups, fog, hud, auto_aim};

fn lerp_angle(from: f64, to: f64, t: f64) -> f64 {
    let mut diff = to - from;
    // Normalize to [-PI, PI]
    while diff > std::f64::consts::PI { diff -= 2.0 * std::f64::consts::PI; }
    while diff < -std::f64::consts::PI { diff += 2.0 * std::f64::consts::PI; }
    from + diff * t
}

thread_local! {
    static SHIPS_LOADED: RefCell<bool> = RefCell::new(false);
    static ASTEROIDS_LOADED: RefCell<bool> = RefCell::new(false);
}

fn ensure_loaded() {
    SHIPS_LOADED.with(|sl| {
        if !*sl.borrow() {
            ships::load_ship_images();
            *sl.borrow_mut() = true;
        }
    });
    ASTEROIDS_LOADED.with(|al| {
        if !*al.borrow() {
            asteroids::load_asteroid_image();
            *al.borrow_mut() = true;
        }
    });
}

pub fn render(state: &SharedState, dt: f64) {
    ensure_loaded();

    let window = web_sys::window().unwrap();
    let document = window.document().unwrap();
    let now = window.performance().unwrap().now();

    let bg_canvas = match document.get_element_by_id("bgCanvas") {
        Some(c) => c.unchecked_into::<web_sys::HtmlCanvasElement>(),
        None => return,
    };
    let game_canvas = match document.get_element_by_id("gameCanvas") {
        Some(c) => c.unchecked_into::<web_sys::HtmlCanvasElement>(),
        None => return,
    };

    let bg_ctx: CanvasRenderingContext2d = bg_canvas
        .get_context("2d").unwrap().unwrap().unchecked_into();
    let ctx: CanvasRenderingContext2d = game_canvas
        .get_context("2d").unwrap().unwrap().unchecked_into();

    // Compute interpolation factor
    let (screen_w, screen_h, cam_x, cam_y, cam_zoom, interp_t);
    {
        let s = state.borrow();
        screen_w = s.screen_w;
        screen_h = s.screen_h;
        cam_zoom = s.cam_zoom;

        // Interpolate camera between prev and current
        let elapsed = now - s.interp_last_update;
        let t = if s.interp_interval > 0.0 { (elapsed / s.interp_interval).min(1.0).max(0.0) } else { 1.0 };
        interp_t = t;
        cam_x = s.prev_cam_x + (s.cam_x - s.prev_cam_x) * t;
        cam_y = s.prev_cam_y + (s.cam_y - s.prev_cam_y) * t;
    }

    // Update effects
    {
        let mut s = state.borrow_mut();
        let mut particles = std::mem::take(&mut s.particles);
        let mut explosions = std::mem::take(&mut s.explosions);
        drop(s);
        effects::update_particles(&mut particles, &mut explosions, dt);
        let mut s = state.borrow_mut();
        s.particles = particles;
        s.explosions = explosions;
    }

    // Animate hyperspace_t
    let hyperspace_t;
    {
        let mut s = state.borrow_mut();
        let target = if s.shift_pressed { 1.0 } else { 0.0 };
        let speed = 3.0; // transition speed
        if s.hyperspace_t < target {
            s.hyperspace_t = (s.hyperspace_t + speed * dt).min(target);
        } else {
            s.hyperspace_t = (s.hyperspace_t - speed * dt).max(target);
        }
        hyperspace_t = s.hyperspace_t;
    }

    // Starfield on bg canvas
    let player_rotation = {
        let s = state.borrow();
        s.my_id.as_ref()
            .and_then(|id| s.players.get(id))
            .map(|p| p.r)
            .unwrap_or(0.0)
    };
    starfield::render_starfield(&bg_ctx, cam_x, cam_y, screen_w, screen_h, hyperspace_t, player_rotation);

    // Clear game canvas
    ctx.clear_rect(0.0, 0.0, screen_w, screen_h);

    // Zoom transform
    let vw = screen_w / cam_zoom;
    let vh = screen_h / cam_zoom;
    let offset_x = cam_x - vw / 2.0;
    let offset_y = cam_y - vh / 2.0;

    ctx.save();
    ctx.scale(cam_zoom, cam_zoom).unwrap_or(());

    // Fog
    fog::render_fog(&ctx, offset_x, offset_y, vw, vh);

    // World bounds
    draw_world_bounds(&ctx, offset_x, offset_y);

    // Pickups
    {
        let s = state.borrow();
        let time_secs = now / 1000.0;
        pickups::render_pickups(&ctx, &s.pickups, offset_x, offset_y, vw, vh, time_secs);
    }

    // Asteroids
    {
        let s = state.borrow();
        asteroids::render_asteroids(&ctx, &s.asteroids, offset_x, offset_y, vw, vh);
    }

    // Projectiles
    {
        let s = state.borrow();
        projectiles::render_projectiles(&ctx, &s.projectiles, &s.players, offset_x, offset_y, vw, vh);
    }

    // Players (with interpolation)
    {
        let (my_id, my_boosting) = {
            let s = state.borrow();
            (s.my_id.clone(), s.boosting)
        };

        // Collect player data with interpolated positions
        let player_data: Vec<(String, f64, f64, f64, f64, f64, i32, i32, i32, String)> = {
            let s = state.borrow();
            s.players.iter()
                .filter(|(_, p)| p.a)
                .map(|(id, p)| {
                    // Interpolate from prev position if available
                    if let Some(prev) = s.prev_players.get(id) {
                        let ix = prev.x + (p.x - prev.x) * interp_t;
                        let iy = prev.y + (p.y - prev.y) * interp_t;
                        let ir = lerp_angle(prev.r, p.r, interp_t);
                        (id.clone(), ix, iy, ir, p.vx, p.vy, p.s, p.hp, p.mhp, p.n.clone())
                    } else {
                        // New entity — render at current position
                        (id.clone(), p.x, p.y, p.r, p.vx, p.vy, p.s, p.hp, p.mhp, p.n.clone())
                    }
                })
                .collect()
        };

        for (ref id, px, py, pr, pvx, pvy, ps, php, pmhp, ref pn) in &player_data {
            let sx = px - offset_x;
            let sy = py - offset_y;
            if sx < -60.0 || sx > vw + 60.0 || sy < -60.0 || sy > vh + 60.0 { continue; }

            let is_me = my_id.as_deref() == Some(id.as_str());
            let speed = (pvx * pvx + pvy * pvy).sqrt();
            let boosting = is_me && my_boosting;

            effects::draw_engine_beam(&ctx, sx, sy, *pr, speed, *ps, boosting);
            ships::draw_ship(&ctx, sx, sy, *pr, *ps);
            hud::draw_player_health_bar(&ctx, sx, sy, *php, *pmhp, pn, is_me);
        }
    }

    // Mobs (with interpolation)
    {
        let s = state.borrow();
        let interp_mobs: HashMap<String, crate::protocol::MobState> = s.mobs.iter().map(|(id, m)| {
            if let Some(prev) = s.prev_mobs.get(id) {
                let mut im = m.clone();
                im.x = prev.x + (m.x - prev.x) * interp_t;
                im.y = prev.y + (m.y - prev.y) * interp_t;
                im.r = lerp_angle(prev.r, m.r, interp_t);
                (id.clone(), im)
            } else {
                (id.clone(), m.clone())
            }
        }).collect();
        mobs::render_mobs(&ctx, &interp_mobs, offset_x, offset_y, vw, vh);
    }

    // Particles & Explosions
    {
        let s = state.borrow();
        effects::render_particles(&ctx, &s.particles, offset_x, offset_y, vw, vh);
        effects::render_explosions(&ctx, &s.explosions, offset_x, offset_y, vw, vh);
    }

    // Auto-aim reticle (when controller attached or mobile)
    {
        let s = state.borrow();
        if s.controller_attached || s.is_mobile {
            drop(s);
            auto_aim::update_and_draw_controller_aim(&ctx, state, offset_x, offset_y, dt);
        }
    }

    // Debug hitboxes
    {
        let s = state.borrow();
        if s.debug_hitboxes {
            draw_debug_hitboxes(&ctx, &s, offset_x, offset_y, vw, vh);
        }
    }

    ctx.restore();

    // HUD (screen-space, no zoom)
    hud::render_hud(&ctx, state);
}

fn draw_world_bounds(ctx: &CanvasRenderingContext2d, offset_x: f64, offset_y: f64) {
    ctx.set_stroke_style(&wasm_bindgen::JsValue::from_str("rgba(255, 100, 100, 0.3)"));
    ctx.set_line_width(2.0);
    ctx.set_line_dash(&js_sys::Array::of2(&10.0.into(), &10.0.into())).unwrap_or(());
    ctx.stroke_rect(-offset_x, -offset_y, WORLD_W, WORLD_H);
    ctx.set_line_dash(&js_sys::Array::new()).unwrap_or(());
}

fn draw_debug_hitboxes(ctx: &CanvasRenderingContext2d, s: &crate::state::GameState, offset_x: f64, offset_y: f64, vw: f64, vh: f64) {
    // Player hitboxes
    for p in s.players.values() {
        if !p.a { continue; }
        let sx = p.x - offset_x;
        let sy = p.y - offset_y;
        if sx < -50.0 || sx > vw + 50.0 || sy < -50.0 || sy > vh + 50.0 { continue; }

        ctx.begin_path();
        let _ = ctx.arc(sx, sy, PLAYER_RADIUS, 0.0, std::f64::consts::PI * 2.0);
        ctx.set_fill_style(&wasm_bindgen::JsValue::from_str("rgba(255, 255, 0, 0.15)"));
        ctx.fill();
        ctx.set_stroke_style(&wasm_bindgen::JsValue::from_str("rgba(255, 255, 0, 0.6)"));
        ctx.set_line_width(1.0);
        ctx.stroke();
    }

    // Projectile hitboxes
    for proj in s.projectiles.values() {
        let sx = proj.x - offset_x;
        let sy = proj.y - offset_y;
        if sx < -50.0 || sx > vw + 50.0 || sy < -50.0 || sy > vh + 50.0 { continue; }

        ctx.begin_path();
        let _ = ctx.arc(sx, sy, PROJECTILE_RADIUS, 0.0, std::f64::consts::PI * 2.0);
        ctx.set_fill_style(&wasm_bindgen::JsValue::from_str("rgba(255, 0, 0, 0.2)"));
        ctx.fill();
        ctx.set_stroke_style(&wasm_bindgen::JsValue::from_str("rgba(255, 0, 0, 0.7)"));
        ctx.set_line_width(1.0);
        ctx.stroke();
    }

    // Mob hitboxes
    for mob in s.mobs.values() {
        if !mob.a { continue; }
        let sx = mob.x - offset_x;
        let sy = mob.y - offset_y;
        if sx < -100.0 || sx > vw + 100.0 || sy < -100.0 || sy > vh + 100.0 { continue; }

        ctx.begin_path();
        let _ = ctx.arc(sx, sy, MOB_RADIUS, 0.0, std::f64::consts::PI * 2.0);
        ctx.set_fill_style(&wasm_bindgen::JsValue::from_str("rgba(255, 165, 0, 0.15)"));
        ctx.fill();
        ctx.set_stroke_style(&wasm_bindgen::JsValue::from_str("rgba(255, 165, 0, 0.6)"));
        ctx.set_line_width(1.0);
        ctx.stroke();
    }

    // Asteroid hitboxes
    for ast in s.asteroids.values() {
        let sx = ast.x - offset_x;
        let sy = ast.y - offset_y;
        if sx < -150.0 || sx > vw + 150.0 || sy < -150.0 || sy > vh + 150.0 { continue; }

        ctx.begin_path();
        let _ = ctx.arc(sx, sy, ASTEROID_RADIUS, 0.0, std::f64::consts::PI * 2.0);
        ctx.set_fill_style(&wasm_bindgen::JsValue::from_str("rgba(139, 90, 43, 0.15)"));
        ctx.fill();
        ctx.set_stroke_style(&wasm_bindgen::JsValue::from_str("rgba(139, 90, 43, 0.6)"));
        ctx.set_line_width(1.0);
        ctx.stroke();
    }

    // Pickup hitboxes
    for pk in s.pickups.values() {
        let sx = pk.x - offset_x;
        let sy = pk.y - offset_y;
        if sx < -50.0 || sx > vw + 50.0 || sy < -50.0 || sy > vh + 50.0 { continue; }

        ctx.begin_path();
        let _ = ctx.arc(sx, sy, PICKUP_RADIUS, 0.0, std::f64::consts::PI * 2.0);
        ctx.set_fill_style(&wasm_bindgen::JsValue::from_str("rgba(0, 255, 0, 0.15)"));
        ctx.fill();
        ctx.set_stroke_style(&wasm_bindgen::JsValue::from_str("rgba(0, 255, 0, 0.6)"));
        ctx.set_line_width(1.0);
        ctx.stroke();
    }
}
