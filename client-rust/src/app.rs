use leptos::prelude::*;
use wasm_bindgen::JsCast;
use crate::state::{self, Phase, SharedState};
use crate::network::{Network, SharedNetwork};
use crate::protocol::{SessionInfo, CheckedMsg};
use crate::lobby;
use crate::game_loop;
use crate::input;
use crate::controller;

/// Detect the base path from current URL: "/rust/" if loaded from /rust/*, otherwise "/"
pub fn base_path() -> &'static str {
    thread_local! {
        static BASE: String = {
            let pathname = web_sys::window().unwrap().location().pathname().unwrap_or_default();
            if pathname.starts_with("/rust") { "/rust/".to_string() } else { "/".to_string() }
        };
    }
    BASE.with(|b| if b == "/rust/" { "/rust/" } else { "/" })
}

const UUID_PATTERN: &str = "[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}";

fn extract_uuid_from_path(pathname: &str) -> Option<String> {
    let uuid_re = js_sys::RegExp::new(&format!("^(?:/rust)?/({})$", UUID_PATTERN), "");
    uuid_re.exec(pathname).and_then(|m| m.get(1).as_string())
}

#[component]
pub fn App() -> impl IntoView {
    // Check for controller mode
    let window = web_sys::window().unwrap();
    let location = window.location();
    let search = location.search().unwrap_or_default();
    let params = web_sys::UrlSearchParams::new_with_str(&search).unwrap();
    let control_pid = params.get("c");

    let pathname = location.pathname().unwrap_or_default();
    let uuid_match = extract_uuid_from_path(&pathname);

    // Controller mode
    if let Some(pid) = control_pid {
        if let Some(sid) = &uuid_match {
            return view! { <ControllerMode sid=sid.clone() pid=pid /> }.into_any();
        }
    }

    // Normal game mode
    let game_state = state::new_shared_state();

    // Check URL for session UUID
    if let Some(sid) = uuid_match {
        game_state.borrow_mut().url_session_id = Some(sid);
    }

    let phase_signal = RwSignal::new(Phase::Lobby);
    let sessions_signal = RwSignal::new(Vec::<SessionInfo>::new());
    let checked_signal = RwSignal::new(None::<CheckedMsg>);
    let expired_signal = RwSignal::new(false);
    let auth_signal = RwSignal::new(None::<String>);
    let lobby_signal = RwSignal::new(0u64);

    // Check localStorage for existing auth
    if let Ok(Some(storage)) = web_sys::window().unwrap().local_storage() {
        if let Ok(Some(username)) = storage.get_item("auth_username") {
            if !username.is_empty() {
                auth_signal.set(Some(username.clone()));
                game_state.borrow_mut().auth_username = Some(username);
            }
        }
    }

    let net = Network::new(
        game_state.clone(),
        phase_signal,
        sessions_signal,
        checked_signal,
        expired_signal,
        auth_signal,
        lobby_signal,
    );

    Network::connect(&net);

    // Start input send loop (20Hz)
    let net_clone = net.clone();
    let _input_interval = gloo_timers::callback::Interval::new(1000 / crate::constants::INPUT_RATE, move || {
        Network::send_input(&net_clone);
    });
    // Leak the interval to keep it alive
    std::mem::forget(_input_interval);

    // Start session list refresh (3s) while in lobby
    let net_clone = net.clone();
    let _refresh_interval = gloo_timers::callback::Interval::new(3000, move || {
        let phase = net_clone.borrow().state.borrow().phase.clone();
        if phase == Phase::Lobby {
            Network::list_sessions(&net_clone);
        }
    });
    std::mem::forget(_refresh_interval);

    // Initial session list + leaderboard fetch
    Network::list_sessions(&net);
    Network::send_leaderboard_request(&net);

    view! {
        <GameView
            state=game_state
            net=net
            phase=phase_signal
            sessions=sessions_signal
            checked=checked_signal
            expired=expired_signal
            auth=auth_signal
            lobby=lobby_signal
        />
    }.into_any()
}

