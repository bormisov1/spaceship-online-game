use std::cell::RefCell;
use std::collections::HashMap;
use wasm_bindgen::JsCast;
use web_sys::{CanvasRenderingContext2d, HtmlCanvasElement};
use crate::state::{Particle, ParticleKind, Explosion};
use crate::constants::SHIP_COLORS;

const MAX_PARTICLES: usize = 200;

// Fast WASM-native xorshift64 RNG (avoids JS interop overhead of Math.random)
thread_local! {
    static RNG_STATE: RefCell<u64> = RefCell::new(0);
}

fn init_rng_if_needed() {
    RNG_STATE.with(|s| {
        let mut state = s.borrow_mut();
        if *state == 0 {
            // Seed from js_sys::Math::random once
            let seed = (js_sys::Math::random() * u64::MAX as f64) as u64;
            *state = if seed == 0 { 1 } else { seed };
        }
    });
}

fn fast_random() -> f64 {
    RNG_STATE.with(|s| {
        let mut state = s.borrow_mut();
        if *state == 0 { *state = 1; }
        *state ^= *state << 13;
        *state ^= *state >> 7;
        *state ^= *state << 17;
        (*state % 10000) as f64 / 10000.0
    })
}

thread_local! {
    static PARTICLE_GLOWS: RefCell<HashMap<String, HtmlCanvasElement>> = RefCell::new(HashMap::new());
    static FLASH_SPRITE: RefCell<Option<HtmlCanvasElement>> = RefCell::new(None);
    static FILL_SPRITE: RefCell<Option<HtmlCanvasElement>> = RefCell::new(None);
}

fn get_particle_glow(color: &str) -> HtmlCanvasElement {
    PARTICLE_GLOWS.with(|pg| {
        let mut sprites = pg.borrow_mut();
        if let Some(canvas) = sprites.get(color) {
            return canvas.clone();
        }

        let document = web_sys::window().unwrap().document().unwrap();
        let canvas: HtmlCanvasElement = document.create_element("canvas").unwrap().unchecked_into();
        let size = 32u32;
        canvas.set_width(size);
        canvas.set_height(size);
        let ctx: CanvasRenderingContext2d = canvas
            .get_context("2d").unwrap().unwrap().unchecked_into();

        let c = size as f64 / 2.0;
        if let Ok(gradient) = ctx.create_radial_gradient(c, c, 0.0, c, c, c) {
            let _ = gradient.add_color_stop(0.0_f32, color);
            let inner = format!("{}66", color);
            let _ = gradient.add_color_stop(0.4_f32, &inner);
            let _ = gradient.add_color_stop(1.0_f32, "transparent");
            ctx.set_fill_style(&gradient);
            ctx.fill_rect(0.0, 0.0, size as f64, size as f64);
        }

        sprites.insert(color.to_string(), canvas.clone());
        canvas
    })
}

fn get_flash_sprite() -> HtmlCanvasElement {
    FLASH_SPRITE.with(|fs| {
        let mut opt = fs.borrow_mut();
        if let Some(canvas) = opt.as_ref() {
            return canvas.clone();
        }

        let document = web_sys::window().unwrap().document().unwrap();
        let canvas: HtmlCanvasElement = document.create_element("canvas").unwrap().unchecked_into();
        let size = 64u32;
        canvas.set_width(size);
        canvas.set_height(size);
        let ctx: CanvasRenderingContext2d = canvas
            .get_context("2d").unwrap().unwrap().unchecked_into();

        let c = size as f64 / 2.0;
        if let Ok(gradient) = ctx.create_radial_gradient(c, c, 0.0, c, c, c) {
            let _ = gradient.add_color_stop(0.0_f32, "rgba(255, 255, 220, 0.9)");
            let _ = gradient.add_color_stop(0.3_f32, "rgba(255, 200, 80, 0.5)");
            let _ = gradient.add_color_stop(1.0_f32, "transparent");
            ctx.set_fill_style(&gradient);
            ctx.fill_rect(0.0, 0.0, size as f64, size as f64);
        }

        *opt = Some(canvas.clone());
        canvas
    })
}

