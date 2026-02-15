use leptos::prelude::*;
use wasm_bindgen::JsCast;
use crate::state::SharedState;
use crate::network::{Network, SharedNetwork};
use crate::protocol::{SessionInfo, CheckedMsg, StoreItem};

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
    let state_friends = state.clone();
    let state_store = state.clone();
    let net_friends = net.clone();
    let net_store = net.clone();
    let net_auth = net.clone();
    let store_open = RwSignal::new(false);
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
                <StoreButton
                    state=state_store.clone()
                    net=net_store.clone()
                    auth_signal=auth_signal
                    store_open=store_open
                />
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
                <FriendsPanel state=state_friends net=net_friends auth_signal=auth_signal />
            </div>
        </div>
    }
}

#[component]
fn StoreButton(
    state: SharedState,
    net: SharedNetwork,
    auth_signal: RwSignal<Option<String>>,
    store_open: RwSignal<bool>,
) -> impl IntoView {
    let net_store = send_wrapper::SendWrapper::new(net.clone());
    let net_daily = send_wrapper::SendWrapper::new(net.clone());
    let state_credits = send_wrapper::SendWrapper::new(state.clone());
    let state_store_view = send_wrapper::SendWrapper::new(state.clone());
    let net_buy = send_wrapper::SendWrapper::new(net.clone());
    let net_equip = send_wrapper::SendWrapper::new(net.clone());

    let on_open_store = move |_: web_sys::MouseEvent| {
        Network::send_store_request(&net_store);
        store_open.set(true);
    };

    let on_close_store = move |_: web_sys::MouseEvent| {
        store_open.set(false);
    };

    let on_daily = move |_: web_sys::MouseEvent| {
        Network::send_daily_login(&net_daily);
    };

    view! {
        <div style:display=move || if auth_signal.get().is_some() { "block" } else { "none" }>
            <div style="display:flex;align-items:center;justify-content:space-between;margin-bottom:8px;gap:6px;">
                <span style="color:#ffcc00;font-size:13px;font-weight:bold;">
                    {move || format!("{} credits", state_credits.borrow().auth_credits)}
                </span>
                <span style="display:flex;gap:4px;">
                    <button class="btn btn-small btn-login" on:click=on_daily>"Daily Bonus"</button>
                    <button class="btn btn-small btn-join" on:click=on_open_store>"Store"</button>
                </span>
            </div>
            {move || {
                if !store_open.get() {
                    return view! { <span></span> }.into_any();
                }
                let s = state_store_view.borrow();
                let items = s.store_items.clone();
                let owned = s.owned_skins.clone();
                let credits = s.auth_credits;
                let equipped_skin = s.equipped_skin.clone();
                let equipped_trail = s.equipped_trail.clone();
                drop(s);

                let rarity_name = |r: i32| match r {
                    0 => "Common",
                    1 => "Rare",
                    2 => "Epic",
                    3 => "Legendary",
                    _ => "?",
                };
                let rarity_color = |r: i32| match r {
                    0 => "#aaaaaa",
                    1 => "#44aaff",
                    2 => "#aa44ff",
                    3 => "#ffcc00",
                    _ => "#ffffff",
                };

                let skins: Vec<_> = items.iter().filter(|i| i.item_type == "skin").cloned().collect();
                let trails: Vec<_> = items.iter().filter(|i| i.item_type == "trail").cloned().collect();

                let make_items = |items: Vec<StoreItem>| {
                    let net_b = (*net_buy).clone();
                    let net_e = (*net_equip).clone();
                    let owned_c = owned.clone();
                    let eq_skin = equipped_skin.clone();
                    let eq_trail = equipped_trail.clone();
                    items.into_iter().map(move |item| {
                        let is_owned = owned_c.contains(&item.id);
                        let is_equipped = item.id == eq_skin || item.id == eq_trail;
                        let can_buy = !is_owned && credits >= item.price;
                        let net_b2 = net_b.clone();
                        let net_e2 = net_e.clone();
                        let id_buy = item.id.clone();
                        let id_equip = item.id.clone();
                        let item_type = item.item_type.clone();
                        let eq_s = eq_skin.clone();
                        let eq_t = eq_trail.clone();
                        view! {
                            <div class="store-item" style=format!("border-left:3px solid {}", rarity_color(item.rarity))>
                                <div style="display:flex;align-items:center;gap:6px;">
                                    <span class="store-swatch" style=format!("background:{}", item.color1)></span>
                                    <span style="color:#ffffff;font-size:12px;font-weight:bold;">{item.name.clone()}</span>
                                    <span style=format!("color:{};font-size:10px;", rarity_color(item.rarity))>
                                        {rarity_name(item.rarity)}
                                    </span>
                                </div>
                                <div style="display:flex;align-items:center;gap:4px;">
                                    {if is_equipped {
                                        view! { <span style="color:#44ff88;font-size:10px;">"EQUIPPED"</span> }.into_any()
                                    } else if is_owned {
                                        let item_t = item_type.clone();
                                        view! {
                                            <button class="btn-accept" on:click=move |_| {
                                                let (sk, tr) = if item_t == "skin" {
                                                    (id_equip.as_str(), eq_t.as_str())
                                                } else {
                                                    (eq_s.as_str(), id_equip.as_str())
                                                };
                                                Network::send_equip(&net_e2, sk, tr);
                                            }>"Equip"</button>
                                        }.into_any()
                                    } else {
                                        view! {
                                            <button
                                                class=if can_buy { "btn-accept" } else { "btn-decline" }
                                                disabled=!can_buy
                                                on:click=move |_| {
                                                    Network::send_buy(&net_b2, &id_buy);
                                                }
                                            >{format!("{} cr", item.price)}</button>
                                        }.into_any()
                                    }}
                                </div>
                            </div>
                        }
                    }).collect::<Vec<_>>()
                };

                let skin_views = make_items(skins);
                let trail_views = make_items(trails);

                view! {
                    <div class="store-panel">
                        <div style="display:flex;justify-content:space-between;align-items:center;margin-bottom:6px;">
                            <h3 style="color:#ffcc00;font-size:14px;margin:0;">"Store"</h3>
                            <button class="btn-decline" on:click=on_close_store>"Close"</button>
                        </div>
                        <h4 style="color:#88aacc;font-size:11px;margin:4px 0;text-transform:uppercase;letter-spacing:1px;">"Ship Skins"</h4>
                        <div class="store-items">{skin_views}</div>
                        <h4 style="color:#88aacc;font-size:11px;margin:4px 0;text-transform:uppercase;letter-spacing:1px;">"Trail Effects"</h4>
                        <div class="store-items">{trail_views}</div>
                    </div>
                }.into_any()
            }}
        </div>
    }
}

