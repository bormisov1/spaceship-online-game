use std::cell::RefCell;
use web_sys::CanvasRenderingContext2d;
use crate::state::SharedState;

const AIM_ORBIT_R: f64 = 360.0;
const AIM_DETECT_R: f64 = 150.0;
const AIM_FREE_R: f64 = 150.0;
const AIM_LOCK_R: f64 = 20.0;
const AIM_ANIM_SPEED: f64 = 4.0;
const AIM_SPIN_MAX: f64 = 8.0;

thread_local! {
    static AIM_STATE: RefCell<AimState> = RefCell::new(AimState::default());
}

#[derive(Default)]
struct AimState {
    target_id: Option<String>,
    target_x: f64,
    target_y: f64,
    progress: f64,
    spin_angle: f64,
}

struct Enemy {
    id: String,
    x: f64,
    y: f64,
}

pub fn update_and_draw_controller_aim(
    ctx: &CanvasRenderingContext2d,
    state: &SharedState,
    offset_x: f64, offset_y: f64,
    dt: f64,
) {
    let s = state.borrow();
    let my_id = match &s.my_id {
        Some(id) => id.clone(),
        None => return,
    };
    let me = match s.players.get(&my_id) {
        Some(p) if p.a => p,
        _ => return,
    };

    let orbit_wx = me.x + me.r.cos() * AIM_ORBIT_R;
    let orbit_wy = me.y + me.r.sin() * AIM_ORBIT_R;

    // Build enemy list
    let mut enemies = Vec::new();
    for (id, p) in &s.players {
        if *id == my_id || !p.a { continue; }
        enemies.push(Enemy { id: format!("p_{}", id), x: p.x, y: p.y });
    }
    for (id, m) in &s.mobs {
        if !m.a { continue; }
        enemies.push(Enemy { id: format!("m_{}", id), x: m.x, y: m.y });
    }

    drop(s);

    AIM_STATE.with(|aim| {
        let mut aim = aim.borrow_mut();

        // Sticky lock check
        let mut locked = false;
        if let Some(ref target_id) = aim.target_id {
            if let Some(t) = enemies.iter().find(|e| &e.id == target_id) {
                let dx = t.x - orbit_wx;
                let dy = t.y - orbit_wy;
                if dx * dx + dy * dy <= AIM_DETECT_R * AIM_DETECT_R {
                    locked = true;
                    aim.target_x = t.x;
                    aim.target_y = t.y;
                }
            }
        }

        if !locked {
            aim.target_id = None;
            let mut best_dist = AIM_DETECT_R * AIM_DETECT_R;
            for e in &enemies {
                let dx = e.x - orbit_wx;
                let dy = e.y - orbit_wy;
                let d2 = dx * dx + dy * dy;
                if d2 <= best_dist {
                    best_dist = d2;
                    aim.target_id = Some(e.id.clone());
                    aim.target_x = e.x;
                    aim.target_y = e.y;
                    locked = true;
                }
            }
        }

        // Animate progress
        let target_progress = if locked { 1.0 } else { 0.0 };
        if aim.progress < target_progress {
            aim.progress = (aim.progress + AIM_ANIM_SPEED * dt).min(1.0);
        } else if aim.progress > target_progress {
            aim.progress = (aim.progress - AIM_ANIM_SPEED * dt).max(0.0);
        }

        let spin_speed = aim.progress * AIM_SPIN_MAX;
        aim.spin_angle += spin_speed * dt;

        // Screen positions
        let orbit_sx = orbit_wx - offset_x;
        let orbit_sy = orbit_wy - offset_y;
        let target_sx = aim.target_x - offset_x;
        let target_sy = aim.target_y - offset_y;

        let p = aim.progress;
        let cx = orbit_sx + (target_sx - orbit_sx) * p;
        let cy = orbit_sy + (target_sy - orbit_sy) * p;
        let radius = AIM_FREE_R + (AIM_LOCK_R - AIM_FREE_R) * p;

        // Draw dashed circle
        ctx.save();
        ctx.translate(cx, cy).unwrap_or(());
        ctx.rotate(aim.spin_angle).unwrap_or(());

        let alpha = 0.3 + 0.3 * p;
        ctx.set_stroke_style_str(&format!("rgba(255, 255, 255, {})", alpha));
        ctx.set_line_width(1.5);
        ctx.set_line_dash(&js_sys::Array::of2(&8.0.into(), &6.0.into())).unwrap_or(());
        ctx.begin_path();
        let _ = ctx.arc(0.0, 0.0, radius, 0.0, std::f64::consts::PI * 2.0);
        ctx.stroke();
        ctx.set_line_dash(&js_sys::Array::new()).unwrap_or(());

        ctx.restore();
    });
}
