use web_sys::CanvasRenderingContext2d;
use crate::ships;
use crate::effects;

/// Render a single mob with pre-interpolated position/rotation
pub fn render_mob(
    ctx: &CanvasRenderingContext2d,
    x: f64, y: f64, r: f64, vx: f64, vy: f64, hp: i32, mhp: i32,
    offset_x: f64, offset_y: f64, vw: f64, vh: f64,
) {
    let sx = x - offset_x;
    let sy = y - offset_y;
    if sx < -60.0 || sx > vw + 60.0 || sy < -60.0 || sy > vh + 60.0 { return; }

    let speed = (vx * vx + vy * vy).sqrt();
    effects::draw_engine_beam(ctx, sx, sy, r, speed, 3, false);
    ships::draw_ship(ctx, sx, sy, r, 3);

    // Health bar above mob
    let bar_w = 40.0;
    let bar_h = 4.0;
    let bar_y = sy - 35.0;
    let ratio = hp as f64 / mhp as f64;

    ctx.set_fill_style_str("rgba(0,0,0,0.5)");
    ctx.fill_rect(sx - bar_w / 2.0, bar_y, bar_w, bar_h);

    let color = if ratio > 0.6 { "#ff8844" } else if ratio > 0.3 { "#ffaa00" } else { "#ff4444" };
    ctx.set_fill_style_str(color);
    ctx.fill_rect(sx - bar_w / 2.0, bar_y, bar_w * ratio, bar_h);
}
