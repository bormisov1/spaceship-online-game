use std::cell::RefCell;
use std::collections::HashMap;
use web_sys::CanvasRenderingContext2d;
use crate::state::{SharedState, Phase, GameMode};
use crate::constants::{SHIP_COLORS, WORLD_W, WORLD_H, TEAM_RED_COLOR, TEAM_BLUE_COLOR};

thread_local! {
    static TEXT_WIDTH_CACHE: RefCell<HashMap<String, f64>> = RefCell::new(HashMap::new());
    static CACHED_FONT_SIZE: RefCell<i32> = RefCell::new(0);
    /// Cached sorted scoreboard: (tick, sorted player list)
    static SCOREBOARD_CACHE: RefCell<(u64, Vec<crate::protocol::PlayerState>)> = RefCell::new((0, Vec::new()));
}

fn cached_measure_text(ctx: &CanvasRenderingContext2d, text: &str, font_size: i32) -> f64 {
    // Invalidate cache when font size changes
    CACHED_FONT_SIZE.with(|fs| {
        let mut fs = fs.borrow_mut();
        if *fs != font_size {
            TEXT_WIDTH_CACHE.with(|c| c.borrow_mut().clear());
            *fs = font_size;
        }
    });

    TEXT_WIDTH_CACHE.with(|cache| {
        let mut cache = cache.borrow_mut();
        if let Some(&width) = cache.get(text) {
            return width;
        }
        let width = ctx.measure_text(text).map(|m| m.width()).unwrap_or(0.0);
        cache.insert(text.to_string(), width);
        width
    })
}

pub fn render_hud(ctx: &CanvasRenderingContext2d, state: &SharedState) {
    let s = state.borrow();
    let screen_w = s.screen_w;
    let screen_h = s.screen_h;

    // Health bar
    if let Some(my_id) = &s.my_id {
        if let Some(me) = s.players.get(my_id) {
            if me.a {
                let min_dim = screen_w.min(screen_h);
                let bar_w = (min_dim * 0.28).max(120.0).min(200.0);
                draw_health_bar(ctx, screen_w / 2.0, screen_h - 40.0, bar_w, 16.0, me.hp, me.mhp);
            }
        }
    }

    // Minimap
    draw_minimap(ctx, &s, screen_w, screen_h);

    // Kill feed
    draw_kill_feed(ctx, &s, screen_w, screen_h);

    // Scoreboard
    draw_scoreboard(ctx, &s, screen_w, screen_h);

    // Match timer (top center)
    if s.match_phase == 2 && s.match_time_left > 0.0 {
        draw_match_timer(ctx, screen_w, s.match_time_left);
    }

    // Team scores (below timer, for team modes)
    if matches!(s.game_mode, GameMode::TDM | GameMode::CTF) && s.match_phase >= 2 {
        draw_team_scores(ctx, screen_w, s.team_red_score, s.team_blue_score);
    }

    // Countdown overlay
    if s.phase == Phase::Countdown {
        draw_countdown(ctx, screen_w, screen_h, s.countdown_time);
    }

    // Result screen
    if s.phase == Phase::Result {
        if let Some((winner, ref players, duration)) = s.match_result {
            draw_result_screen(ctx, screen_w, screen_h, winner, players, duration, s.game_mode);
        }
    }

    // Death screen
    if s.phase == Phase::Dead {
        if let Some(ref death_info) = s.death_info {
            draw_death_screen(ctx, screen_w, screen_h, &death_info.killer_name);
        }
    }

    // Crosshair
    if s.phase == Phase::Playing && !s.is_mobile && !s.controller_attached {
        draw_crosshair(ctx, s.mouse_x, s.mouse_y);
    }

    // Mobile controls overlay (joystick only, no visual markers for fire/boost zones)
    if s.is_mobile && (s.phase == Phase::Playing || s.phase == Phase::Dead) {
        if let Some(ref tj) = s.touch_joystick {
            draw_mobile_joystick(ctx, tj.start_x, tj.start_y, tj.current_x, tj.current_y);
        }
    }

    // Connection status
    if !s.connected {
        ctx.set_fill_style_str("#ff4444");
        ctx.set_font("16px monospace");
        ctx.set_text_align("center");
        let _ = ctx.fill_text("DISCONNECTED - Reconnecting...", screen_w / 2.0, 30.0);
    }
}

