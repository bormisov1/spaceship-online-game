use std::cell::RefCell;
use wasm_bindgen::JsCast;
use web_sys::{CanvasRenderingContext2d, HtmlCanvasElement};
use crate::constants::{WORLD_W, WORLD_H};

thread_local! {
    static FOG_CANVAS: RefCell<Option<HtmlCanvasElement>> = RefCell::new(None);
    static FOG_BUILT: RefCell<bool> = RefCell::new(false);
}

fn build_fog_canvas() {
    let document = web_sys::window().unwrap().document().unwrap();
    let canvas: HtmlCanvasElement = document.create_element("canvas").unwrap().unchecked_into();
    let w = WORLD_W as u32;
    let h = WORLD_H as u32;
    // Use a smaller scale for performance
    let scale = 4;
    let sw = w / scale;
    let sh = h / scale;
    canvas.set_width(sw);
    canvas.set_height(sh);

    let ctx: CanvasRenderingContext2d = canvas
        .get_context("2d").unwrap().unwrap().unchecked_into();

    // Draw fog patches
    let fog_colors = [
        "rgba(10, 20, 40, 0.15)",
        "rgba(20, 10, 30, 0.1)",
        "rgba(5, 15, 25, 0.12)",
    ];

    for _ in 0..12 {
        let x = js_sys::Math::random() * sw as f64;
        let y = js_sys::Math::random() * sh as f64;
        let r = 100.0 + js_sys::Math::random() * 300.0;
        let color_idx = (js_sys::Math::random() * fog_colors.len() as f64) as usize % fog_colors.len();

        if let Ok(gradient) = ctx.create_radial_gradient(x, y, 0.0, x, y, r) {
            let _ = gradient.add_color_stop(0.0_f32, fog_colors[color_idx]);
            let _ = gradient.add_color_stop(1.0_f32, "transparent");
            ctx.set_fill_style_canvas_gradient(&gradient);
            ctx.fill_rect(x - r, y - r, r * 2.0, r * 2.0);
        }
    }

    FOG_CANVAS.with(|fc| *fc.borrow_mut() = Some(canvas));
    FOG_BUILT.with(|fb| *fb.borrow_mut() = true);
}

pub fn render_fog(ctx: &CanvasRenderingContext2d, offset_x: f64, offset_y: f64, _vw: f64, _vh: f64) {
    let built = FOG_BUILT.with(|fb| *fb.borrow());
    if !built {
        build_fog_canvas();
    }

    FOG_CANVAS.with(|fc| {
        if let Some(fog) = fc.borrow().as_ref() {
            // Draw the fog canvas stretched to world coords, offset by camera
            let _ = ctx.draw_image_with_html_canvas_element_and_dw_and_dh(
                fog, -offset_x, -offset_y, WORLD_W, WORLD_H,
            );
        }
    });
}