#[component]
fn FriendsPanel(
    state: SharedState,
    net: SharedNetwork,
    auth_signal: RwSignal<Option<String>>,
) -> impl IntoView {
    let net_add = send_wrapper::SendWrapper::new(net.clone());
    let net_accept = send_wrapper::SendWrapper::new(net.clone());
    let net_decline = send_wrapper::SendWrapper::new(net.clone());
    let state_friends = send_wrapper::SendWrapper::new(state.clone());
    let net_list = send_wrapper::SendWrapper::new(net.clone());

    // Fetch friend list when panel appears and user is logged in
    let net_init = send_wrapper::SendWrapper::new(net.clone());
    Effect::new(move |_| {
        if auth_signal.get().is_some() {
            Network::send_friend_list(&net_init);
        }
    });

    let on_add_friend = move |_: web_sys::MouseEvent| {
        let doc = web_sys::window().unwrap().document().unwrap();
        if let Some(input) = doc.get_element_by_id("friendInput")
            .and_then(|e| e.dyn_into::<web_sys::HtmlInputElement>().ok())
        {
            let username = input.value();
            if !username.trim().is_empty() {
                Network::send_friend_add(&net_add, username.trim());
                input.set_value("");
            }
        }
    };

    view! {
        <div class="friends-panel" style:display=move || if auth_signal.get().is_some() { "block" } else { "none" }>
            <h3>"Friends"</h3>
            <div class="friend-add-form">
                <input type="text" id="friendInput" placeholder="Add friend by username..." maxlength="16" />
                <button class="btn-friend-add" on:click=on_add_friend>"Add"</button>
            </div>
            {move || {
                let s = state_friends.borrow();
                let requests = s.friend_requests.clone();
                let friends = s.friends.clone();
                drop(s);

                let request_views: Vec<_> = requests.iter().map(|r| {
                    let name = r.username.clone();
                    let name_a = name.clone();
                    let name_d = name.clone();
                    let net_a = (*net_accept).clone();
                    let net_d = (*net_decline).clone();
                    let net_l1 = (*net_list).clone();
                    let net_l2 = (*net_list).clone();
                    view! {
                        <div class="friend-request-item">
                            <span class="friend-name">{name}" wants to be friends"</span>
                            <span>
                                <button class="btn-accept" on:click=move |_| {
                                    Network::send_friend_accept(&net_a, &name_a);
                                    // Refresh after action
                                    let _ = gloo_timers::callback::Timeout::new(500, {
                                        let n = net_l1.clone();
                                        move || Network::send_friend_list(&n)
                                    });
                                }>"Accept"</button>
                                <button class="btn-decline" on:click=move |_| {
                                    Network::send_friend_decline(&net_d, &name_d);
                                    let _ = gloo_timers::callback::Timeout::new(500, {
                                        let n = net_l2.clone();
                                        move || Network::send_friend_list(&n)
                                    });
                                }>"Decline"</button>
                            </span>
                        </div>
                    }
                }).collect();

                let friend_views: Vec<_> = friends.iter().map(|f| {
                    let name = f.username.clone();
                    let online = f.online;
                    let level = f.level;
                    view! {
                        <div class="friend-item">
                            <span>
                                <span class="friend-name">{name}</span>
                                {if online {
                                    view! { <span class="friend-online">"ONLINE"</span> }.into_any()
                                } else {
                                    view! { <span class="friend-offline">"offline"</span> }.into_any()
                                }}
                            </span>
                            <span class="lb-level">"Lv." {level}</span>
                        </div>
                    }
                }).collect();

                view! {
                    <div>
                        {request_views}
                        {friend_views}
                        {if friends.is_empty() && requests.is_empty() {
                            view! { <p class="no-sessions" style="font-size:11px">"No friends yet. Add someone!"</p> }.into_any()
                        } else {
                            view! { <span></span> }.into_any()
                        }}
                    </div>
                }
            }}
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
