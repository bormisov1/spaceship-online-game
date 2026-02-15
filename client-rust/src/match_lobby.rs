use leptos::prelude::*;
use crate::state::SharedState;
use crate::network::{Network, SharedNetwork};

#[component]
pub fn MatchLobby(
    state: SharedState,
    net: SharedNetwork,
) -> impl IntoView {
    let state_title = send_wrapper::SendWrapper::new(state.clone());
    let state_roster_r = send_wrapper::SendWrapper::new(state.clone());
    let state_roster_b = send_wrapper::SendWrapper::new(state.clone());
    let state_is_team = send_wrapper::SendWrapper::new(state.clone());
    let net_ready = send_wrapper::SendWrapper::new(net.clone());
    let net_team_r = send_wrapper::SendWrapper::new(net.clone());
    let net_team_b = send_wrapper::SendWrapper::new(net.clone());

    view! {
        <div class="match-lobby-overlay">
            <div class="match-lobby-panel">
                <h2 class="match-lobby-title">
                    {move || {
                        let s = state_title.borrow();
                        s.game_mode.name().to_string()
                    }}
                </h2>

                // Team picker (only for team modes)
                <div class="team-picker" style={move || {
                    let s = state_is_team.borrow();
                    if matches!(s.game_mode, crate::state::GameMode::TDM | crate::state::GameMode::CTF) {
                        "display:flex"
                    } else {
                        "display:none"
                    }
                }}>
                    <div class="team-side team-red">
                        <h3 class="team-label" style="color: #ff4444">"RED TEAM"</h3>
                        <button class="btn btn-team-red" on:click=move |_| {
                            Network::send_team_pick(&net_team_r, 1);
                        }>"Join Red"</button>
                        <div class="team-roster">
                            {move || {
                                let s = state_roster_r.borrow();
                                s.team_red.iter().map(|p| {
                                    let ready_class = if p.ready { " ready" } else { "" };
                                    let name = p.n.clone();
                                    let ready = p.ready;
                                    view! {
                                        <div class={format!("team-player{}", ready_class)}>
                                            <span class="player-name">{name}</span>
                                            {if ready {
                                                view! { <span class="ready-check">" \u{2713}"</span> }.into_any()
                                            } else {
                                                view! { <span></span> }.into_any()
                                            }}
                                        </div>
                                    }
                                }).collect::<Vec<_>>()
                            }}
                        </div>
                    </div>
                    <div class="team-side team-blue">
                        <h3 class="team-label" style="color: #4488ff">"BLUE TEAM"</h3>
                        <button class="btn btn-team-blue" on:click=move |_| {
                            Network::send_team_pick(&net_team_b, 2);
                        }>"Join Blue"</button>
                        <div class="team-roster">
                            {move || {
                                let s = state_roster_b.borrow();
                                s.team_blue.iter().map(|p| {
                                    let ready_class = if p.ready { " ready" } else { "" };
                                    let name = p.n.clone();
                                    let ready = p.ready;
                                    view! {
                                        <div class={format!("team-player{}", ready_class)}>
                                            <span class="player-name">{name}</span>
                                            {if ready {
                                                view! { <span class="ready-check">" \u{2713}"</span> }.into_any()
                                            } else {
                                                view! { <span></span> }.into_any()
                                            }}
                                        </div>
                                    }
                                }).collect::<Vec<_>>()
                            }}
                        </div>
                    </div>
                </div>

                <button class="btn btn-ready" on:click=move |_| {
                    Network::send_ready(&net_ready);
                }>"Ready"</button>
            </div>
        </div>
    }
}