fn get_fill_sprite() -> HtmlCanvasElement {
    FILL_SPRITE.with(|fs| {
        let mut opt = fs.borrow_mut();
        if let Some(canvas) = opt.as_ref() {
            return canvas.clone();
        }

        let document = web_sys::window().unwrap().document().unwrap();
        let canvas: HtmlCanvasElement = document.create_element("canvas").unwrap().unchecked_into();
        let size = 64u32;
        canvas.set_width(size);
        canvas.set_height(size);
        let ctx: CanvasRenderingContext2d = canvas
            .get_context("2d").unwrap().unwrap().unchecked_into();

        let c = size as f64 / 2.0;
        if let Ok(gradient) = ctx.create_radial_gradient(c, c, 0.0, c, c, c) {
            let _ = gradient.add_color_stop(0.0_f32, "rgba(255, 150, 50, 0.3)");
            let _ = gradient.add_color_stop(0.6_f32, "rgba(255, 100, 20, 0.1)");
            let _ = gradient.add_color_stop(1.0_f32, "transparent");
            ctx.set_fill_style(&gradient);
            ctx.fill_rect(0.0, 0.0, size as f64, size as f64);
        }

        *opt = Some(canvas.clone());
        canvas
    })
}

pub fn add_engine_particles(
    particles: &mut Vec<Particle>,
    x: f64, y: f64, rotation: f64, speed: f64, ship_type: i32,
) {
    if speed < 20.0 { return; }
    init_rng_if_needed();
    if particles.len() >= MAX_PARTICLES { return; }

    let idx = (ship_type as usize).min(SHIP_COLORS.len() - 1);
    let engine_color = SHIP_COLORS[idx].engine.to_string();
    let count = ((speed / 50.0) as usize).min(5);

    for _ in 0..count {
        if particles.len() >= MAX_PARTICLES { break; }
        let angle = rotation + std::f64::consts::PI + (fast_random() - 0.5) * 0.6;
        let spd = speed * 0.3 + fast_random() * 80.0;
        let life = 0.3 + fast_random() * 0.3;
        particles.push(Particle {
            x: x - rotation.cos() * 15.0 + (fast_random() - 0.5) * 6.0,
            y: y - rotation.sin() * 15.0 + (fast_random() - 0.5) * 6.0,
            vx: angle.cos() * spd,
            vy: angle.sin() * spd,
            life,
            max_life: life,
            size: 2.0 + fast_random() * 3.0,
            color: engine_color.clone(),
            kind: ParticleKind::Engine,
        });
    }
}

pub fn add_explosion(
    particles: &mut Vec<Particle>,
    explosions: &mut Vec<Explosion>,
    x: f64, y: f64,
) {
    init_rng_if_needed();
    // Hot core particles - bright white/yellow, fast but short-lived
    let core_colors = ["#ffffff", "#ffffcc", "#ffeeaa"];
    for _ in 0..5 {
        if particles.len() >= MAX_PARTICLES { break; }
        let angle = fast_random() * std::f64::consts::PI * 2.0;
        let spd = 40.0 + fast_random() * 120.0;
        let life = 0.15 + fast_random() * 0.2;
        let ci = (fast_random() * core_colors.len() as f64) as usize;
        particles.push(Particle {
            x: x + (fast_random() - 0.5) * 4.0,
            y: y + (fast_random() - 0.5) * 4.0,
            vx: angle.cos() * spd,
            vy: angle.sin() * spd,
            life, max_life: life,
            size: 4.0 + fast_random() * 4.0,
            color: core_colors[ci % core_colors.len()].to_string(),
            kind: ParticleKind::Explosion,
        });
    }

    // Fire particles - orange/red, medium speed
    let fire_colors = ["#ff4400", "#ff6600", "#ff8800", "#ffaa00", "#ff2200"];
    for i in 0..12 {
        if particles.len() >= MAX_PARTICLES { break; }
        let angle = (std::f64::consts::PI * 2.0 * i as f64) / 12.0
            + (fast_random() - 0.5) * 0.8;
        let spd = 80.0 + fast_random() * 250.0;
        let life = 0.4 + fast_random() * 0.6;
        let ci = (fast_random() * fire_colors.len() as f64) as usize;
        particles.push(Particle {
            x: x + (fast_random() - 0.5) * 8.0,
            y: y + (fast_random() - 0.5) * 8.0,
            vx: angle.cos() * spd,
            vy: angle.sin() * spd,
            life, max_life: life,
            size: 3.0 + fast_random() * 5.0,
            color: fire_colors[ci % fire_colors.len()].to_string(),
            kind: ParticleKind::Explosion,
        });
    }

    // Smoke/ember particles - dark red/grey, slow, long-lived
    let smoke_colors = ["#882200", "#664422", "#553311", "#aa4400"];
    for _ in 0..8 {
        if particles.len() >= MAX_PARTICLES { break; }
        let angle = fast_random() * std::f64::consts::PI * 2.0;
        let spd = 20.0 + fast_random() * 80.0;
        let life = 0.8 + fast_random() * 1.0;
        let ci = (fast_random() * smoke_colors.len() as f64) as usize;
        particles.push(Particle {
            x: x + (fast_random() - 0.5) * 12.0,
            y: y + (fast_random() - 0.5) * 12.0,
            vx: angle.cos() * spd,
            vy: angle.sin() * spd,
            life, max_life: life,
            size: 4.0 + fast_random() * 6.0,
            color: smoke_colors[ci % smoke_colors.len()].to_string(),
            kind: ParticleKind::Explosion,
        });
    }

    // Primary shockwave - fast expanding
    explosions.push(Explosion {
        x, y,
        radius: 0.0,
        max_radius: 90.0,
        life: 0.35,
        max_life: 0.35,
    });

    // Secondary shockwave - slower, wider
    explosions.push(Explosion {
        x, y,
        radius: 0.0,
        max_radius: 50.0,
        life: 0.5,
        max_life: 0.5,
    });
}

