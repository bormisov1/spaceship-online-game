use std::cell::RefCell;
use wasm_bindgen::JsCast;
use web_sys::{CanvasRenderingContext2d, HtmlImageElement};
use crate::constants::SHIP_SIZE;

const SHIP_NAMES: [&str; 4] = ["Fighter", "Cruiser", "Artillery", "Destroyer"];

thread_local! {
    static SHIP_IMAGES: RefCell<Vec<HtmlImageElement>> = RefCell::new(Vec::new());
    static IMAGES_LOADED: RefCell<bool> = RefCell::new(false);
}

pub fn load_ship_images() {
    let mut images = Vec::new();
    for name in &SHIP_NAMES {
        let img = HtmlImageElement::new().unwrap();
        img.set_src(&format!("assets/ships/{}.png", name));
        images.push(img);
    }
    SHIP_IMAGES.with(|si| *si.borrow_mut() = images);
    IMAGES_LOADED.with(|il| *il.borrow_mut() = true);
}

pub fn draw_ship(ctx: &CanvasRenderingContext2d, x: f64, y: f64, rotation: f64, ship_type: i32) {
    SHIP_IMAGES.with(|si| {
        let images = si.borrow();
        let idx = (ship_type as usize).min(images.len().saturating_sub(1));
        if idx >= images.len() { return; }
        let img = &images[idx];

        if img.natural_width() == 0 { return; } // Not loaded yet

        ctx.save();
        ctx.translate(x, y).unwrap_or(());
        ctx.rotate(rotation).unwrap_or(());

        let half = SHIP_SIZE / 2.0;
        let _ = ctx.draw_image_with_html_image_element_and_dw_and_dh(
            img, -half, -half, SHIP_SIZE, SHIP_SIZE,
        );

        ctx.restore();
    });
}
