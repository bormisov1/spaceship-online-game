use std::cell::RefCell;
use wasm_bindgen::JsCast;
use web_sys::{CanvasRenderingContext2d, HtmlCanvasElement};
use crate::state::SharedState;

thread_local! {
    static STAR_LAYERS: RefCell<Vec<HtmlCanvasElement>> = RefCell::new(Vec::new());
    static NEBULA_CANVAS: RefCell<Option<HtmlCanvasElement>> = RefCell::new(None);
    static CACHED_W: RefCell<f64> = RefCell::new(0.0);
    static CACHED_H: RefCell<f64> = RefCell::new(0.0);
    static STAR_DATA: RefCell<Vec<StarInfo>> = RefCell::new(Vec::new());
    static IS_MOBILE: RefCell<bool> = RefCell::new(false);
}

const LAYER_COUNTS: [usize; 3] = [120, 100, 80];
const LAYER_SIZES: [(f64, f64); 3] = [(0.5, 1.5), (1.0, 2.5), (2.0, 3.5)];
const LAYER_ALPHAS: [(f64, f64); 3] = [(0.3, 0.6), (0.4, 0.7), (0.6, 1.0)];
const LAYER_FACTORS: [f64; 3] = [0.02, 0.05, 0.10];
const NEBULA_FACTOR: f64 = 0.03;
const NEBULA_COUNT: usize = 8;

struct StarInfo {
    x: f64,
    y: f64,
    size: f64,
    layer: usize,
}

fn rand() -> f64 {
    js_sys::Math::random()
}

fn build_offscreen_canvases(w: f64, h: f64) {
    let document = web_sys::window().unwrap().document().unwrap();
    let mut layers = Vec::new();
    let mut star_data = Vec::new();
    let size_scale = IS_MOBILE.with(|m| if *m.borrow() { 1.0 / 3.0 } else { 1.0 });

    for layer in 0..3 {
        let canvas: HtmlCanvasElement = document.create_element("canvas").unwrap().unchecked_into();
        canvas.set_width(w as u32);
        canvas.set_height(h as u32);
        let ctx: CanvasRenderingContext2d = canvas
            .get_context("2d").unwrap().unwrap().unchecked_into();

        let count = LAYER_COUNTS[layer];
        let (min_size, max_size) = (LAYER_SIZES[layer].0 * size_scale, LAYER_SIZES[layer].1 * size_scale);
        let (min_alpha, max_alpha) = LAYER_ALPHAS[layer];

        for _ in 0..count {
            let x = rand() * w;
            let y = rand() * h;
            let size = min_size + rand() * (max_size - min_size);
            let alpha = min_alpha + rand() * (max_alpha - min_alpha);

            let tint = rand();
            let (r, g, b) = if tint < 0.1 {
                (200u8, 220u8, 255u8)
            } else if tint < 0.15 {
                (255, 220, 200)
            } else {
                (255, 255, 255)
            };

            ctx.set_global_alpha(alpha);
            ctx.set_fill_style_str(&format!("rgb({},{},{})", r, g, b));
            ctx.begin_path();
            let _ = ctx.arc(x, y, size, 0.0, std::f64::consts::PI * 2.0);
            ctx.fill();

            star_data.push(StarInfo { x, y, size, layer });
        }
        ctx.set_global_alpha(1.0);
        layers.push(canvas);
    }

    // Nebula canvas
    let nebula: HtmlCanvasElement = document.create_element("canvas").unwrap().unchecked_into();
    let nw = w * 2.0;
    let nh = h * 2.0;
    nebula.set_width(nw as u32);
    nebula.set_height(nh as u32);
    let nctx: CanvasRenderingContext2d = nebula
        .get_context("2d").unwrap().unwrap().unchecked_into();

    let nebula_colors = [
        "rgba(30, 0, 60, 0.03)",
        "rgba(0, 20, 60, 0.03)",
        "rgba(60, 0, 30, 0.02)",
        "rgba(0, 40, 40, 0.02)",
    ];

    for _ in 0..NEBULA_COUNT {
        let x = rand() * nw;
        let y = rand() * nh;
        let r = 200.0 + rand() * 400.0;
        let color = nebula_colors[(rand() * nebula_colors.len() as f64) as usize % nebula_colors.len()];

        if let Ok(gradient) = nctx.create_radial_gradient(x, y, 0.0, x, y, r) {
            let _ = gradient.add_color_stop(0.0_f32, color);
            let _ = gradient.add_color_stop(1.0_f32, "transparent");
            nctx.set_fill_style_canvas_gradient(&gradient);
            nctx.fill_rect(x - r, y - r, r * 2.0, r * 2.0);
        }
    }

    STAR_LAYERS.with(|sl| *sl.borrow_mut() = layers);
    NEBULA_CANVAS.with(|nc| *nc.borrow_mut() = Some(nebula));
    STAR_DATA.with(|sd| *sd.borrow_mut() = star_data);
    CACHED_W.with(|cw| *cw.borrow_mut() = w);
    CACHED_H.with(|ch| *ch.borrow_mut() = h);
}

