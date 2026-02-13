use std::cell::RefCell;
use std::collections::HashMap;
use web_sys::{CanvasRenderingContext2d, HtmlImageElement};
use crate::constants::ASTEROID_RENDER_SIZE;
use crate::protocol::AsteroidState;

thread_local! {
    static ASTEROID_IMG: RefCell<Option<HtmlImageElement>> = RefCell::new(None);
}

pub fn load_asteroid_image() {
    let img = HtmlImageElement::new().unwrap();
    img.set_src("assets/asteroid.png");
    ASTEROID_IMG.with(|ai| *ai.borrow_mut() = Some(img));
}

pub fn render_asteroids(
    ctx: &CanvasRenderingContext2d,
    asteroids: &HashMap<String, AsteroidState>,
    offset_x: f64, offset_y: f64, vw: f64, vh: f64,
) {
    ASTEROID_IMG.with(|ai| {
        let img = ai.borrow();
        let img = match img.as_ref() {
            Some(i) if i.natural_width() > 0 => i,
            _ => return,
        };

        let half = ASTEROID_RENDER_SIZE / 2.0;

        for (_, ast) in asteroids {
            let sx = ast.x - offset_x;
            let sy = ast.y - offset_y;
            if sx < -half - 20.0 || sx > vw + half + 20.0 || sy < -half - 20.0 || sy > vh + half + 20.0 {
                continue;
            }

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
