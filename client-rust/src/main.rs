mod app;
mod constants;
mod protocol;
mod state;
mod network;
mod lobby;
mod match_lobby;
mod canvas;
mod game_loop;
mod renderer;
mod starfield;
mod ships;
mod effects;
mod projectiles;
mod mobs;
mod asteroids;
mod pickups;
mod fog;
mod hud;
mod input;
mod auto_aim;
mod controller;
mod hyperspace;

fn main() {
    console_error_panic_hook::set_once();
    leptos::mount::mount_to_body(app::App);
}
