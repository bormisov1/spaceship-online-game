use std::collections::HashMap;
use web_sys::CanvasRenderingContext2d;
use crate::protocol::MobState;
use crate::ships;

pub fn render_mobs(
    ctx: &CanvasRenderingContext2d,
    mobs: &HashMap<String, MobState>,
    offset_x: f64, offset_y: f64, vw: f64, vh: f64,
) {
    for (_, mob) in mobs {
        if !mob.a { continue; }
        let sx = mob.x - offset_x;
        let sy = mob.y - offset_y;
        if sx < -60.0 || sx > vw + 60.0 || sy < -60.0 || sy > vh + 60.0 { continue; }

        // Draw ship type 3 (Destroyer)
        ships::draw_ship(ctx, sx, sy, mob.r, 3);

        // Health bar above mob
        let bar_w = 40.0;
        let bar_h = 4.0;
        let bar_y = sy - 35.0;
        let ratio = mob.hp as f64 / mob.mhp as f64;

        ctx.set_fill_style(&wasm_bindgen::JsValue::from_str("rgba(0,0,0,0.5)"));
        ctx.fill_rect(sx - bar_w / 2.0, bar_y, bar_w, bar_h);

        let color = if ratio > 0.6 { "#ff8844" } else if ratio > 0.3 { "#ffaa00" } else { "#ff4444" };
        ctx.set_fill_style(&wasm_bindgen::JsValue::from_str(color));
        ctx.fill_rect(sx - bar_w / 2.0, bar_y, bar_w * ratio, bar_h);
    }
}