fn draw_health_bar(ctx: &CanvasRenderingContext2d, x: f64, y: f64, w: f64, h: f64, hp: i32, max_hp: i32) {
    let ratio = hp as f64 / max_hp as f64;

    ctx.set_fill_style_str("rgba(0, 0, 0, 0.5)");
    ctx.fill_rect(x - w / 2.0 - 2.0, y - 2.0, w + 4.0, h + 4.0);

    let color = if ratio > 0.6 { "#44ff44" } else if ratio > 0.3 { "#ffaa00" } else { "#ff4444" };
    ctx.set_fill_style_str(color);
    ctx.fill_rect(x - w / 2.0, y, w * ratio, h);

    ctx.set_stroke_style_str("#ffffff44");
    ctx.set_line_width(1.0);
    ctx.stroke_rect(x - w / 2.0, y, w, h);

    ctx.set_fill_style_str("#ffffff");
    ctx.set_font("bold 12px monospace");
    ctx.set_text_align("center");
    let _ = ctx.fill_text(&format!("{}/{}", hp, max_hp), x, y + h - 3.0);
}

fn draw_minimap(ctx: &CanvasRenderingContext2d, s: &crate::state::GameState, screen_w: f64, screen_h: f64) {
    let min_dim = screen_w.min(screen_h);
    let size = (min_dim * 0.22).max(80.0).min(180.0);
    let margin = 10.0;
    let x = screen_w - size - margin;
    let y = margin;

    ctx.set_fill_style_str("rgba(0, 40, 0, 0.5)");
    ctx.fill_rect(x, y, size, size);

    ctx.set_stroke_style_str("#00ff00");
    ctx.set_line_width(1.0);
    ctx.stroke_rect(x, y, size, size);

    // Players
    for p in s.players.values() {
        if !p.a { continue; }
        let is_me = s.my_id.as_ref() == Some(&p.id);
        let idx = (p.s as usize).min(SHIP_COLORS.len() - 1);
        let dot_x = x + (p.x / WORLD_W) * size;
        let dot_y = y + (p.y / WORLD_H) * size;
        let radius = if is_me { 3.0 } else { 2.0 };

        ctx.begin_path();
        let _ = ctx.arc(dot_x, dot_y, radius, 0.0, std::f64::consts::PI * 2.0);
        ctx.set_fill_style_str(if is_me { "#ffffff" } else { SHIP_COLORS[idx].main });
        ctx.fill();
    }

    // Mobs
    for mob in s.mobs.values() {
        if !mob.a { continue; }
        let dot_x = x + (mob.x / WORLD_W) * size;
        let dot_y = y + (mob.y / WORLD_H) * size;
        ctx.begin_path();
        let _ = ctx.arc(dot_x, dot_y, 2.0, 0.0, std::f64::consts::PI * 2.0);
        ctx.set_fill_style_str("#ffff44");
        ctx.fill();
    }

    // Asteroids
    for ast in s.asteroids.values() {
        let dot_x = x + (ast.x / WORLD_W) * size;
        let dot_y = y + (ast.y / WORLD_H) * size;
        ctx.begin_path();
        let _ = ctx.arc(dot_x, dot_y, 3.0, 0.0, std::f64::consts::PI * 2.0);
        ctx.set_fill_style_str("#aa7744");
        ctx.fill();
    }

    // Pickups
    for pk in s.pickups.values() {
        let dot_x = x + (pk.x / WORLD_W) * size;
        let dot_y = y + (pk.y / WORLD_H) * size;
        ctx.begin_path();
        let _ = ctx.arc(dot_x, dot_y, 2.5, 0.0, std::f64::consts::PI * 2.0);
        ctx.set_fill_style_str("#44ff88");
        ctx.fill();
    }
}

fn draw_kill_feed(ctx: &CanvasRenderingContext2d, s: &crate::state::GameState, screen_w: f64, screen_h: f64) {
    let now = web_sys::window().unwrap().performance().unwrap().now();
    let x = screen_w - 20.0;
    let min_dim = screen_w.min(screen_h);
    let map_size = (min_dim * 0.22).max(80.0).min(180.0);
    let mut y = map_size + 30.0;

    ctx.set_text_align("right");
    let font_size = (min_dim * 0.018).max(10.0).min(13.0) as i32;
    ctx.set_font(&format!("{}px monospace", font_size));

    for kill in s.kill_feed.iter().rev() {
        let age = (now - kill.time) / 1000.0;
        if age > 8.0 { continue; }

        let alpha = if age > 6.0 { (8.0 - age) / 2.0 } else { 1.0 };
        ctx.set_global_alpha(alpha);

        // Measure text segments right-to-left (cached)
        let victim_w = cached_measure_text(ctx, &kill.victim, font_size);
        let killed_text = " killed ";
        let killed_w = cached_measure_text(ctx, killed_text, font_size);

        // Draw killer name (orange)
        ctx.set_fill_style_str("#ffaa00");
        let _ = ctx.fill_text(&kill.killer, x - victim_w - killed_w, y);
        // Draw " killed " (white)
        ctx.set_fill_style_str("#ffffff");
        let _ = ctx.fill_text(killed_text, x - victim_w, y);
        // Draw victim name (red)
        ctx.set_fill_style_str("#ff4444");
        let _ = ctx.fill_text(&kill.victim, x, y);

        y += 20.0;
    }
    ctx.set_global_alpha(1.0);
}

