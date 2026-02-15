use std::cell::RefCell;
use std::collections::HashMap;
use wasm_bindgen::JsCast;
use web_sys::{CanvasRenderingContext2d, HtmlCanvasElement};
use crate::state::{Particle, ParticleKind, Explosion, DamageNumber, HitMarker, MobSpeech, GameState};
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
            ctx.set_fill_style_canvas_gradient(&gradient);
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
            ctx.set_fill_style_canvas_gradient(&gradient);
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
            ctx.set_fill_style_canvas_gradient(&gradient);
            ctx.fill_rect(0.0, 0.0, size as f64, size as f64);
        }

        *opt = Some(canvas.clone());
        canvas
    })
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

    // Scale for Star Destroyer (5x larger ship)
    let ship_scale = if ship_type == 3 { 5.0 } else { 1.0 };
    let beam_len = beam_len * ship_scale;
    let beam_width = beam_width * ship_scale;

    // Beam points backward from ship
    let bx = -rotation.cos();
    let by = -rotation.sin();
    // Perpendicular
    let px = -by;
    let py = bx;

    // Nozzle position (back of ship)
    let nozzle_x = sx + bx * 16.0 * ship_scale;
    let nozzle_y = sy + by * 16.0 * ship_scale;
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
        3 => "rgba(50, 255, 100, 0.6)",  // Star Destroyer
        4 | 5 => "rgba(100, 100, 255, 0.6)", // TIE fighters
        _ => "rgba(255, 200, 50, 0.6)",
    };
    ctx.set_fill_style_str(glow_color);
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
    ctx.set_fill_style_str(core_color);
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
            ctx.set_fill_style_str(&p.color);
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

// --- Screen Shake ---

pub fn trigger_shake(state: &mut GameState, intensity: f64) {
    state.shake_intensity = (state.shake_intensity + intensity).min(20.0);
    state.shake_decay = state.shake_intensity;
}

pub fn update_shake(state: &mut GameState, dt: f64) {
    if state.shake_intensity <= 0.0 {
        state.shake_x = 0.0;
        state.shake_y = 0.0;
        return;
    }
    init_rng_if_needed();
    let angle = fast_random() * std::f64::consts::PI * 2.0;
    state.shake_x = angle.cos() * state.shake_intensity;
    state.shake_y = angle.sin() * state.shake_intensity;
    state.shake_intensity -= state.shake_decay * dt * 6.0;
    if state.shake_intensity < 0.5 {
        state.shake_intensity = 0.0;
        state.shake_x = 0.0;
        state.shake_y = 0.0;
    }
}

// --- Damage Numbers ---

const MAX_DAMAGE_NUMBERS: usize = 30;

pub fn add_damage_number(state: &mut GameState, x: f64, y: f64, dmg: i32, is_heal: bool) {
    init_rng_if_needed();
    if state.damage_numbers.len() >= MAX_DAMAGE_NUMBERS {
        state.damage_numbers.remove(0);
    }
    state.damage_numbers.push(DamageNumber {
        x,
        y,
        text: if is_heal { format!("+{}", dmg) } else { format!("-{}", dmg) },
        color: if is_heal { "#44ff44" } else { "#ff4444" },
        life: 1.0,
        max_life: 1.0,
        vy: -60.0,
        offset_x: (fast_random() - 0.5) * 20.0,
    });
}

pub fn update_damage_numbers(numbers: &mut Vec<DamageNumber>, dt: f64) {
    let mut i = 0;
    while i < numbers.len() {
        numbers[i].life -= dt;
        numbers[i].y += numbers[i].vy * dt;
        if numbers[i].life <= 0.0 {
            numbers.swap_remove(i);
        } else {
            i += 1;
        }
    }
}

pub fn render_damage_numbers(ctx: &CanvasRenderingContext2d, numbers: &[DamageNumber], offset_x: f64, offset_y: f64, vw: f64, vh: f64) {
    ctx.set_text_align("center");
    for dn in numbers {
        let sx = dn.x + dn.offset_x - offset_x;
        let sy = dn.y - offset_y;
        if sx < -50.0 || sx > vw + 50.0 || sy < -50.0 || sy > vh + 50.0 { continue; }

        let alpha = (dn.life / dn.max_life).max(0.0);
        let scale = 1.0 + (1.0 - alpha) * 0.3;
        let font_size = (14.0 * scale) as i32;

        ctx.set_global_alpha(alpha);
        ctx.set_font(&format!("bold {}px monospace", font_size));
        // Shadow
        ctx.set_fill_style_str("#000000");
        let _ = ctx.fill_text(&dn.text, sx + 1.0, sy + 1.0);
        // Color
        ctx.set_fill_style_str(dn.color);
        let _ = ctx.fill_text(&dn.text, sx, sy);
    }
    ctx.set_global_alpha(1.0);
}

// --- Hit Markers (screen-space) ---

const HIT_MARKER_DURATION: f64 = 0.25;

pub fn add_hit_marker(state: &mut GameState) {
    state.hit_markers.push(HitMarker {
        life: HIT_MARKER_DURATION,
        max_life: HIT_MARKER_DURATION,
    });
}

