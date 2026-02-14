use std::cell::RefCell;
use std::collections::HashMap;
use wasm_bindgen::JsCast;
use web_sys::{CanvasRenderingContext2d, HtmlCanvasElement};
use crate::constants::LASER_COLORS;
use crate::protocol::ProjectileState;

thread_local! {
    static GLOW_SPRITES: RefCell<HashMap<String, HtmlCanvasElement>> = RefCell::new(HashMap::new());
    static BOLT_SPRITES: RefCell<HashMap<String, HtmlCanvasElement>> = RefCell::new(HashMap::new());
}

fn get_glow_sprite(color: &str) -> HtmlCanvasElement {
    GLOW_SPRITES.with(|gs| {
        let mut sprites = gs.borrow_mut();
        if let Some(canvas) = sprites.get(color) {
            return canvas.clone();
        }

        let document = web_sys::window().unwrap().document().unwrap();
        let canvas: HtmlCanvasElement = document.create_element("canvas").unwrap().unchecked_into();
        let size = 48;
        canvas.set_width(size);
        canvas.set_height(size);
        let ctx: CanvasRenderingContext2d = canvas
            .get_context("2d").unwrap().unwrap().unchecked_into();

        let cx = size as f64 / 2.0;
        let cy = size as f64 / 2.0;

        if let Ok(gradient) = ctx.create_radial_gradient(cx, cy, 0.0, cx, cy, cx) {
            let _ = gradient.add_color_stop(0.0_f32, "rgba(255,255,255,0.7)");
            let _ = gradient.add_color_stop(0.15_f32, color);
            let _ = gradient.add_color_stop(0.5_f32, &format!("{}88", color));
            let _ = gradient.add_color_stop(1.0_f32, "transparent");
            ctx.set_fill_style(&gradient);
            ctx.fill_rect(0.0, 0.0, size as f64, size as f64);
        }

        sprites.insert(color.to_string(), canvas.clone());
        canvas
    })
}

fn get_bolt_sprite(color: &str) -> HtmlCanvasElement {
    BOLT_SPRITES.with(|bs| {
        let mut sprites = bs.borrow_mut();
        if let Some(canvas) = sprites.get(color) {
            return canvas.clone();
        }

        let document = web_sys::window().unwrap().document().unwrap();
        let canvas: HtmlCanvasElement = document.create_element("canvas").unwrap().unchecked_into();
        let w = 40u32;
        let h = 10u32;
        canvas.set_width(w);
        canvas.set_height(h);
        let ctx: CanvasRenderingContext2d = canvas
            .get_context("2d").unwrap().unwrap().unchecked_into();

        let cx = w as f64 / 2.0;
        let cy = h as f64 / 2.0;

        // Outer colored glow
        ctx.set_global_alpha(0.3);
        ctx.set_fill_style(&wasm_bindgen::JsValue::from_str(color));
        ctx.begin_path();
        let _ = ctx.ellipse(cx, cy, 18.0, 3.5, 0.0, 0.0, std::f64::consts::PI * 2.0);
        ctx.fill();

        // Mid body glow
        ctx.set_global_alpha(0.6);
        ctx.set_fill_style(&wasm_bindgen::JsValue::from_str(color));
        ctx.begin_path();
        let _ = ctx.ellipse(cx, cy, 15.0, 2.2, 0.0, 0.0, std::f64::consts::PI * 2.0);
        ctx.fill();

        // Bright white core
        ctx.set_global_alpha(0.95);
        ctx.set_fill_style(&wasm_bindgen::JsValue::from_str("#ffffff"));
        ctx.begin_path();
        let _ = ctx.ellipse(cx, cy, 12.0, 1.4, 0.0, 0.0, std::f64::consts::PI * 2.0);
        ctx.fill();

        // Front tip highlight
        ctx.set_global_alpha(1.0);
        ctx.set_fill_style(&wasm_bindgen::JsValue::from_str("#ffffff"));
        ctx.begin_path();
        let _ = ctx.arc(cx + 10.0, cy, 1.8, 0.0, std::f64::consts::PI * 2.0);
        ctx.fill();

        sprites.insert(color.to_string(), canvas.clone());
        canvas
    })
}

pub fn render_projectiles(
    ctx: &CanvasRenderingContext2d,
    projectiles: &HashMap<String, ProjectileState>,
    players: &HashMap<String, crate::protocol::PlayerState>,
    offset_x: f64, offset_y: f64, vw: f64, vh: f64,
) {
    for (_, proj) in projectiles {
        let sx = proj.x - offset_x;
        let sy = proj.y - offset_y;
        if sx < -50.0 || sx > vw + 50.0 || sy < -50.0 || sy > vh + 50.0 { continue; }

        // Determine color from owner ship type
        let ship_type = players.get(&proj.o).map(|p| p.s).unwrap_or(0);
        let color_idx = (ship_type as usize).min(LASER_COLORS.len() - 1);
        let color = LASER_COLORS[color_idx];

        // Glow sprite (ambient light around bolt)
        let sprite = get_glow_sprite(color);
        let glow_size = 15.0;
        ctx.save();
        ctx.set_global_alpha(0.8);
        let _ = ctx.draw_image_with_html_canvas_element_and_dw_and_dh(
            &sprite, sx - glow_size, sy - glow_size, glow_size * 2.0, glow_size * 2.0,
        );
        ctx.restore();

        // Star Wars laser bolt: pre-rendered sprite
        let bolt = get_bolt_sprite(color);
        ctx.save();
        ctx.translate(sx, sy).unwrap_or(());
        ctx.rotate(proj.r).unwrap_or(());
        let _ = ctx.draw_image_with_html_canvas_element_and_dw_and_dh(
            &bolt, -20.0, -5.0, 40.0, 10.0,
        );
        ctx.restore();
    }
}
