use leptos::prelude::*;
use wasm_bindgen::JsCast;
use crate::state::SharedState;
use crate::network::{Network, SharedNetwork};
use crate::protocol::{SessionInfo, CheckedMsg};

#[component]
pub fn NormalLobby(
    state: SharedState,
    net: SharedNetwork,
    sessions: RwSignal<Vec<SessionInfo>>,
    expired: RwSignal<bool>,
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
                <div class="name-input-group">
                    <label for="playerName">"Pilot Name"</label>
                    <input type="text" id="playerName" maxlength="16" placeholder="Enter your name..." value="Pilot" />
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
