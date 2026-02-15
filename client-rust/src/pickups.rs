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
        let circle_r = glow_size * 0.45;

        // Outer radial glow (sun-like halo)
        ctx.set_global_alpha(0.12 + 0.08 * pulse);
        if let Ok(gradient) = ctx.create_radial_gradient(sx, sy, 0.0, sx, sy, glow_size * 1.3) {
            let _ = gradient.add_color_stop(0.0_f32, "rgba(100, 255, 150, 0.5)");
            let _ = gradient.add_color_stop(0.4_f32, "rgba(0, 255, 100, 0.2)");
            let _ = gradient.add_color_stop(1.0_f32, "transparent");
            ctx.set_fill_style_canvas_gradient(&gradient);
            ctx.begin_path();
            let _ = ctx.arc(sx, sy, glow_size * 1.3, 0.0, std::f64::consts::PI * 2.0);
            ctx.fill();
        }

        // Green filled circle
        ctx.set_global_alpha(0.5 + 0.2 * pulse);
        if let Ok(gradient) = ctx.create_radial_gradient(sx, sy, 0.0, sx, sy, circle_r) {
            let _ = gradient.add_color_stop(0.0_f32, "rgba(80, 255, 140, 0.9)");
            let _ = gradient.add_color_stop(0.7_f32, "rgba(40, 200, 100, 0.7)");
            let _ = gradient.add_color_stop(1.0_f32, "rgba(20, 160, 80, 0.5)");
            ctx.set_fill_style_canvas_gradient(&gradient);
        }
        ctx.begin_path();
        let _ = ctx.arc(sx, sy, circle_r, 0.0, std::f64::consts::PI * 2.0);
        ctx.fill();

        // Green circle border
        ctx.set_global_alpha(0.6 + 0.3 * pulse);
        ctx.set_stroke_style_str("rgba(100, 255, 160, 0.8)");
        ctx.set_line_width(1.5);
        ctx.begin_path();
        let _ = ctx.arc(sx, sy, circle_r, 0.0, std::f64::consts::PI * 2.0);
        ctx.stroke();

        // White sharp-edged cross inside the green circle
        let arm_len = circle_r * 0.75;
        let arm_width = circle_r * 0.3;

        ctx.set_global_alpha(0.7 + 0.25 * pulse);

        // White gradient fill for the cross
        if let Ok(gradient) = ctx.create_radial_gradient(sx, sy, 0.0, sx, sy, arm_len) {
            let _ = gradient.add_color_stop(0.0_f32, "rgba(255, 255, 255, 0.95)");
            let _ = gradient.add_color_stop(0.5_f32, "rgba(240, 255, 245, 0.85)");
            let _ = gradient.add_color_stop(1.0_f32, "rgba(220, 255, 235, 0.6)");
            ctx.set_fill_style_canvas_gradient(&gradient);
        }

        ctx.begin_path();
        // Right arm
        ctx.move_to(sx + arm_len, sy);
        ctx.line_to(sx + arm_width * 0.3, sy - arm_width);
        ctx.line_to(sx + arm_width * 0.3, sy + arm_width);
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

        // White center diamond
        ctx.set_global_alpha(0.85 + 0.15 * pulse);
        ctx.set_fill_style_str("rgba(255, 255, 255, 0.9)");
        ctx.begin_path();
        ctx.move_to(sx, sy - arm_width);
        ctx.line_to(sx + arm_width, sy);
        ctx.line_to(sx, sy + arm_width);
        ctx.line_to(sx - arm_width, sy);
        ctx.close_path();
        ctx.fill();

        // White hot core dot
        ctx.set_global_alpha(0.9 + 0.1 * pulse);
        ctx.set_fill_style_str("#ffffff");
        ctx.begin_path();
        let _ = ctx.arc(sx, sy, 2.5, 0.0, std::f64::consts::PI * 2.0);
        ctx.fill();

        ctx.set_global_alpha(1.0);
    }
}
