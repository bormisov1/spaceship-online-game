use std::collections::HashMap;
use web_sys::CanvasRenderingContext2d;
use crate::constants::PICKUP_RENDER_SIZE;
use crate::protocol::PickupState;

pub fn render_pickups(
    ctx: &CanvasRenderingContext2d,
    pickups: &HashMap<String, PickupState>,
    offset_x: f64, offset_y: f64, vw: f64, vh: f64,
    time: f64,
) {
    for (_, pk) in pickups {
        let sx = pk.x - offset_x;
        let sy = pk.y - offset_y;
        if sx < -40.0 || sx > vw + 40.0 || sy < -40.0 || sy > vh + 40.0 { continue; }

        // Pulsing green glow
        let pulse = 0.5 + 0.5 * (time * 3.0).sin();
        let size = PICKUP_RENDER_SIZE * (0.8 + 0.2 * pulse);

        // Outer glow
        ctx.set_global_alpha(0.15 + 0.1 * pulse);
        if let Ok(gradient) = ctx.create_radial_gradient(sx, sy, 0.0, sx, sy, size) {
            let _ = gradient.add_color_stop(0.0_f32, "rgba(0, 255, 100, 0.6)");
            let _ = gradient.add_color_stop(1.0_f32, "transparent");
            ctx.set_fill_style(&gradient);
            ctx.begin_path();
            let _ = ctx.arc(sx, sy, size, 0.0, std::f64::consts::PI * 2.0);
            ctx.fill();
        }

        // Inner core
        ctx.set_global_alpha(0.6 + 0.3 * pulse);
        ctx.set_fill_style(&wasm_bindgen::JsValue::from_str("#44ff88"));
        ctx.begin_path();
        let _ = ctx.arc(sx, sy, 6.0, 0.0, std::f64::consts::PI * 2.0);
        ctx.fill();

        // Cross symbol
        ctx.set_stroke_style(&wasm_bindgen::JsValue::from_str("#ffffff"));
        ctx.set_line_width(2.0);
        ctx.set_global_alpha(0.5 + 0.3 * pulse);
        ctx.begin_path();
        ctx.move_to(sx - 4.0, sy);
        ctx.line_to(sx + 4.0, sy);
        ctx.move_to(sx, sy - 4.0);
        ctx.line_to(sx, sy + 4.0);
        ctx.stroke();

        ctx.set_global_alpha(1.0);
    }
}