fn draw_scoreboard(ctx: &CanvasRenderingContext2d, s: &crate::state::GameState, screen_w: f64, screen_h: f64) {
    let min_dim = screen_w.min(screen_h);
    let scale = (min_dim / 800.0).max(0.7).min(1.0);
    let font_size = (13.0 * scale) as i32;
    let header_size = (12.0 * scale) as i32;
    let line_h = (18.0 * scale) as i32;
    let panel_w = 180.0 * scale;
    let score_x = 150.0 * scale;
    let max_players = if min_dim < 500.0 { 5 } else { 8 };

    // Re-sort only when tick changes (new server state arrived)
    SCOREBOARD_CACHE.with(|cache| {
        let mut cache = cache.borrow_mut();
        if cache.0 != s.tick {
            cache.1.clear();
            cache.1.extend(s.players.values().cloned());
            cache.1.sort_by(|a, b| b.sc.cmp(&a.sc).then_with(|| a.id.cmp(&b.id)));
            cache.1.truncate(max_players);
            cache.0 = s.tick;
        }

        ctx.set_text_align("left");
        ctx.set_font(&format!("{}px monospace", font_size));

        let x = 15.0;
        let mut y = 60.0 * scale;

        ctx.set_fill_style_str("rgba(0, 0, 0, 0.4)");
        ctx.fill_rect(x - 5.0, y - line_h as f64, panel_w, (cache.1.len() as f64 * (line_h as f64 + 2.0)) + line_h as f64 + 6.0);

        ctx.set_fill_style_str("#ffffff88");
        ctx.set_font(&format!("bold {}px monospace", header_size));
        let _ = ctx.fill_text("SCOREBOARD", x, y - 2.0);
        y += line_h as f64;

        ctx.set_font(&format!("{}px monospace", font_size));
        let max_name_len = if min_dim < 500.0 { 8 } else { 12 };

        for p in &cache.1 {
            let is_me = s.my_id.as_ref() == Some(&p.id);
            let idx = (p.s as usize).min(SHIP_COLORS.len() - 1);

            ctx.set_fill_style_str(if is_me { "#ffffff" } else { "#aaaaaa" });
            let name = if p.n.len() > max_name_len {
                format!("{}..", &p.n[..max_name_len])
            } else {
                p.n.clone()
            };
            let _ = ctx.fill_text(&name, x, y);

            ctx.set_fill_style_str(SHIP_COLORS[idx].main);
            let _ = ctx.fill_text(&p.sc.to_string(), x + score_x, y);
            y += line_h as f64;
        }
    });
}

fn draw_death_screen(ctx: &CanvasRenderingContext2d, screen_w: f64, screen_h: f64, killer_name: &str) {
    ctx.set_fill_style_str("rgba(0, 0, 0, 0.5)");
    ctx.fill_rect(0.0, 0.0, screen_w, screen_h);

    ctx.set_text_align("center");
    ctx.set_fill_style_str("#ff4444");
    ctx.set_font("bold 36px monospace");
    let _ = ctx.fill_text("DESTROYED", screen_w / 2.0, screen_h / 2.0 - 30.0);

    ctx.set_fill_style_str("#ffffff");
    ctx.set_font("20px monospace");
    let _ = ctx.fill_text(&format!("by {}", killer_name), screen_w / 2.0, screen_h / 2.0 + 10.0);

    ctx.set_fill_style_str("#aaaaaa");
    ctx.set_font("16px monospace");
    let _ = ctx.fill_text("Respawning...", screen_w / 2.0, screen_h / 2.0 + 50.0);
}