pub fn update_particles(particles: &mut Vec<Particle>, explosions: &mut Vec<Explosion>, dt: f64) {
    let mut i = 0;
    while i < particles.len() {
        let p = &mut particles[i];
        p.x += p.vx * dt;
        p.y += p.vy * dt;
        p.life -= dt;
        p.vx *= 0.98;
        p.vy *= 0.98;

        if p.life <= 0.0 {
            particles.swap_remove(i);
        } else {
            i += 1;
        }
    }

    let mut j = 0;
    while j < explosions.len() {
        let e = &mut explosions[j];
        e.life -= dt;
        e.radius = e.max_radius * (1.0 - e.life / e.max_life);
        if e.life <= 0.0 {
            explosions.swap_remove(j);
        } else {
            j += 1;
        }
    }
}

/// Draw a glowing engine beam behind a ship (Star Wars style thrust)
pub fn draw_engine_beam(ctx: &CanvasRenderingContext2d, sx: f64, sy: f64, rotation: f64, speed: f64, ship_type: i32, boosting: bool) {
    if speed < 15.0 && !boosting { return; }
    init_rng_if_needed();

    let mut intensity = ((speed - 15.0) / 200.0).min(1.0).max(0.0);
    let boost_mul = if boosting { 2.0 } else { 1.0 };
    intensity = (intensity * boost_mul).min(1.5);

    // Flicker: random jitter each frame for realistic thruster effect
    let flicker = 0.85 + fast_random() * 0.3;
    let len_flicker = 0.9 + fast_random() * 0.2;
    let base_len = if boosting { 30.0 + intensity * 35.0 } else { 18.0 + intensity * 22.0 };
    let base_width = if boosting { 4.0 + intensity * 4.0 } else { 3.0 + intensity * 3.0 };
    let beam_len = base_len * len_flicker;
    let beam_width = base_width * (0.9 + fast_random() * 0.2);
    let intensity = intensity * flicker;

    // Beam points backward from ship
    let bx = -rotation.cos();
    let by = -rotation.sin();
    // Perpendicular
    let px = -by;
    let py = bx;

    // Nozzle position (back of ship)
    let nozzle_x = sx + bx * 16.0;
    let nozzle_y = sy + by * 16.0;
    // Tip position
    let tip_x = nozzle_x + bx * beam_len;
    let tip_y = nozzle_y + by * beam_len;

    // Outer glow (wide, faint blue)
    ctx.save();
    ctx.set_global_alpha(intensity * 0.25);
    ctx.begin_path();
    ctx.move_to(nozzle_x + px * beam_width * 1.8, nozzle_y + py * beam_width * 1.8);
    ctx.line_to(nozzle_x - px * beam_width * 1.8, nozzle_y - py * beam_width * 1.8);
    ctx.line_to(tip_x, tip_y);
    ctx.close_path();

    let idx = (ship_type as usize).min(SHIP_COLORS.len() - 1);
    let glow_color = match idx {
        0 => "rgba(255, 100, 50, 0.6)",
        1 => "rgba(50, 150, 255, 0.6)",
        2 => "rgba(50, 255, 100, 0.6)",
        _ => "rgba(255, 200, 50, 0.6)",
    };
    ctx.set_fill_style(&wasm_bindgen::JsValue::from_str(glow_color));
    ctx.fill();

    // Core beam (bright white-blue, narrower)
    ctx.set_global_alpha(intensity * 0.7);
    ctx.begin_path();
    ctx.move_to(nozzle_x + px * beam_width, nozzle_y + py * beam_width);
    ctx.line_to(nozzle_x - px * beam_width, nozzle_y - py * beam_width);
    ctx.line_to(tip_x, tip_y);
    ctx.close_path();

    let core_color = match idx {
        0 => "rgba(255, 180, 120, 0.8)",
        1 => "rgba(150, 200, 255, 0.8)",
        2 => "rgba(150, 255, 180, 0.8)",
        _ => "rgba(255, 240, 150, 0.8)",
    };
    ctx.set_fill_style(&wasm_bindgen::JsValue::from_str(core_color));
    ctx.fill();

    ctx.restore();
}

