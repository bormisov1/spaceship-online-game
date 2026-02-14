use web_sys::CanvasRenderingContext2d;
use crate::state::{Particle, ParticleKind, Explosion};
use crate::constants::SHIP_COLORS;

const MAX_PARTICLES: usize = 200;

pub fn add_engine_particles(
    particles: &mut Vec<Particle>,
    x: f64, y: f64, rotation: f64, speed: f64, ship_type: i32,
) {
    if speed < 20.0 { return; }
    if particles.len() >= MAX_PARTICLES { return; }

    let idx = (ship_type as usize).min(SHIP_COLORS.len() - 1);
    let engine_color = SHIP_COLORS[idx].engine.to_string();
    let count = ((speed / 50.0) as usize).min(5);

    for _ in 0..count {
        if particles.len() >= MAX_PARTICLES { break; }
        let angle = rotation + std::f64::consts::PI + (js_sys::Math::random() - 0.5) * 0.6;
        let spd = speed * 0.3 + js_sys::Math::random() * 80.0;
        let life = 0.3 + js_sys::Math::random() * 0.3;
        particles.push(Particle {
            x: x - rotation.cos() * 15.0 + (js_sys::Math::random() - 0.5) * 6.0,
            y: y - rotation.sin() * 15.0 + (js_sys::Math::random() - 0.5) * 6.0,
            vx: angle.cos() * spd,
            vy: angle.sin() * spd,
            life,
            max_life: life,
            size: 2.0 + js_sys::Math::random() * 3.0,
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
    // Hot core particles - bright white/yellow, fast but short-lived
    let core_colors = ["#ffffff", "#ffffcc", "#ffeeaa"];
    for _ in 0..8 {
        let angle = js_sys::Math::random() * std::f64::consts::PI * 2.0;
        let spd = 40.0 + js_sys::Math::random() * 120.0;
        let life = 0.15 + js_sys::Math::random() * 0.2;
        let ci = (js_sys::Math::random() * core_colors.len() as f64) as usize;
        particles.push(Particle {
            x: x + (js_sys::Math::random() - 0.5) * 4.0,
            y: y + (js_sys::Math::random() - 0.5) * 4.0,
            vx: angle.cos() * spd,
            vy: angle.sin() * spd,
            life, max_life: life,
            size: 4.0 + js_sys::Math::random() * 4.0,
            color: core_colors[ci % core_colors.len()].to_string(),
            kind: ParticleKind::Explosion,
        });
    }

    // Fire particles - orange/red, medium speed
    let fire_colors = ["#ff4400", "#ff6600", "#ff8800", "#ffaa00", "#ff2200"];
    for i in 0..20 {
        let angle = (std::f64::consts::PI * 2.0 * i as f64) / 20.0
            + (js_sys::Math::random() - 0.5) * 0.8;
        let spd = 80.0 + js_sys::Math::random() * 250.0;
        let life = 0.4 + js_sys::Math::random() * 0.6;
        let ci = (js_sys::Math::random() * fire_colors.len() as f64) as usize;
        particles.push(Particle {
            x: x + (js_sys::Math::random() - 0.5) * 8.0,
            y: y + (js_sys::Math::random() - 0.5) * 8.0,
            vx: angle.cos() * spd,
            vy: angle.sin() * spd,
            life, max_life: life,
            size: 3.0 + js_sys::Math::random() * 5.0,
            color: fire_colors[ci % fire_colors.len()].to_string(),
            kind: ParticleKind::Explosion,
        });
    }

    // Smoke/ember particles - dark red/grey, slow, long-lived
    let smoke_colors = ["#882200", "#664422", "#553311", "#aa4400"];
    for _ in 0..12 {
        let angle = js_sys::Math::random() * std::f64::consts::PI * 2.0;
        let spd = 20.0 + js_sys::Math::random() * 80.0;
        let life = 0.8 + js_sys::Math::random() * 1.0;
        let ci = (js_sys::Math::random() * smoke_colors.len() as f64) as usize;
        particles.push(Particle {
            x: x + (js_sys::Math::random() - 0.5) * 12.0,
            y: y + (js_sys::Math::random() - 0.5) * 12.0,
            vx: angle.cos() * spd,
            vy: angle.sin() * spd,
            life, max_life: life,
            size: 4.0 + js_sys::Math::random() * 6.0,
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

/// Draw a glowing engine beam behind a ship (Star Wars style blue thrust)
pub fn draw_engine_beam(ctx: &CanvasRenderingContext2d, sx: f64, sy: f64, rotation: f64, speed: f64, ship_type: i32) {
    if speed < 15.0 { return; }

    let intensity = ((speed - 15.0) / 200.0).min(1.0); // 0..1 based on speed
    // Flicker: random jitter each frame for realistic thruster effect
    let flicker = 0.85 + js_sys::Math::random() * 0.3; // 0.85-1.15
    let len_flicker = 0.9 + js_sys::Math::random() * 0.2; // 0.9-1.1
    let beam_len = (18.0 + intensity * 22.0) * len_flicker;
    let beam_width = (3.0 + intensity * 3.0) * (0.9 + js_sys::Math::random() * 0.2);
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
    ctx.set_global_alpha(intensity * 0.6);
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

    // Inner hot core (white, thinnest)
    ctx.set_global_alpha(intensity * 0.8);
    ctx.begin_path();
    ctx.move_to(nozzle_x + px * beam_width * 0.4, nozzle_y + py * beam_width * 0.4);
    ctx.line_to(nozzle_x - px * beam_width * 0.4, nozzle_y - py * beam_width * 0.4);
    ctx.line_to(nozzle_x + bx * beam_len * 0.6, nozzle_y + by * beam_len * 0.6);
    ctx.close_path();
    ctx.set_fill_style(&wasm_bindgen::JsValue::from_str("rgba(255, 255, 255, 0.9)"));
    ctx.fill();

    // Nozzle glow dot
    ctx.set_global_alpha(intensity * 0.5);
    ctx.begin_path();
    let _ = ctx.arc(nozzle_x, nozzle_y, beam_width * 1.2, 0.0, std::f64::consts::PI * 2.0);
    ctx.set_fill_style(&wasm_bindgen::JsValue::from_str("rgba(255, 255, 255, 0.6)"));
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

            // Draw soft glow with radial gradient
            ctx.set_global_alpha(alpha * 0.9);
            if let Ok(gradient) = ctx.create_radial_gradient(sx, sy, 0.0, sx, sy, size) {
                let _ = gradient.add_color_stop(0.0_f32, &p.color);
                let inner = format!("{}66", &p.color); // semi-transparent
                let _ = gradient.add_color_stop(0.4_f32, &inner);
                let _ = gradient.add_color_stop(1.0_f32, "transparent");
                ctx.set_fill_style(&gradient);
                ctx.begin_path();
                let _ = ctx.arc(sx, sy, size, 0.0, std::f64::consts::PI * 2.0);
                ctx.fill();
            }
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
            ctx.set_global_alpha(flash_alpha * 0.5);
            if let Ok(gradient) = ctx.create_radial_gradient(sx, sy, 0.0, sx, sy, flash_r.max(5.0)) {
                let _ = gradient.add_color_stop(0.0_f32, "rgba(255, 255, 220, 0.9)");
                let _ = gradient.add_color_stop(0.3_f32, "rgba(255, 200, 80, 0.5)");
                let _ = gradient.add_color_stop(1.0_f32, "transparent");
                ctx.set_fill_style(&gradient);
                ctx.begin_path();
                let _ = ctx.arc(sx, sy, flash_r.max(5.0), 0.0, std::f64::consts::PI * 2.0);
                ctx.fill();
            }
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
            if let Ok(gradient) = ctx.create_radial_gradient(sx, sy, 0.0, sx, sy, e.radius) {
                let _ = gradient.add_color_stop(0.0_f32, "rgba(255, 150, 50, 0.3)");
                let _ = gradient.add_color_stop(0.6_f32, "rgba(255, 100, 20, 0.1)");
                let _ = gradient.add_color_stop(1.0_f32, "transparent");
                ctx.set_fill_style(&gradient);
                ctx.begin_path();
                let _ = ctx.arc(sx, sy, e.radius, 0.0, std::f64::consts::PI * 2.0);
                ctx.fill();
            }
        }
    }
    ctx.set_global_alpha(1.0);
}