fn draw_crosshair(ctx: &CanvasRenderingContext2d, mx: f64, my: f64) {
    let size = 12.0;
    ctx.set_stroke_style_str("rgba(255, 255, 255, 0.6)");
    ctx.set_line_width(1.5);

    ctx.begin_path();
    ctx.move_to(mx - size, my);
    ctx.line_to(mx - size / 3.0, my);
    ctx.move_to(mx + size / 3.0, my);
    ctx.line_to(mx + size, my);
    ctx.move_to(mx, my - size);
    ctx.line_to(mx, my - size / 3.0);
    ctx.move_to(mx, my + size / 3.0);
    ctx.line_to(mx, my + size);
    ctx.stroke();

    ctx.begin_path();
    let _ = ctx.arc(mx, my, 2.0, 0.0, std::f64::consts::PI * 2.0);
    ctx.set_fill_style_str("rgba(255, 255, 255, 0.6)");
    ctx.fill();
}

fn draw_mobile_joystick(ctx: &CanvasRenderingContext2d, start_x: f64, start_y: f64, current_x: f64, current_y: f64) {
    let max_radius = 60.0;

    // Base circle
    ctx.begin_path();
    let _ = ctx.arc(start_x, start_y, max_radius, 0.0, std::f64::consts::PI * 2.0);
    ctx.set_stroke_style_str("rgba(255, 255, 255, 0.2)");
    ctx.set_line_width(2.0);
    ctx.stroke();

    // Inner dead zone
    ctx.begin_path();
    let _ = ctx.arc(start_x, start_y, 12.0, 0.0, std::f64::consts::PI * 2.0);
    ctx.set_stroke_style_str("rgba(255, 255, 255, 0.1)");
    ctx.set_line_width(1.0);
    ctx.stroke();

    // Thumb position
    let mut dx = current_x - start_x;
    let mut dy = current_y - start_y;
    let dist = (dx * dx + dy * dy).sqrt();
    if dist > max_radius {
        dx = (dx / dist) * max_radius;
        dy = (dy / dist) * max_radius;
    }

    ctx.begin_path();
    let _ = ctx.arc(start_x + dx, start_y + dy, 18.0, 0.0, std::f64::consts::PI * 2.0);
    ctx.set_fill_style_str("rgba(255, 255, 255, 0.25)");
    ctx.fill();
    ctx.set_stroke_style_str("rgba(255, 255, 255, 0.4)");
    ctx.set_line_width(1.5);
    ctx.stroke();
}

pub fn draw_player_health_bar(ctx: &CanvasRenderingContext2d, x: f64, y: f64, hp: i32, max_hp: i32, name: &str, is_me: bool) {
    let bar_w = 40.0;
    let bar_h = 4.0;
    let bar_y = y - 30.0;

    if !is_me {
        ctx.set_fill_style_str("#ffffff99");
        ctx.set_font("11px monospace");
        ctx.set_text_align("center");
        let _ = ctx.fill_text(name, x, bar_y - 8.0);
    }

    let ratio = hp as f64 / max_hp as f64;
    ctx.set_fill_style_str("rgba(0,0,0,0.5)");
    ctx.fill_rect(x - bar_w / 2.0, bar_y, bar_w, bar_h);

    let color = if ratio > 0.6 { "#44ff44" } else if ratio > 0.3 { "#ffaa00" } else { "#ff4444" };
    ctx.set_fill_style_str(color);
    ctx.fill_rect(x - bar_w / 2.0, bar_y, bar_w * ratio, bar_h);
}

fn draw_match_timer(ctx: &CanvasRenderingContext2d, screen_w: f64, time_left: f64) {
    let minutes = (time_left / 60.0) as i32;
    let seconds = (time_left % 60.0) as i32;
    let text = format!("{:02}:{:02}", minutes, seconds);

    ctx.set_text_align("center");
    ctx.set_fill_style_str("rgba(0, 0, 0, 0.5)");
    ctx.fill_rect(screen_w / 2.0 - 40.0, 8.0, 80.0, 28.0);

    ctx.set_fill_style_str(if time_left < 30.0 { "#ff4444" } else { "#ffffff" });
    ctx.set_font("bold 18px monospace");
    let _ = ctx.fill_text(&text, screen_w / 2.0, 28.0);
}

fn draw_team_scores(ctx: &CanvasRenderingContext2d, screen_w: f64, red: i32, blue: i32) {
    let cx = screen_w / 2.0;

    ctx.set_fill_style_str("rgba(0, 0, 0, 0.4)");
    ctx.fill_rect(cx - 80.0, 38.0, 160.0, 22.0);

    ctx.set_font("bold 14px monospace");
    ctx.set_text_align("right");
    ctx.set_fill_style_str(TEAM_RED_COLOR);
    let _ = ctx.fill_text(&format!("RED {}", red), cx - 8.0, 54.0);

    ctx.set_text_align("left");
    ctx.set_fill_style_str(TEAM_BLUE_COLOR);
    let _ = ctx.fill_text(&format!("{} BLUE", blue), cx + 8.0, 54.0);

    ctx.set_text_align("center");
    ctx.set_fill_style_str("#ffffff44");
    let _ = ctx.fill_text("-", cx, 54.0);
}