#[component]
fn GameView(
    state: SharedState,
    net: SharedNetwork,
    phase: RwSignal<Phase>,
    sessions: RwSignal<Vec<SessionInfo>>,
    checked: RwSignal<Option<CheckedMsg>>,
    expired: RwSignal<bool>,
    auth: RwSignal<Option<String>>,
    lobby: RwSignal<u64>,
) -> impl IntoView {
    let state_clone = send_wrapper::SendWrapper::new(state.clone());
    let net_clone = send_wrapper::SendWrapper::new(net.clone());

    // Setup canvases once mounted
    let state_for_mount = send_wrapper::SendWrapper::new(state.clone());
    let net_for_mount = send_wrapper::SendWrapper::new(net.clone());
    let phase_for_mount = phase;

    Effect::new(move |_| {
        let state = (*state_for_mount).clone();
        let net = (*net_for_mount).clone();
        let _phase = phase_for_mount;

        // Get canvases
        let document = web_sys::window().unwrap().document().unwrap();
        let bg_canvas = document.get_element_by_id("bgCanvas")
            .and_then(|e| e.dyn_into::<web_sys::HtmlCanvasElement>().ok());
        let game_canvas = document.get_element_by_id("gameCanvas")
            .and_then(|e| e.dyn_into::<web_sys::HtmlCanvasElement>().ok());

        if let (Some(_bg), Some(_game)) = (bg_canvas, game_canvas) {
            // Resize
            crate::canvas::resize(&state);

            // Setup resize handler
            crate::canvas::setup_resize_handler(state.clone());

            // Setup input
            input::setup_input(state.clone(), net.clone());

            // Init starfield
            crate::starfield::init_starfield(&state);

            // Start game loop
            game_loop::start_game_loop(state.clone());

            // Handle popstate
            let state_pop = state.clone();
            let net_pop = net.clone();
            let phase_pop = _phase;
            let closure = wasm_bindgen::closure::Closure::wrap(Box::new(move |_: web_sys::Event| {
                let s = state_pop.borrow();
                if matches!(s.phase, Phase::Playing | Phase::Dead | Phase::MatchLobby | Phase::Countdown | Phase::Result) {
                    drop(s);
                    Network::send_leave(&net_pop);
                    let mut s = state_pop.borrow_mut();
                    s.session_id = None;
                    s.my_id = None;
                    s.controller_attached = false;
                    s.phase = Phase::Lobby;
                    phase_pop.set(Phase::Lobby);
                }
            }) as Box<dyn FnMut(web_sys::Event)>);
            let window = web_sys::window().unwrap();
            let _ = window.add_event_listener_with_callback("popstate", closure.as_ref().unchecked_ref());
            closure.forget();
        }
    });

    view! {
        <canvas id="bgCanvas"></canvas>
        <canvas id="gameCanvas"></canvas>
        <DonationBanner />

        {move || {
            let p = phase.get();
            // Subscribe to expired signal to re-render when session expires
            let _expired = expired.get();
            match p {
                Phase::Lobby => {
                    let has_url_session = state_clone.borrow().url_session_id.is_some();
                    if has_url_session {
                        view! {
                            <lobby::JoinMode
                                state=(*state_clone).clone()
                                net=(*net_clone).clone()
                                checked=checked
                            />
                        }.into_any()
                    } else {
                        view! {
                            <lobby::NormalLobby
                                state=(*state_clone).clone()
                                net=(*net_clone).clone()
                                sessions=sessions
                                expired=expired
                                auth_signal=auth
                            />
                        }.into_any()
                    }
                }
                Phase::MatchLobby => {
                    view! {
                        <IngameUI state=(*state_clone).clone() net=(*net_clone).clone() />
                        <crate::match_lobby::MatchLobby
                            state=(*state_clone).clone()
                            net=(*net_clone).clone()
                            lobby=lobby
                        />
                    }.into_any()
                }
                _ => {
                    view! {
                        <IngameUI state=(*state_clone).clone() net=(*net_clone).clone() />
                    }.into_any()
                }
            }
        }}
    }
}