pub fn render_particles(ctx: &CanvasRenderingContext2d, particles: &[Particle], offset_x: f64, offset_y: f64, vw: f64, vh: f64) {
    for p in particles {
        let sx = p.x - offset_x;
        let sy = p.y - offset_y;
        if sx < -20.0 || sx > vw + 20.0 || sy < -20.0 || sy > vh + 20.0 { continue; }

        let t = (p.life / p.max_life).max(0.0); // 1.0 = just born, 0.0 = dead

        if p.kind == ParticleKind::Explosion {
            // Explosion particles: soft glow that expands and fades
            let size = p.size * (1.0 + (1.0 - t) * 1.5);
            let alpha = t * t; // quadratic fade for smoother look

            // Draw soft glow with cached sprite
            ctx.set_global_alpha(alpha * 0.9);
            let glow = get_particle_glow(&p.color);
            let _ = ctx.draw_image_with_html_canvas_element_and_dw_and_dh(
                &glow, sx - size, sy - size, size * 2.0, size * 2.0,
            );
        } else {
            // Engine particles: simple dots that shrink
            let size = p.size * t;
            ctx.set_global_alpha(t);
            ctx.set_fill_style(&wasm_bindgen::JsValue::from_str(&p.color));
            if size < 3.0 {
                ctx.fill_rect(sx - size, sy - size, size * 2.0, size * 2.0);
            } else {
                ctx.begin_path();
                let _ = ctx.arc(sx, sy, size, 0.0, std::f64::consts::PI * 2.0);
                ctx.fill();
            }
        }
    }
    ctx.set_global_alpha(1.0);
}

pub fn render_explosions(ctx: &CanvasRenderingContext2d, explosions: &[Explosion], offset_x: f64, offset_y: f64, vw: f64, vh: f64) {
    for e in explosions {
        let sx = e.x - offset_x;
        let sy = e.y - offset_y;
        if sx < -100.0 || sx > vw + 100.0 || sy < -100.0 || sy > vh + 100.0 { continue; }

        let t = (e.life / e.max_life).max(0.0); // 1.0 = just started, 0.0 = done

        // Bright flash at center (early phase)
        if t > 0.6 {
            let flash_alpha = (t - 0.6) / 0.4; // 1.0 at start, 0.0 at t=0.6
            let flash_r = e.radius * 0.5;
            let r = flash_r.max(5.0);
            ctx.set_global_alpha(flash_alpha * 0.5);
            let flash = get_flash_sprite();
            let _ = ctx.draw_image_with_html_canvas_element_and_dw_and_dh(
                &flash, sx - r, sy - r, r * 2.0, r * 2.0,
            );
        }

        // Shockwave ring with gradient thickness
        let ring_alpha = t * 0.6;
        let ring_width = 4.0 + (1.0 - t) * 2.0; // thins as it expands
        ctx.set_global_alpha(ring_alpha);
        ctx.set_stroke_style(&wasm_bindgen::JsValue::from_str("rgba(255, 180, 50, 0.8)"));
        ctx.set_line_width(ring_width);
        ctx.begin_path();
        let _ = ctx.arc(sx, sy, e.radius, 0.0, std::f64::consts::PI * 2.0);
        ctx.stroke();

        // Inner faint fill (hot gas)
        if t > 0.3 {
            let fill_alpha = (t - 0.3) * 0.15;
            ctx.set_global_alpha(fill_alpha);
            let fill = get_fill_sprite();
            let r = e.radius;
            let _ = ctx.draw_image_with_html_canvas_element_and_dw_and_dh(
                &fill, sx - r, sy - r, r * 2.0, r * 2.0,
            );
        }
    }
    ctx.set_global_alpha(1.0);
}
