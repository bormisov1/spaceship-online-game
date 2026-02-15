use leptos::prelude::*;
use wasm_bindgen::JsCast;
use crate::state::SharedState;
use crate::network::{Network, SharedNetwork};
use crate::protocol::{SessionInfo, CheckedMsg, LeaderboardEntry};

#[component]
pub fn AuthPanel(
    state: SharedState,
    net: SharedNetwork,
    auth_signal: RwSignal<Option<String>>,
) -> impl IntoView {
    let net_login = send_wrapper::SendWrapper::new(net.clone());
    let net_register = send_wrapper::SendWrapper::new(net.clone());
    let state_logout = send_wrapper::SendWrapper::new(state.clone());
    let state_info = send_wrapper::SendWrapper::new(state.clone());

    let on_login = move |_: web_sys::MouseEvent| {
        let document = web_sys::window().unwrap().document().unwrap();
        let username = document.get_element_by_id("authUsername")
            .and_then(|e| e.dyn_into::<web_sys::HtmlInputElement>().ok())
            .map(|i| i.value()).unwrap_or_default();
        let password = document.get_element_by_id("authPassword")
            .and_then(|e| e.dyn_into::<web_sys::HtmlInputElement>().ok())
            .map(|i| i.value()).unwrap_or_default();
        if !username.is_empty() && !password.is_empty() {
            if let Some(el) = document.get_element_by_id("authError") {
                el.set_text_content(Some(""));
            }
            Network::send_login(&net_login, &username, &password);
        }
    };

    let on_register = move |_: web_sys::MouseEvent| {
        let document = web_sys::window().unwrap().document().unwrap();
        let username = document.get_element_by_id("authUsername")
            .and_then(|e| e.dyn_into::<web_sys::HtmlInputElement>().ok())
            .map(|i| i.value()).unwrap_or_default();
        let password = document.get_element_by_id("authPassword")
            .and_then(|e| e.dyn_into::<web_sys::HtmlInputElement>().ok())
            .map(|i| i.value()).unwrap_or_default();
        if !username.is_empty() && !password.is_empty() {
            if let Some(el) = web_sys::window().unwrap().document().unwrap().get_element_by_id("authError") {
                el.set_text_content(Some(""));
            }
            Network::send_register(&net_register, &username, &password);
        }
    };

    let auth_for_logout = auth_signal;
    let on_logout = move |_: web_sys::MouseEvent| {
        state_logout.borrow_mut().auth_token = None;
        state_logout.borrow_mut().auth_username = None;
        state_logout.borrow_mut().auth_player_id = 0;
        if let Ok(Some(storage)) = web_sys::window().unwrap().local_storage() {
            let _ = storage.remove_item("auth_token");
            let _ = storage.remove_item("auth_username");
        }
        auth_for_logout.set(None);
    };

    view! {
        <div class="auth-panel">
            // Logged-in view
            <div class="auth-logged-in" style:display=move || if auth_signal.get().is_some() { "flex" } else { "none" }>
                <span class="auth-user-info">
                    {move || {
                        let s = state_info.borrow();
                        let level = s.auth_level;
                        let username = s.auth_username.clone().unwrap_or_default();
                        let kd = if s.auth_deaths > 0 {
                            format!("{:.1}", s.auth_kills as f64 / s.auth_deaths as f64)
                        } else {
                            format!("{}", s.auth_kills)
                        };
                        format!("Lv.{} {} | K/D: {}", level, username, kd)
                    }}
                </span>
                <button class="btn btn-small btn-logout" on:click=on_logout>"Logout"</button>
            </div>
            // Login/Register form
            <div class="auth-form" style:display=move || if auth_signal.get().is_none() { "flex" } else { "none" }>
                <input type="text" id="authUsername" placeholder="Username" maxlength="16" class="auth-input" />
                <input type="password" id="authPassword" placeholder="Password" class="auth-input" />
                <div class="auth-buttons">
                    <button class="btn btn-small btn-login" on:click=on_login>"Login"</button>
                    <button class="btn btn-small btn-register" on:click=on_register>"Register"</button>
                </div>
                <p id="authError" class="auth-error"></p>
            </div>
        </div>
    }
}