#[component]
fn IngameUI(state: SharedState, net: SharedNetwork) -> impl IntoView {
    // Setup buttons after this component mounts
    let state_for_setup = send_wrapper::SendWrapper::new(state.clone());
    let net_for_chat = send_wrapper::SendWrapper::new(net.clone());
    let state_for_chat = send_wrapper::SendWrapper::new(state.clone());

    Effect::new(move |_| {
        crate::canvas::setup_fullscreen();
        crate::canvas::setup_controller_btn((*state_for_setup).clone());

        // Setup Enter key to toggle chat
        let state_k = (*state_for_chat).clone();
        let net_k = (*net_for_chat).clone();
        let document = web_sys::window().unwrap().document().unwrap();
        let closure = wasm_bindgen::closure::Closure::wrap(Box::new(move |e: web_sys::KeyboardEvent| {
            let doc = web_sys::window().unwrap().document().unwrap();
            if e.key() == "Enter" {
                let chat_open = state_k.borrow().chat_open;
                if chat_open {
                    // Send message
                    if let Some(input) = doc.get_element_by_id("chatInput")
                        .and_then(|el| el.dyn_into::<web_sys::HtmlInputElement>().ok())
                    {
                        let text = input.value();
                        if !text.trim().is_empty() {
                            let team = text.starts_with("/t ") || text.starts_with("/team ");
                            let clean = if team {
                                text.trim_start_matches("/t ").trim_start_matches("/team ").to_string()
                            } else {
                                text
                            };
                            Network::send_chat(&net_k, &clean, team);
                        }
                        input.set_value("");
                        let _ = input.blur();
                    }
                    state_k.borrow_mut().chat_open = false;
                    // Hide chat box
                    if let Some(box_el) = doc.get_element_by_id("chatInputBox") {
                        let _ = box_el.class_list().remove_1("open");
                    }
                } else {
                    // Open chat
                    let phase = state_k.borrow().phase.clone();
                    if matches!(phase, crate::state::Phase::Playing | crate::state::Phase::Dead) {
                        state_k.borrow_mut().chat_open = true;
                        // Show chat box and focus input
                        if let Some(box_el) = doc.get_element_by_id("chatInputBox") {
                            let _ = box_el.class_list().add_1("open");
                        }
                        if let Some(input) = doc.get_element_by_id("chatInput")
                            .and_then(|el| el.dyn_into::<web_sys::HtmlInputElement>().ok())
                        {
                            let _ = input.focus();
                        }
                        e.prevent_default();
                    }
                }
            } else if e.key() == "Escape" {
                let chat_open = state_k.borrow().chat_open;
                if chat_open {
                    state_k.borrow_mut().chat_open = false;
                    if let Some(box_el) = doc.get_element_by_id("chatInputBox") {
                        let _ = box_el.class_list().remove_1("open");
                    }
                    if let Some(input) = doc.get_element_by_id("chatInput")
                        .and_then(|el| el.dyn_into::<web_sys::HtmlInputElement>().ok())
                    {
                        input.set_value("");
                        let _ = input.blur();
                    }
                }
            }
        }) as Box<dyn FnMut(web_sys::KeyboardEvent)>);
        let _ = document.add_event_listener_with_callback("keydown", closure.as_ref().unchecked_ref());
        closure.forget();
    });

    view! {
        <button id="fullscreenBtn" title="Toggle Fullscreen">
            <svg width="16" height="16" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round">
                <path d="M2 6V2h4M10 2h4v4M14 10v4h-4M6 14H2v-4"/>
            </svg>
        </button>
        <button id="controllerBtn" title="Phone Controller">
            <svg width="16" height="16" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round">
                <rect x="4" y="1" width="8" height="14" rx="1.5"/>
                <line x1="6" y1="12" x2="10" y2="12"/>
            </svg>
        </button>
        <div id="controllerOverlay">
            <p class="qr-hint">"Scan with your phone to use as controller"</p>
            <div class="qr-box"><img id="qrImg" alt="QR Code"/></div>
            <p class="qr-url" id="qrUrl"></p>
            <button class="btn-close" id="qrClose">"Close"</button>
        </div>
        <div id="chatInputBox">
            <input type="text" id="chatInput" placeholder="Press Enter to chat (/t for team)" maxlength="200" autocomplete="off" />
        </div>
    }
}

