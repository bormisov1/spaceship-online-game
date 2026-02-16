use std::cell::RefCell;
use web_sys::{CanvasRenderingContext2d, HtmlImageElement};
use crate::constants::SHIP_SIZE;

const SHIP_NAMES: [&str; 6] = ["rebel-ship-1", "rebel-ship-2", "rebel-ship-3", "star-destroyer-1", "tie-1", "tie-2"];

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

// Ship type 3 (Star Destroyer) renders 5x larger
const SHIP_SCALE: [f64; 6] = [1.0, 1.0, 1.0, 5.0, 1.0, 1.0];

// Per-ship sprite rotation offset: rebel/TIE sprites face UP (+π/2), Star Destroyer faces LEFT (+π)
const SHIP_ROT_OFFSET: [f64; 6] = [
    std::f64::consts::FRAC_PI_2, // rebel 1 (faces up)
    std::f64::consts::FRAC_PI_2, // rebel 2 (faces up)
    std::f64::consts::FRAC_PI_2, // rebel 3 (faces up)
    std::f64::consts::PI,        // Star Destroyer (faces left)
    std::f64::consts::FRAC_PI_2, // TIE 1 (faces up)
    std::f64::consts::FRAC_PI_2, // TIE 2 (faces up)
];

pub fn draw_ship(ctx: &CanvasRenderingContext2d, x: f64, y: f64, rotation: f64, ship_type: i32) {
    SHIP_IMAGES.with(|si| {
        let images = si.borrow();
        let idx = (ship_type as usize).min(images.len().saturating_sub(1));
        if idx >= images.len() { return; }
        let img = &images[idx];

        if img.natural_width() == 0 { return; } // Not loaded yet

        let scale = SHIP_SCALE.get(idx).copied().unwrap_or(1.0);
        let size = SHIP_SIZE * scale;
        let half = size / 2.0;

        ctx.save();
        ctx.translate(x, y).unwrap_or(());
        let rot_offset = SHIP_ROT_OFFSET.get(idx).copied().unwrap_or(std::f64::consts::FRAC_PI_2);
        ctx.rotate(rotation + rot_offset).unwrap_or(());

        let _ = ctx.draw_image_with_html_image_element_and_dw_and_dh(
            img, -half, -half, size, size,
        );

        ctx.restore();
    });
}