fn draw_countdown(ctx: &CanvasRenderingContext2d, screen_w: f64, screen_h: f64, countdown: f64) {
    ctx.set_fill_style_str("rgba(0, 0, 0, 0.4)");
    ctx.fill_rect(0.0, 0.0, screen_w, screen_h);

    ctx.set_text_align("center");

    let num = countdown.ceil() as i32;
    let text = if num <= 0 { "FIGHT!".to_string() } else { num.to_string() };
    let frac = countdown - countdown.floor();
    let scale = 1.0 + frac * 0.3;
    let font_size = (72.0 * scale) as i32;

    ctx.set_font(&format!("bold {}px monospace", font_size));
    ctx.set_fill_style_str(if num <= 0 { "#44ff44" } else { "#ffcc00" });
    let _ = ctx.fill_text(&text, screen_w / 2.0, screen_h / 2.0 + 20.0);
}

fn draw_result_screen(
    ctx: &CanvasRenderingContext2d,
    screen_w: f64,
    screen_h: f64,
    winner_team: i32,
    players: &[crate::protocol::PlayerMatchResult],
    duration: f64,
    _mode: GameMode,
) {
    ctx.set_fill_style_str("rgba(0, 0, 0, 0.7)");
    ctx.fill_rect(0.0, 0.0, screen_w, screen_h);

    ctx.set_text_align("center");

    // Winner text
    let winner_text = match winner_team {
        1 => "RED TEAM WINS!",
        2 => "BLUE TEAM WINS!",
        _ => "MATCH OVER",
    };
    let winner_color = match winner_team {
        1 => TEAM_RED_COLOR,
        2 => TEAM_BLUE_COLOR,
        _ => "#ffcc00",
    };
    ctx.set_font("bold 36px monospace");
    ctx.set_fill_style_str(winner_color);
    let _ = ctx.fill_text(winner_text, screen_w / 2.0, screen_h * 0.2);

    // Duration
    let dur_min = (duration / 60.0) as i32;
    let dur_sec = (duration % 60.0) as i32;
    ctx.set_font("14px monospace");
    ctx.set_fill_style_str("#aaaaaa");
    let _ = ctx.fill_text(&format!("Duration: {:02}:{:02}", dur_min, dur_sec), screen_w / 2.0, screen_h * 0.2 + 30.0);

    // Player table
    ctx.set_font("bold 12px monospace");
    ctx.set_fill_style_str("#ffffff88");
    let table_y = screen_h * 0.32;
    let col_name = screen_w / 2.0 - 120.0;
    let col_k = screen_w / 2.0 + 30.0;
    let col_d = screen_w / 2.0 + 70.0;
    let col_a = screen_w / 2.0 + 110.0;

    ctx.set_text_align("left");
    let _ = ctx.fill_text("PLAYER", col_name, table_y);
    ctx.set_text_align("center");
    let _ = ctx.fill_text("K", col_k, table_y);
    let _ = ctx.fill_text("D", col_d, table_y);
    let _ = ctx.fill_text("A", col_a, table_y);

    ctx.set_font("12px monospace");
    let mut y = table_y + 20.0;
    for p in players {
        let team_color = match p.tm {
            1 => TEAM_RED_COLOR,
            2 => TEAM_BLUE_COLOR,
            _ => "#ffffff",
        };
        let name_display = if p.mvp {
            format!("\u{2605} {}", p.n)
        } else {
            p.n.clone()
        };

        ctx.set_fill_style_str(team_color);
        ctx.set_text_align("left");
        let _ = ctx.fill_text(&name_display, col_name, y);

        ctx.set_text_align("center");
        ctx.set_fill_style_str("#ffffff");
        let _ = ctx.fill_text(&p.k.to_string(), col_k, y);
        let _ = ctx.fill_text(&p.d.to_string(), col_d, y);
        let _ = ctx.fill_text(&p.a.to_string(), col_a, y);

        y += 18.0;
    }

    // Rematch hint
    ctx.set_text_align("center");
    ctx.set_fill_style_str("#aaaaaa");
    ctx.set_font("14px monospace");
    let _ = ctx.fill_text("Returning to lobby...", screen_w / 2.0, screen_h * 0.85);
}