#[component]
fn DonationBanner() -> impl IntoView {
    const ADDRS: &[(&str, &str)] = &[
        ("BTC", "bc1qqx35t04knmy7l2y520l7tpzpmz0qvsl3289vuk"),
        ("ETH", "0x759094ACa57603032db78bE296a7EE962876E190"),
        ("BNB", "0x759094ACa57603032db78bE296a7EE962876E190"),
        ("TRX", "TTvHHqx99xnfANinQugRyJyyXrhck7JxKi"),
        ("TON", "UQBDXLK1Qk6CfT3pI7A1ot_Y2zUtviJH_p55MTpW8lqFDwd0"),
        ("SOL", "5jDfJKRqnAbSTb2U9s1FxfXVeetm26GYXBcqLa5jGVdk"),
        ("SUI", "0x6dba702610c133b35f7f508acfec8461683baea2842e73a7b66015129b4c2c93"),
    ];

    let make_spans = || -> Vec<_> {
        ADDRS.iter().map(|(net, addr)| {
            let a = addr.to_string();
            let display = addr.to_string();
            let network = *net;
            view! {
                <span class="donation-sep">"|"</span>
                <span class="donation-net">{network}": "</span>
                <span class="donation-addr" title="Click to copy"
                    on:click=move |e: web_sys::MouseEvent| {
                        let _ = js_sys::eval(&format!("navigator.clipboard.writeText('{}')", a));
                        if let Some(target) = e.target() {
                            if let Ok(el) = target.dyn_into::<web_sys::HtmlElement>() {
                                let _ = el.style().set_property("color", "#44dd88");
                                let el2 = el.clone();
                                gloo_timers::callback::Timeout::new(1500, move || {
                                    let _ = el2.style().set_property("color", "");
                                }).forget();
                            }
                        }
                    }
                >{display}</span>
            }
        }).collect()
    };

    view! {
        <div class="donation-banner">
            <div class="donation-scroll">
                <span class="donation-text">
                    "\u{2605} This game is free & runs on donations \u{2014} no ads, ever! "
                    {make_spans()}
                    " \u{2605}"
                </span>
                <span class="donation-text">
                    "\u{2605} This game is free & runs on donations \u{2014} no ads, ever! "
                    {make_spans()}
                    " \u{2605}"
                </span>
            </div>
        </div>
    }
}

#[component]
fn ControllerMode(sid: String, pid: String) -> impl IntoView {
    // Init controller on mount
    let sid_clone = sid.clone();
    let pid_clone = pid.clone();
    Effect::new(move |_| {
        controller::init_controller(&sid_clone, &pid_clone);
    });

    view! {
        <div id="controllerRoot">
            <div id="ctrlRotateMsg">
                <div class="rotate-icon">
                    <svg width="80" height="80" viewBox="0 0 80 80" fill="none" stroke="#6688aa" stroke-width="2">
                        <rect x="20" y="10" width="40" height="60" rx="4" stroke-dasharray="4 2"/>
                        <path d="M50 70 L70 50 L70 30 L30 30 L10 50 L10 70 Z" fill="rgba(50,100,200,0.1)" stroke="#4488ff" stroke-dasharray="4 2"/>
                        <path d="M55 25 C60 15, 70 20, 65 28" stroke="#ffcc00" stroke-width="2" fill="none"/>
                        <path d="M63 22 L65 28 L59 27" stroke="#ffcc00" stroke-width="2" fill="none"/>
                    </svg>
                </div>
                <p>"Rotate your phone to landscape"</p>
            </div>
            <div id="ctrlPad" style="display:none;">
                <div id="ctrlStatus">"Connecting..."</div>
                <div class="ctrl-divider-left"></div>
                <div class="ctrl-divider-right"></div>
                <div class="ctrl-center">
                    <div class="ctrl-boost-indicator" id="boostIndicator"></div>
                    <div class="ctrl-label">"BOOST"</div>
                </div>
                <div class="ctrl-left">
                    <div class="ctrl-label">"Drag to navigate"</div>
                    <div class="ctrl-joystick-ring" id="joystickRing">
                        <div class="ctrl-joystick-knob" id="joystickKnob"></div>
                    </div>
                </div>
                <div class="ctrl-right">
                    <div class="ctrl-label">"Tap to fire"</div>
                    <div class="ctrl-fire-indicator" id="fireIndicator"></div>
                </div>
            </div>
        </div>
    }
}
