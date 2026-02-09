import { state } from './state.js';
import { listSessions, createSession, joinSession } from './network.js';

let lobbyEl = null;
let refreshInterval = null;

export function initLobby() {
    lobbyEl = document.getElementById('lobby');
    renderLobby();
    startRefresh();
}

export function showLobby() {
    state.phase = 'lobby';
    lobbyEl.style.display = 'flex';
    renderLobby();
    startRefresh();
}

export function hideLobby() {
    lobbyEl.style.display = 'none';
    stopRefresh();
}

function startRefresh() {
    stopRefresh();
    listSessions();
    refreshInterval = setInterval(() => listSessions(), 3000);
}

function stopRefresh() {
    if (refreshInterval) {
        clearInterval(refreshInterval);
        refreshInterval = null;
    }
}

export function updateSessions(sessions) {
    state.sessions = sessions;
    if (state.phase === 'lobby') {
        renderSessionList();
    }
}

function renderLobby() {
    lobbyEl.innerHTML = `
        <div class="lobby-panel">
            <h1 class="title">STAR WARS</h1>
            <h2 class="subtitle">Space Battle</h2>
            <div class="name-input-group">
                <label for="playerName">Pilot Name</label>
                <input type="text" id="playerName" maxlength="16" placeholder="Enter your name..." value="Pilot" />
            </div>
            <div class="lobby-actions">
                <button id="btnCreate" class="btn btn-primary">Create Battle</button>
            </div>
            <div class="session-list-container">
                <h3>Active Battles</h3>
                <div id="sessionList" class="session-list">
                    <p class="no-sessions">Searching for battles...</p>
                </div>
            </div>
        </div>
    `;

    document.getElementById('btnCreate').addEventListener('click', () => {
        const name = document.getElementById('playerName').value.trim() || 'Pilot';
        createSession(name, 'Battle Arena');
    });
}

function renderSessionList() {
    const listEl = document.getElementById('sessionList');
    if (!listEl) return;

    if (state.sessions.length === 0) {
        listEl.innerHTML = '<p class="no-sessions">No active battles. Create one!</p>';
        return;
    }

    listEl.innerHTML = state.sessions.map(s => `
        <div class="session-item" data-sid="${s.id}">
            <span class="session-name">${escapeHtml(s.name)}</span>
            <span class="session-players">${s.players} pilot${s.players !== 1 ? 's' : ''}</span>
            <button class="btn btn-join" data-sid="${s.id}">Join</button>
        </div>
    `).join('');

    listEl.querySelectorAll('.btn-join').forEach(btn => {
        btn.addEventListener('click', () => {
            const name = document.getElementById('playerName').value.trim() || 'Pilot';
            joinSession(name, btn.dataset.sid);
        });
    });
}

function escapeHtml(str) {
    const div = document.createElement('div');
    div.textContent = str;
    return div.innerHTML;
}