#[component]
pub fn NormalLobby(
    state: SharedState,
    net: SharedNetwork,
    sessions: RwSignal<Vec<SessionInfo>>,
    expired: RwSignal<bool>,
    auth_signal: RwSignal<Option<String>>,
) -> impl IntoView {
    let net_create = net.clone();
    let net_join = send_wrapper::SendWrapper::new(net.clone());

    let state_for_create = state.clone();
    let on_create = move |_| {
        let document = web_sys::window().unwrap().document().unwrap();
        let name = document
            .get_element_by_id("playerName")
            .and_then(|e| e.dyn_into::<web_sys::HtmlInputElement>().ok())
            .map(|i: web_sys::HtmlInputElement| i.value())
            .unwrap_or_else(|| "Pilot".to_string());
        let name = if name.trim().is_empty() { "Pilot".to_string() } else { name.trim().to_string() };
        state_for_create.borrow_mut().pending_name = Some(name.clone());
        let mode_str = document
            .get_element_by_id("gameMode")
            .and_then(|e| e.dyn_into::<web_sys::HtmlSelectElement>().ok())
            .map(|s| s.value())
            .unwrap_or_else(|| "0".to_string());
        let mode: i32 = mode_str.parse().unwrap_or(0);
        let mode_name = match mode {
            1 => "Team Deathmatch",
            2 => "CTF",
            3 => "Wave Survival",
            _ => "Battle Arena",
        };
        Network::create_session(&net_create, &name, mode_name, mode);
    };

    let state_auth = state.clone();
    let state_lb = send_wrapper::SendWrapper::new(state.clone());
    let net_auth = net.clone();
    let default_name = state.borrow().auth_username.clone().unwrap_or_else(|| "Pilot".to_string());

    view! {
        <div id="lobby">
            <div class="lobby-panel">
                {move || {
                    if expired.get() {
                        view! {
                            <div class="expired-banner">"Session does not exist or has ended."</div>
                        }.into_any()
                    } else {
                        view! { <span></span> }.into_any()
                    }
                }}
                <h1 class="title">"STAR WARS"</h1>
                <h2 class="subtitle">"Space Battle"</h2>
                <AuthPanel state=state_auth.clone() net=net_auth.clone() auth_signal=auth_signal />
                <div class="name-input-group">
                    <label for="playerName">"Pilot Name"</label>
                    <input type="text" id="playerName" maxlength="16" placeholder="Enter your name..."
                        value={default_name} />
                </div>
                <div class="mode-select-group">
                    <label for="gameMode">"Game Mode"</label>
                    <select id="gameMode">
                        <option value="0">"Free-For-All"</option>
                        <option value="1">"Team Deathmatch"</option>
                        <option value="2">"Capture the Flag"</option>
                        <option value="3">"Wave Survival"</option>
                    </select>
                </div>
                <div class="lobby-actions">
                    <button class="btn btn-primary" on:click=on_create>"Create Battle"</button>
                </div>
                <div class="session-list-container">
                    <h3>"Active Battles"</h3>
                    <div class="session-list">
                        {move || {
                            let sessions = sessions.get();
                            if sessions.is_empty() {
                                view! { <p class="no-sessions">"No active battles. Create one!"</p> }.into_any()
                            } else {
                                let net_j = net_join.clone();
                                view! {
                                    <For
                                        each=move || sessions.clone()
                                        key=|s| s.id.clone()
                                        let:session
                                    >
                                        {
                                            let sid = session.id.clone();
                                            let name = session.name.clone();
                                            let players = session.players;
                                            let net_click = (*net_j).clone();
                                            let sid_click = sid.clone();
                                            let mode = session.mode;
                                            let mode_label = match mode {
                                                1 => "TDM",
                                                2 => "CTF",
                                                3 => "Survival",
                                                _ => "FFA",
                                            };
                                            let player_text = if players == 1 {
                                                format!("{} pilot", players)
                                            } else {
                                                format!("{} pilots", players)
                                            };
                                            view! {
                                                <div class="session-item">
                                                    <span class="session-name">{name}</span>
                                                    <span class="session-mode">{mode_label}</span>
                                                    <span class="session-players">{player_text}</span>
                                                    <button class="btn btn-join" on:click=move |_| {
                                                        let document = web_sys::window().unwrap().document().unwrap();
                                                        let pname = document
                                                            .get_element_by_id("playerName")
                                                            .and_then(|e| e.dyn_into::<web_sys::HtmlInputElement>().ok())
                                                            .map(|i: web_sys::HtmlInputElement| i.value())
                                                            .unwrap_or_else(|| "Pilot".to_string());
                                                        let pname = if pname.trim().is_empty() { "Pilot".to_string() } else { pname.trim().to_string() };
                                                        Network::join_session(&net_click, &pname, &sid_click);
                                                    }>"Join"</button>
                                                </div>
                                            }
                                        }
                                    </For>
                                }.into_any()
                            }
                        }}
                    </div>
                </div>
                <div class="leaderboard-container">
                    <h3>"Leaderboard"</h3>
                    <div class="leaderboard">
                        {move || {
                            let lb = state_lb.borrow().leaderboard.clone();
                            if lb.is_empty() {
                                view! { <p class="no-sessions">"No rankings yet"</p> }.into_any()
                            } else {
                                view! {
                                    <table class="leaderboard-table">
                                        <thead><tr>
                                            <th>"#"</th><th>"Pilot"</th><th>"Lv"</th><th>"K"</th><th>"D"</th><th>"W"</th>
                                        </tr></thead>
                                        <tbody>
                                            {lb.iter().map(|e| {
                                                let kd = if e.deaths > 0 {
                                                    format!("{:.1}", e.kills as f64 / e.deaths as f64)
                                                } else {
                                                    format!("{}", e.kills)
                                                };
                                                view! {
                                                    <tr>
                                                        <td class="lb-rank">{e.rank}</td>
                                                        <td class="lb-name">{e.username.clone()}</td>
                                                        <td class="lb-level">{e.level}</td>
                                                        <td>{e.kills}</td>
                                                        <td>{e.deaths}</td>
                                                        <td>{e.wins}</td>
                                                    </tr>
                                                }
                                            }).collect::<Vec<_>>()}
                                        </tbody>
                                    </table>
                                }.into_any()
                            }
                        }}
                    </div>
                </div>
            </div>
        </div>
    }
}