pub fn update_hit_markers(markers: &mut Vec<HitMarker>, dt: f64) {
    let mut i = 0;
    while i < markers.len() {
        markers[i].life -= dt;
        if markers[i].life <= 0.0 {
            markers.swap_remove(i);
        } else {
            i += 1;
        }
    }
}

pub fn render_hit_markers(ctx: &CanvasRenderingContext2d, markers: &[HitMarker], screen_w: f64, screen_h: f64) {
    if markers.is_empty() { return; }

    let cx = screen_w / 2.0;
    let cy = screen_h / 2.0;

    for hm in markers {
        let alpha = (hm.life / hm.max_life).max(0.0);
        let size = 10.0 + (1.0 - alpha) * 4.0;
        let gap = 3.0;

        ctx.set_global_alpha(alpha);
        ctx.set_stroke_style_str("#ffffff");
        ctx.set_line_width(2.5);
        ctx.begin_path();
        // Top-left to center
        ctx.move_to(cx - size, cy - size);
        ctx.line_to(cx - gap, cy - gap);
        // Top-right to center
        ctx.move_to(cx + size, cy - size);
        ctx.line_to(cx + gap, cy - gap);
        // Bottom-left to center
        ctx.move_to(cx - size, cy + size);
        ctx.line_to(cx - gap, cy + gap);
        // Bottom-right to center
        ctx.move_to(cx + size, cy + size);
        ctx.line_to(cx + gap, cy + gap);
        ctx.stroke();
    }
    ctx.set_global_alpha(1.0);
}

// --- Mob Speech Bubbles ---

const MOB_SPEECH_DURATION: f64 = 3000.0; // 3 seconds in ms

pub fn add_mob_speech(state: &mut GameState, mob_id: String, text: String) {
    let now = js_sys::Date::now();
    // Remove existing speech for this mob
    state.mob_speech.retain(|s| s.mob_id != mob_id);
    state.mob_speech.push(MobSpeech {
        mob_id,
        text,
        time: now,
    });
}

pub fn render_mob_speech(ctx: &CanvasRenderingContext2d, speech: &[MobSpeech], mobs: &std::collections::HashMap<String, crate::protocol::MobState>, offset_x: f64, offset_y: f64, vw: f64, vh: f64) {
    let now = js_sys::Date::now();

    for s in speech {
        let age = now - s.time;
        if age > MOB_SPEECH_DURATION { continue; }

        let mob = match mobs.get(&s.mob_id) {
            Some(m) if m.a => m,
            _ => continue,
        };

        let sx = mob.x - offset_x;
        let sy = mob.y - offset_y;
        if sx < -100.0 || sx > vw + 100.0 || sy < -100.0 || sy > vh + 100.0 { continue; }

        // Fade in/out
        let alpha = if age < 200.0 {
            age / 200.0
        } else if age > MOB_SPEECH_DURATION - 500.0 {
            (MOB_SPEECH_DURATION - age) / 500.0
        } else {
            1.0
        }.max(0.0);

        // Bubble position above mob (scaled for Star Destroyer)
        let bubble_offset = if mob.s == 3 { 170.0 } else { 50.0 };
        let bx = sx;
        let by = sy - bubble_offset;

        ctx.set_global_alpha(alpha);
        ctx.set_font("12px monospace");
        ctx.set_text_align("center");

        // Measure text for bubble background
        let metrics = ctx.measure_text(&s.text).unwrap_or_else(|_| ctx.measure_text("").unwrap());
        let tw = metrics.width();
        let pad = 6.0;
        let bw = tw + pad * 2.0;
        let bh = 20.0;

        // Bubble background
        ctx.set_fill_style_str("rgba(0, 0, 0, 0.7)");
        let corner_r = 6.0;
        ctx.begin_path();
        let _ = ctx.arc(bx - bw / 2.0 + corner_r, by - bh / 2.0 + corner_r, corner_r, std::f64::consts::PI, 1.5 * std::f64::consts::PI);
        let _ = ctx.arc(bx + bw / 2.0 - corner_r, by - bh / 2.0 + corner_r, corner_r, 1.5 * std::f64::consts::PI, 0.0);
        let _ = ctx.arc(bx + bw / 2.0 - corner_r, by + bh / 2.0 - corner_r, corner_r, 0.0, 0.5 * std::f64::consts::PI);
        let _ = ctx.arc(bx - bw / 2.0 + corner_r, by + bh / 2.0 - corner_r, corner_r, 0.5 * std::f64::consts::PI, std::f64::consts::PI);
        ctx.close_path();
        ctx.fill();

        // Bubble border
        ctx.set_stroke_style_str("rgba(255, 200, 50, 0.5)");
        ctx.set_line_width(1.0);
        ctx.stroke();

        // Small triangle pointing down to mob
        ctx.begin_path();
        ctx.move_to(bx - 4.0, by + bh / 2.0);
        ctx.line_to(bx, by + bh / 2.0 + 5.0);
        ctx.line_to(bx + 4.0, by + bh / 2.0);
        ctx.close_path();
        ctx.set_fill_style_str("rgba(0, 0, 0, 0.7)");
        ctx.fill();

        // Text
        ctx.set_fill_style_str("#ffffff");
        let _ = ctx.fill_text(&s.text, bx, by + 4.0);
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
        ctx.set_stroke_style_str("rgba(255, 180, 50, 0.8)");
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
