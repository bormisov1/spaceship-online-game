import { state } from './state.js';
import { listSessions, createSession, joinSession, checkSession } from './network.js';

const UUID_RE = /^\/([0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12})$/;

let lobbyEl = null;
let refreshInterval = null;

export function initLobby() {
    lobbyEl = document.getElementById('lobby');

    // Check URL for session UUID
    const match = window.location.pathname.match(UUID_RE);
    if (match) {
        state.urlSessionID = match[1];
    }

    renderLobby();
    if (!state.urlSessionID) {
        startRefresh();
    }
}

export function showLobby() {
    state.phase = 'lobby';
    state.urlSessionID = null;
    lobbyEl.style.display = 'flex';
    history.pushState({}, '', '/');
    renderLobby();
    startRefresh();
    // Hide controller QR button when returning to lobby
    const ctrlBtn = document.getElementById('controllerBtn');
    if (ctrlBtn) ctrlBtn.style.display = 'none';
    const ctrlOverlay = document.getElementById('controllerOverlay');
    if (ctrlOverlay) ctrlOverlay.classList.remove('visible');
}

export function hideLobby() {
    lobbyEl.style.display = 'none';
    stopRefresh();
}

export function updateURL(sid) {
    history.pushState({}, '', '/' + sid);
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
    if (state.phase === 'lobby' && !state.urlSessionID) {
        renderSessionList();
    }
}

// Called when WS connects and we have a URL session ID to verify
export function checkURLSession() {
    if (state.urlSessionID) {
        checkSession(state.urlSessionID);
    }
}

export function handleSessionCheck(data) {
    if (!state.urlSessionID) return;

    const panel = lobbyEl.querySelector('.lobby-panel');
    if (!panel) return;

    const statusEl = panel.querySelector('.join-status');
    const btnJoin = panel.querySelector('#btnJoin');

    if (!data.exists) {
        if (statusEl) {
            statusEl.innerHTML = `
                <p class="error-msg">Session does not exist or has ended.</p>
                <a href="/" class="btn btn-primary" style="text-decoration:none;display:inline-block;margin-top:12px;">Go to Lobby</a>
            `;
        }
        if (btnJoin) btnJoin.style.display = 'none';
    } else {
        if (statusEl) {
            const players = Number.isFinite(data.players) ? data.players : 0;
            statusEl.innerHTML = `<p class="session-info">Battle: <strong>${escapeHtml(data.name)}</strong> — ${players} pilot${players !== 1 ? 's' : ''}</p>`;
        }
        if (btnJoin) {
            btnJoin.disabled = false;
            btnJoin.textContent = 'Join Battle';
        }
    }
}

function renderLobby() {
    if (state.urlSessionID) {
        renderJoinMode();
    } else {
        renderNormalLobby();
    }
}

function renderJoinMode() {
    const savedName = sessionStorage.getItem('pilotName') || 'Pilot';
    sessionStorage.removeItem('pilotName');

    lobbyEl.innerHTML = `
        <div class="lobby-panel">
            <h1 class="title">STAR WARS</h1>
            <h2 class="subtitle">Space Battle</h2>
            <div class="name-input-group">
                <label for="playerName">Pilot Name</label>
                <input type="text" id="playerName" maxlength="16" placeholder="Enter your name..." value="${escapeHtml(savedName)}" />
            </div>
            <div class="join-status">
                <p class="no-sessions">Checking session...</p>
            </div>
            <div class="lobby-actions">
                <button id="btnJoin" class="btn btn-primary" disabled>Join Battle</button>
            </div>
        </div>
    `;

    document.getElementById('btnJoin').addEventListener('click', () => {
        const name = document.getElementById('playerName').value.trim() || 'Pilot';
        joinSession(name, state.urlSessionID);
    });
}

function renderNormalLobby() {
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

    listEl.innerHTML = state.sessions.map(s => {
        const players = Number.isFinite(s.players) ? s.players : 0;
        return `
        <div class="session-item" data-sid="${s.id}">
            <span class="session-name">${escapeHtml(s.name)}</span>
            <span class="session-players">${players} pilot${players !== 1 ? 's' : ''}</span>
            <button class="btn btn-join" data-sid="${s.id}">Join</button>
        </div>
    `;
    }).join('');

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