#[component]
pub fn JoinMode(
    state: SharedState,
    net: SharedNetwork,
    checked: RwSignal<Option<CheckedMsg>>,
) -> impl IntoView {
    let net_join = net.clone();
    let state_clone = state.clone();

    view! {
        <div id="lobby">
            <div class="lobby-panel">
                <h1 class="title">"STAR WARS"</h1>
                <h2 class="subtitle">"Space Battle"</h2>
                <div class="name-input-group">
                    <label for="playerName">"Pilot Name"</label>
                    <input type="text" id="playerName" maxlength="16" placeholder="Enter your name..." value="Pilot" />
                </div>
                <div class="join-status">
                    {move || {
                        match checked.get() {
                            None => view! { <p class="no-sessions">"Checking session..."</p> }.into_any(),
                            Some(c) => {
                                if !c.exists {
                                    view! {
                                        <div>
                                            <p class="error-msg">"Session does not exist or has ended."</p>
                                            <a href={crate::app::base_path()} class="btn btn-primary" style="text-decoration:none;display:inline-block;margin-top:12px;">"Go to Lobby"</a>
                                        </div>
                                    }.into_any()
                                } else {
                                    let player_text = if c.players == 1 {
                                        format!("{} pilot", c.players)
                                    } else {
                                        format!("{} pilots", c.players)
                                    };
                                    view! {
                                        <p class="session-info">
                                            "Battle: " <strong>{c.name.clone()}</strong> " — " {player_text}
                                        </p>
                                    }.into_any()
                                }
                            }
                        }
                    }}
                </div>
                <div class="lobby-actions">
                    {
                        let net_j = send_wrapper::SendWrapper::new(net_join.clone());
                        let st = send_wrapper::SendWrapper::new(state_clone.clone());
                        move || {
                            let show = checked.get().map(|c| c.exists).unwrap_or(false);
                            if show {
                                let net_j2 = (*net_j).clone();
                                let st2 = (*st).clone();
                                view! {
                                    <button class="btn btn-primary" on:click=move |_| {
                                        let document = web_sys::window().unwrap().document().unwrap();
                                        let name = document
                                            .get_element_by_id("playerName")
                                            .and_then(|e| e.dyn_into::<web_sys::HtmlInputElement>().ok())
                                            .map(|i: web_sys::HtmlInputElement| i.value())
                                            .unwrap_or_else(|| "Pilot".to_string());
                                        let name = if name.trim().is_empty() { "Pilot".to_string() } else { name.trim().to_string() };
                                        if let Some(sid) = &st2.borrow().url_session_id {
                                            Network::join_session(&net_j2, &name, sid);
                                        }
                                    }>"Join Battle"</button>
                                }.into_any()
                            } else {
                                view! { <span></span> }.into_any()
                            }
                        }
                    }
                </div>
            </div>
        </div>
    }
}