pub fn init_starfield(state: &SharedState) {
    let s = state.borrow();
    IS_MOBILE.with(|m| *m.borrow_mut() = s.is_mobile);
    if s.screen_w > 0.0 && s.screen_h > 0.0 {
        build_offscreen_canvases(s.screen_w, s.screen_h);
    }
}

pub fn render_starfield(ctx: &CanvasRenderingContext2d, cx: f64, cy: f64, w: f64, h: f64, hyperspace_t: f64, player_rotation: f64) {
    let cached_w = CACHED_W.with(|cw| *cw.borrow());
    let cached_h = CACHED_H.with(|ch| *ch.borrow());

    if w != cached_w || h != cached_h {
        build_offscreen_canvases(w, h);
    }

    ctx.set_fill_style_str("#0a0a1a");
    ctx.fill_rect(0.0, 0.0, w, h);

    // Nebula (always rendered)
    NEBULA_CANVAS.with(|nc| {
        if let Some(nebula) = nc.borrow().as_ref() {
            let nebula_off_x = cx * NEBULA_FACTOR + (cx - w / 2.0);
            let nebula_off_y = cy * NEBULA_FACTOR + (cy - h / 2.0);
            let _ = ctx.draw_image_with_html_canvas_element(nebula, -nebula_off_x, -nebula_off_y);
        }
    });

    if hyperspace_t < 0.01 {
        // Normal mode: use pre-rendered canvases (fast path)
        STAR_LAYERS.with(|sl| {
            let layers = sl.borrow();
            for (layer, canvas) in layers.iter().enumerate() {
                let factor = LAYER_FACTORS[layer];
                let ox = ((cx * factor) % w + w) % w;
                let oy = ((cy * factor) % h + h) % h;

                let _ = ctx.draw_image_with_html_canvas_element(canvas, -ox, -oy);
                let _ = ctx.draw_image_with_html_canvas_element(canvas, w - ox, -oy);
                let _ = ctx.draw_image_with_html_canvas_element(canvas, -ox, h - oy);
                let _ = ctx.draw_image_with_html_canvas_element(canvas, w - ox, h - oy);
            }
        });
    } else {
        // Hyperspace mode: batch streaks by layer to minimize canvas API calls
        let trail_nx = -(player_rotation.cos());
        let trail_ny = -(player_rotation.sin());

        STAR_DATA.with(|sd| {
            let stars = sd.borrow();

            // Batch by layer: one beginPath/stroke per layer for streaks, one for dots
            for layer in 0..3 {
                let factor = LAYER_FACTORS[layer];
                let ox = ((cx * factor) % w + w) % w;
                let oy = ((cy * factor) % h + h) % h;

                // Most stars are white (255,255,255), batch them together
                // Set a representative style for the layer
                ctx.set_stroke_style_str("rgb(255,255,255)");
                ctx.set_global_alpha((LAYER_ALPHAS[layer].1 * (1.0 + hyperspace_t * 0.3)).min(1.0));
                ctx.set_line_width(LAYER_SIZES[layer].1 * 0.7);

                // Batch all streak lines in this layer
                ctx.begin_path();
                for star in stars.iter().filter(|s| s.layer == layer) {
                    let sx = ((star.x - ox) % w + w) % w;
                    let sy = ((star.y - oy) % h + h) % h;
                    let streak = hyperspace_t * 12.5;
                    let x2 = sx + trail_nx * streak;
                    let y2 = sy + trail_ny * streak;
                    ctx.move_to(sx, sy);
                    ctx.line_to(x2, y2);
                }
                ctx.stroke();

                // Batch dots as fill_rect (tiny squares, visually identical to arcs)
                ctx.set_fill_style_str("rgb(255,255,255)");
                for star in stars.iter().filter(|s| s.layer == layer) {
                    let sx = ((star.x - ox) % w + w) % w;
                    let sy = ((star.y - oy) % h + h) % h;
                    let dot_size = star.size * (1.0 - hyperspace_t * 0.3).max(0.3);
                    ctx.fill_rect(sx - dot_size, sy - dot_size, dot_size * 2.0, dot_size * 2.0);
                }
            }
        });

        ctx.set_global_alpha(1.0);
    }
}
