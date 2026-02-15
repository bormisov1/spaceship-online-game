use std::cell::RefCell;
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
        effects::update_shake(&mut s, dt);
        let mut particles = std::mem::take(&mut s.particles);
        let mut explosions = std::mem::take(&mut s.explosions);
        let mut damage_numbers = std::mem::take(&mut s.damage_numbers);
        let mut hit_markers = std::mem::take(&mut s.hit_markers);
        drop(s);
        effects::update_particles(&mut particles, &mut explosions, dt);
        effects::update_damage_numbers(&mut damage_numbers, dt);
        effects::update_hit_markers(&mut hit_markers, dt);
        let mut s = state.borrow_mut();
        s.particles = particles;
        s.explosions = explosions;
        s.damage_numbers = damage_numbers;
        s.hit_markers = hit_markers;
        // Clean up expired mob speech
        let now = js_sys::Date::now();
        s.mob_speech.retain(|sp| now - sp.time < 3000.0);
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

    // Zoom transform (with screen shake offset)
    let vw = screen_w / cam_zoom;
    let vh = screen_h / cam_zoom;
    let (shake_x, shake_y) = {
        let s = state.borrow();
        (s.shake_x, s.shake_y)
    };
    let offset_x = cam_x - vw / 2.0 + shake_x;
    let offset_y = cam_y - vh / 2.0 + shake_y;

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

    // Players (with interpolation — render inline to avoid per-frame Vec/String allocations)
    {
        let s = state.borrow();
        let my_id = s.my_id.as_deref();
        let my_boosting = s.boosting;

        for (id, p) in &s.players {
            if !p.a { continue; }
            let (px, py, pr) = if let Some(prev) = s.prev_players.get(id) {
                (prev.x + (p.x - prev.x) * interp_t,
                 prev.y + (p.y - prev.y) * interp_t,
                 lerp_angle(prev.r, p.r, interp_t))
            } else {
                (p.x, p.y, p.r)
            };

            let sx = px - offset_x;
            let sy = py - offset_y;
            if sx < -60.0 || sx > vw + 60.0 || sy < -60.0 || sy > vh + 60.0 { continue; }

            let is_me = my_id == Some(id.as_str());
            let pvx = p.vx.unwrap_or(0.0);
            let pvy = p.vy.unwrap_or(0.0);
            let speed = (pvx * pvx + pvy * pvy).sqrt();
            let boosting = is_me && my_boosting;

            effects::draw_engine_beam(&ctx, sx, sy, pr, speed, p.s, boosting);
            ships::draw_ship(&ctx, sx, sy, pr, p.s);
            hud::draw_player_health_bar(&ctx, sx, sy, p.hp, p.mhp, &p.n, is_me);
        }
    }

    // Mobs (with interpolation — render inline to avoid per-frame HashMap allocation)
    {
        let s = state.borrow();
        for (id, mob) in &s.mobs {
            if !mob.a { continue; }
            let (mx, my, mr) = if let Some(prev) = s.prev_mobs.get(id) {
                (prev.x + (mob.x - prev.x) * interp_t,
                 prev.y + (mob.y - prev.y) * interp_t,
                 lerp_angle(prev.r, mob.r, interp_t))
            } else {
                (mob.x, mob.y, mob.r)
            };
            mobs::render_mob(&ctx, mx, my, mr, mob.vx.unwrap_or(0.0), mob.vy.unwrap_or(0.0), mob.hp, mob.mhp, offset_x, offset_y, vw, vh);
        }
    }

    // Particles & Explosions
    {
        let s = state.borrow();
        effects::render_particles(&ctx, &s.particles, offset_x, offset_y, vw, vh);
        effects::render_explosions(&ctx, &s.explosions, offset_x, offset_y, vw, vh);
    }

    // Mob speech bubbles (world-space, inside zoom)
    {
        let s = state.borrow();
        effects::render_mob_speech(&ctx, &s.mob_speech, &s.mobs, offset_x, offset_y, vw, vh);
    }

    // Damage numbers (world-space, inside zoom)
    {
        let s = state.borrow();
        effects::render_damage_numbers(&ctx, &s.damage_numbers, offset_x, offset_y, vw, vh);
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

    // Hit markers (screen-space, no zoom)
    {
        let s = state.borrow();
        effects::render_hit_markers(&ctx, &s.hit_markers, screen_w, screen_h);
    }

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
