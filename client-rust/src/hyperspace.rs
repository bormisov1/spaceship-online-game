use std::cell::RefCell;
use web_sys::CanvasRenderingContext2d;

struct HyperStar {
    angle: f64,
    dist: f64,
    speed: f64,
    brightness: f64,
}

const NUM_STARS: usize = 300;

thread_local! {
    static STARS: RefCell<Vec<HyperStar>> = RefCell::new(Vec::new());
    static INITIALIZED: RefCell<bool> = RefCell::new(false);
}

fn new_star(random_dist: bool) -> HyperStar {
    HyperStar {
        angle: js_sys::Math::random() * std::f64::consts::PI * 2.0,
        dist: if random_dist {
            js_sys::Math::random() * 0.9 + 0.05
        } else {
            js_sys::Math::random() * 0.03
        },
        speed: 0.3 + js_sys::Math::random() * 0.8,
        brightness: 0.4 + js_sys::Math::random() * 0.6,
    }
}

pub fn render_hyperspace(ctx: &CanvasRenderingContext2d, w: f64, h: f64, dt: f64) {
    INITIALIZED.with(|init| {
        if !*init.borrow() {
            STARS.with(|stars| {
                let mut s = stars.borrow_mut();
                s.clear();
                for _ in 0..NUM_STARS {
                    s.push(new_star(true));
                }
            });
            *init.borrow_mut() = true;
        }
    });

    // Clear with dark background
    ctx.set_fill_style(&wasm_bindgen::JsValue::from_str("#0a0a1a"));
    ctx.fill_rect(0.0, 0.0, w, h);

    // Subtle radial gradient overlay for depth
    if let Ok(grad) = ctx.create_radial_gradient(w / 2.0, h / 2.0, 0.0, w / 2.0, h / 2.0, w.max(h) * 0.6) {
        let _ = grad.add_color_stop(0.0, "rgba(20, 20, 50, 0.3)");
        let _ = grad.add_color_stop(1.0, "rgba(10, 10, 26, 0)");
        ctx.set_fill_style(&grad);
        ctx.fill_rect(0.0, 0.0, w, h);
    }

    let cx = w / 2.0;
    let cy = h / 2.0;
    let max_dist = (cx * cx + cy * cy).sqrt();

    STARS.with(|stars| {
        let mut s = stars.borrow_mut();
        for star in s.iter_mut() {
            // Update: accelerate as stars get further from center
            let accel = 1.0 + star.dist * 3.0;
            star.dist += star.speed * accel * dt;

            // Respawn at center if off screen
            if star.dist > 1.3 {
                *star = new_star(false);
                continue;
            }

            let d = star.dist * max_dist;
            let x = cx + star.angle.cos() * d;
            let y = cy + star.angle.sin() * d;

            // Trail length grows with distance (short lines, not dots)
            let trail = (star.dist * star.dist * 60.0 + 2.0).min(50.0);
            let x2 = x - star.angle.cos() * trail;
            let y2 = y - star.angle.sin() * trail;

            // Alpha: fade in from center, full brightness further out
            let alpha = (star.dist * 3.0).min(1.0) * star.brightness;
            // Width: thin near center, slightly thicker at edges
            let width = 0.5 + star.dist * 1.5;

            // Blueish-white color with varying warmth
            let r = 180 + (star.brightness * 75.0) as u32;
            let g = 190 + (star.brightness * 65.0) as u32;
            let b = 255;
            ctx.set_stroke_style(&wasm_bindgen::JsValue::from_str(
                &format!("rgba({}, {}, {}, {})", r.min(255), g.min(255), b, alpha),
            ));
            ctx.set_line_width(width);
            ctx.begin_path();
            ctx.move_to(x, y);
            ctx.line_to(x2, y2);
            ctx.stroke();
        }
    });
}
