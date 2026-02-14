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
    let size = PICKUP_RENDER_SIZE * 2.5; // 2.5x larger radius

    for (_, pk) in pickups {
        let sx = pk.x - offset_x;
        let sy = pk.y - offset_y;
        if sx < -size - 20.0 || sx > vw + size + 20.0 || sy < -size - 20.0 || sy > vh + size + 20.0 { continue; }

        let pulse = 0.5 + 0.5 * (time * 3.0).sin();
        let glow_size = size * (0.85 + 0.15 * pulse);

        // Outer radial glow (sun-like halo)
        ctx.set_global_alpha(0.12 + 0.08 * pulse);
        if let Ok(gradient) = ctx.create_radial_gradient(sx, sy, 0.0, sx, sy, glow_size * 1.3) {
            let _ = gradient.add_color_stop(0.0_f32, "rgba(100, 255, 150, 0.5)");
            let _ = gradient.add_color_stop(0.4_f32, "rgba(0, 255, 100, 0.2)");
            let _ = gradient.add_color_stop(1.0_f32, "transparent");
            ctx.set_fill_style(&gradient);
            ctx.begin_path();
            let _ = ctx.arc(sx, sy, glow_size * 1.3, 0.0, std::f64::consts::PI * 2.0);
            ctx.fill();
        }

        // Inner bright glow
        ctx.set_global_alpha(0.2 + 0.15 * pulse);
        if let Ok(gradient) = ctx.create_radial_gradient(sx, sy, 0.0, sx, sy, glow_size * 0.6) {
            let _ = gradient.add_color_stop(0.0_f32, "rgba(200, 255, 220, 0.8)");
            let _ = gradient.add_color_stop(0.5_f32, "rgba(0, 255, 100, 0.3)");
            let _ = gradient.add_color_stop(1.0_f32, "transparent");
            ctx.set_fill_style(&gradient);
            ctx.begin_path();
            let _ = ctx.arc(sx, sy, glow_size * 0.6, 0.0, std::f64::consts::PI * 2.0);
            ctx.fill();
        }

        // Sharp-edged aesthetic plus sign (diamond-shaped arms that widen toward center)
        // Each arm is a triangle: sharp point at the tip, widening to the center
        let arm_len = glow_size * 0.55; // length from center to tip
        let arm_width = glow_size * 0.22; // half-width at the base (center intersection)

        ctx.set_global_alpha(0.6 + 0.3 * pulse);

        // Gradient fill for the plus
        if let Ok(gradient) = ctx.create_radial_gradient(sx, sy, 0.0, sx, sy, arm_len) {
            let _ = gradient.add_color_stop(0.0_f32, "rgba(255, 255, 255, 0.95)");
            let _ = gradient.add_color_stop(0.3_f32, "rgba(150, 255, 200, 0.8)");
            let _ = gradient.add_color_stop(0.7_f32, "rgba(0, 255, 100, 0.5)");
            let _ = gradient.add_color_stop(1.0_f32, "rgba(0, 200, 80, 0.1)");
            ctx.set_fill_style(&gradient);
        }

        ctx.begin_path();
        // Right arm: sharp tip at right, widens to center
        ctx.move_to(sx + arm_len, sy);              // tip (sharp point)
        ctx.line_to(sx + arm_width * 0.3, sy - arm_width); // top-left of base
        ctx.line_to(sx + arm_width * 0.3, sy + arm_width); // bottom-left of base
        ctx.close_path();

        // Left arm
        ctx.move_to(sx - arm_len, sy);
        ctx.line_to(sx - arm_width * 0.3, sy - arm_width);
        ctx.line_to(sx - arm_width * 0.3, sy + arm_width);
        ctx.close_path();

        // Top arm
        ctx.move_to(sx, sy - arm_len);
        ctx.line_to(sx - arm_width, sy - arm_width * 0.3);
        ctx.line_to(sx + arm_width, sy - arm_width * 0.3);
        ctx.close_path();

        // Bottom arm
        ctx.move_to(sx, sy + arm_len);
        ctx.line_to(sx - arm_width, sy + arm_width * 0.3);
        ctx.line_to(sx + arm_width, sy + arm_width * 0.3);
        ctx.close_path();

        ctx.fill();

        // Center diamond (fills the intersection)
        ctx.set_global_alpha(0.7 + 0.25 * pulse);
        ctx.set_fill_style(&wasm_bindgen::JsValue::from_str("rgba(220, 255, 240, 0.9)"));
        ctx.begin_path();
        ctx.move_to(sx, sy - arm_width);
        ctx.line_to(sx + arm_width, sy);
        ctx.line_to(sx, sy + arm_width);
        ctx.line_to(sx - arm_width, sy);
        ctx.close_path();
        ctx.fill();

        // White hot core dot
        ctx.set_global_alpha(0.8 + 0.2 * pulse);
        ctx.set_fill_style(&wasm_bindgen::JsValue::from_str("#ffffff"));
        ctx.begin_path();
        let _ = ctx.arc(sx, sy, 3.0, 0.0, std::f64::consts::PI * 2.0);
        ctx.fill();

        ctx.set_global_alpha(1.0);
    }
}
