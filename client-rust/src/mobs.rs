use web_sys::CanvasRenderingContext2d;
use crate::ships;
use crate::effects;

/// Render a single mob with pre-interpolated position/rotation
pub fn render_mob(
    ctx: &CanvasRenderingContext2d,
    x: f64, y: f64, r: f64, vx: f64, vy: f64, hp: i32, mhp: i32, ship_type: i32,
    offset_x: f64, offset_y: f64, vw: f64, vh: f64,
) {
    let sx = x - offset_x;
    let sy = y - offset_y;
    let is_sd = ship_type == 3;
    let margin = if is_sd { 200.0 } else { 60.0 };
    if sx < -margin || sx > vw + margin || sy < -margin || sy > vh + margin { return; }

    let speed = (vx * vx + vy * vy).sqrt();
    effects::draw_engine_beam(ctx, sx, sy, r, speed, ship_type, false);
    ships::draw_ship(ctx, sx, sy, r, ship_type);

    // Health bar above mob (scaled for Star Destroyer)
    let scale = if is_sd { 5.0 } else { 1.0 };
    let bar_w = if is_sd { 120.0 } else { 40.0 };
    let bar_h = 4.0;
    let bar_y = sy - 30.0 * scale - 5.0;
    let ratio = hp as f64 / mhp as f64;

    ctx.set_fill_style_str("rgba(0,0,0,0.5)");
    ctx.fill_rect(sx - bar_w / 2.0, bar_y, bar_w, bar_h);

    let color = if ratio > 0.6 { "#ff8844" } else if ratio > 0.3 { "#ffaa00" } else { "#ff4444" };
    ctx.set_fill_style_str(color);
    ctx.fill_rect(sx - bar_w / 2.0, bar_y, bar_w * ratio, bar_h);
}
