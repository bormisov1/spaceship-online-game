use std::cell::RefCell;
use std::collections::HashMap;
use web_sys::{CanvasRenderingContext2d, HtmlImageElement};
use crate::constants::ASTEROID_RENDER_SIZE;
use crate::protocol::AsteroidState;

const ASTEROID_FILES: [&str; 4] = [
    "assets/asteroid-1.png",
    "assets/asteroid-2.png",
    "assets/asteroid-3.png",
    "assets/asteroid-4.png",
];

thread_local! {
    static ASTEROID_IMGS: RefCell<Vec<HtmlImageElement>> = RefCell::new(Vec::new());
}

pub fn load_asteroid_image() {
    let mut images = Vec::new();
    for src in &ASTEROID_FILES {
        let img = HtmlImageElement::new().unwrap();
        img.set_src(src);
        images.push(img);
    }
    ASTEROID_IMGS.with(|ai| *ai.borrow_mut() = images);
}

fn id_to_variant(id: &str) -> usize {
    let mut h: i32 = 0;
    for b in id.bytes() {
        h = h.wrapping_mul(31).wrapping_add(b as i32);
    }
    ((h % ASTEROID_FILES.len() as i32) + ASTEROID_FILES.len() as i32) as usize % ASTEROID_FILES.len()
}

pub fn render_asteroids(
    ctx: &CanvasRenderingContext2d,
    asteroids: &HashMap<String, AsteroidState>,
    offset_x: f64, offset_y: f64, vw: f64, vh: f64,
) {
    ASTEROID_IMGS.with(|ai| {
        let images = ai.borrow();
        if images.is_empty() { return; }

        let half = ASTEROID_RENDER_SIZE / 2.0;

        for (id, ast) in asteroids {
            let sx = ast.x - offset_x;
            let sy = ast.y - offset_y;
            if sx < -half - 20.0 || sx > vw + half + 20.0 || sy < -half - 20.0 || sy > vh + half + 20.0 {
                continue;
            }

            let variant = id_to_variant(id);
            let img = &images[variant];
            if img.natural_width() == 0 { continue; }

            ctx.save();
            ctx.translate(sx, sy).unwrap_or(());
            ctx.rotate(ast.r).unwrap_or(());
            let _ = ctx.draw_image_with_html_image_element_and_dw_and_dh(
                img, -half, -half, ASTEROID_RENDER_SIZE, ASTEROID_RENDER_SIZE,
            );
            ctx.restore();
        }
    });
}
