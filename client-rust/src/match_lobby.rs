use leptos::prelude::*;
use crate::state::SharedState;
use crate::network::{Network, SharedNetwork};

#[component]
pub fn MatchLobby(
    state: SharedState,
    net: SharedNetwork,
    lobby: RwSignal<u64>,
) -> impl IntoView {
    let state_title = send_wrapper::SendWrapper::new(state.clone());
    let state_roster_r = send_wrapper::SendWrapper::new(state.clone());
    let state_roster_b = send_wrapper::SendWrapper::new(state.clone());
    let state_unassigned = send_wrapper::SendWrapper::new(state.clone());
    let state_unassigned2 = send_wrapper::SendWrapper::new(state.clone());
    let state_is_team = send_wrapper::SendWrapper::new(state.clone());
    let state_status = send_wrapper::SendWrapper::new(state.clone());
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
                                let _ver = lobby.get();
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
                                let _ver = lobby.get();
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

                // Unassigned players
                <div class="team-unassigned" style={move || {
                    let _ver = lobby.get();
                    let s = state_unassigned.borrow();
                    if s.team_unassigned.is_empty() {
                        "display:none"
                    } else {
                        "display:block"
                    }
                }}>
                    <h4 class="unassigned-label">"Unassigned"</h4>
                    <div class="team-roster">
                        {move || {
                            let _ver = lobby.get();
                            let s = state_unassigned2.borrow();
                            s.team_unassigned.iter().map(|p| {
                                let name = p.n.clone();
                                view! {
                                    <div class="team-player unassigned">
                                        <span class="player-name">{name}</span>
                                    </div>
                                }
                            }).collect::<Vec<_>>()
                        }}
                    </div>
                </div>

                // Status message
                <div class="lobby-status">
                    {move || {
                        let _ver = lobby.get();
                        let s = state_status.borrow();
                        let is_team = matches!(s.game_mode, crate::state::GameMode::TDM | crate::state::GameMode::CTF);
                        let count = s.lobby_player_count;
                        let min = s.lobby_min_players;
                        if is_team && count < min {
                            format!("Need at least {} players to start ({}/{})", min, count, min)
                        } else if is_team && !s.team_unassigned.is_empty() {
                            "Pick a team to get started!".to_string()
                        } else {
                            let total = s.team_red.len() + s.team_blue.len() + s.team_unassigned.len();
                            let ready_count = s.team_red.iter().chain(s.team_blue.iter()).chain(s.team_unassigned.iter())
                                .filter(|p| p.ready).count();
                            if total > 0 && ready_count < total {
                                format!("Waiting for players to ready up ({}/{})", ready_count, total)
                            } else {
                                String::new()
                            }
                        }
                    }}
                </div>

                <button class="btn btn-ready" on:click=move |_| {
                    Network::send_ready(&net_ready);
                }>"Ready"</button>
            </div>
        </div>
    }
}
