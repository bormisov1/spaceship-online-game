import { state } from './state.js';
import { connect, setMessageHandler, onConnect, send } from './network.js';
import { setupInput } from './input.js';
import { initLobby, hideLobby, showLobby, updateSessions, updateURL, checkURLSession, handleSessionCheck } from './lobby.js';
import { render } from './renderer.js';
import { initStarfield } from './starfield.js';
import { addExplosion, addEngineParticles } from './effects.js';
import { WORLD_W, WORLD_H } from './constants.js';

// Initialize game
export function init() {
    // Setup canvases
    state.bgCanvas = document.getElementById('bgCanvas');
    state.canvas = document.getElementById('gameCanvas');
    state.bgCtx = state.bgCanvas.getContext('2d');
    state.ctx = state.canvas.getContext('2d');

    resize();
    window.addEventListener('resize', resize);

    // Init subsystems
    initStarfield();
    setupInput();
    initLobby();

    // Setup message routing
    setMessageHandler(handleMessage);

    // On WS connect, check URL session if present
    onConnect(() => checkURLSession());

    // Handle browser back/forward
    window.addEventListener('popstate', () => {
        if (state.phase === 'playing' || state.phase === 'dead') {
            send('leave', {});
            state.sessionID = null;
            state.myID = null;
            showLobby();
        }
    });

    // Fullscreen toggle
    setupFullscreen();

    // Connect to server
    connect();

    // Start render loop
    requestAnimationFrame(gameLoop);
}

function setupFullscreen() {
    const btn = document.getElementById('fullscreenBtn');
    if (!btn) return;

    // Hide button if already running as installed PWA (no browser chrome)
    const isStandalone = window.matchMedia('(display-mode: standalone)').matches
        || window.matchMedia('(display-mode: fullscreen)').matches
        || window.navigator.standalone;
    if (isStandalone) {
        btn.style.display = 'none';
        return;
    }

    // Hide button if Fullscreen API is not supported (e.g. iOS Safari)
    const elem = document.documentElement;
    if (!elem.requestFullscreen && !elem.webkitRequestFullscreen) {
        btn.style.display = 'none';
        return;
    }

    btn.addEventListener('click', () => {
        const doc = document;

        if (!doc.fullscreenElement && !doc.webkitFullscreenElement) {
            if (elem.requestFullscreen) {
                elem.requestFullscreen().catch(() => {});
            } else if (elem.webkitRequestFullscreen) {
                elem.webkitRequestFullscreen();
            }
        } else {
            if (doc.exitFullscreen) {
                doc.exitFullscreen().catch(() => {});
            } else if (doc.webkitExitFullscreen) {
                doc.webkitExitFullscreen();
            }
        }
    });

    // Update button icon on fullscreen change
    const updateIcon = () => {
        const isFs = !!(document.fullscreenElement || document.webkitFullscreenElement);
        btn.innerHTML = isFs
            ? '<svg width="16" height="16" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"><path d="M6 2v4H2M14 6h-4V2M10 14v-4h4M2 10h4v4"/></svg>'
            : '<svg width="16" height="16" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"><path d="M2 6V2h4M10 2h4v4M14 10v4h-4M6 14H2v-4"/></svg>';
    };
    document.addEventListener('fullscreenchange', updateIcon);
    document.addEventListener('webkitfullscreenchange', updateIcon);
}

function resize() {
    state.screenW = window.innerWidth;
    state.screenH = window.innerHeight;
    state.canvas.width = state.screenW;
    state.canvas.height = state.screenH;
    state.bgCanvas.width = state.screenW;
    state.bgCanvas.height = state.screenH;

    // Zoom out on small screens so more of the world is visible
    // Reference: screens with min dimension >= 700px get zoom 1.0 (no change)
    const minDim = Math.min(state.screenW, state.screenH);
    state.camZoom = Math.min(1.0, minDim / 700);

    // On mobile, reset virtual mouse to center (dead zone) so ship doesn't drift
    if (state.isMobile && !state.touchJoystick) {
        state.mouseX = state.screenW / 2;
        state.mouseY = state.screenH / 2;
    }
}

function handleMessage(msg) {
    switch (msg.t) {
        case 'state':
            handleState(msg.d);
            break;
        case 'welcome':
            handleWelcome(msg.d);
            break;
        case 'joined':
            handleJoined(msg.d);
            break;
        case 'sessions':
            updateSessions(msg.d || []);
            break;
        case 'kill':
            handleKill(msg.d);
            break;
        case 'death':
            handleDeath(msg.d);
            break;
        case 'checked':
            handleSessionCheck(msg.d);
            break;
        case 'error':
            console.error('Server error:', msg.d.msg);
            break;
    }
}

function handleState(data) {
    // Store previous state for interpolation
    state.prevPlayers = new Map(state.players);
    state.prevProjectiles = new Map(state.projectiles);
    state.prevMobs = new Map(state.mobs);
    state.prevAsteroids = new Map(state.asteroids);
    state.prevPickups = new Map(state.pickups);
    state.lastStateTime = performance.now();

    // Update current state
    state.players.clear();
    for (const p of data.p) {
        state.players.set(p.id, p);
    }

    state.projectiles.clear();
    for (const pr of data.pr) {
        state.projectiles.set(pr.id, pr);
    }

    state.mobs.clear();
    if (data.m) {
        for (const m of data.m) {
            state.mobs.set(m.id, m);
        }
    }

    state.asteroids.clear();
    if (data.a) {
        for (const a of data.a) {
            state.asteroids.set(a.id, a);
        }
    }

    state.pickups.clear();
    if (data.pk) {
        for (const pk of data.pk) {
            state.pickups.set(pk.id, pk);
        }
    }

    state.tick = data.tick;

    // Update camera to follow local player
    const me = state.players.get(state.myID);
    if (me) {
        state.camX = me.x;
        state.camY = me.y;

        // Update dead/alive phase
        if (!me.a && state.phase === 'playing') {
            state.phase = 'dead';
        } else if (me.a && state.phase === 'dead') {
            state.phase = 'playing';
            state.deathInfo = null;
        }
    }
}

function handleWelcome(data) {
    state.myID = data.id;
    state.myShip = data.s;
    state.phase = 'playing';
    hideLobby();
}

function handleJoined(data) {
    state.sessionID = data.sid;
    updateURL(data.sid);
}

function handleKill(data) {
    state.killFeed.push({
        killer: data.kn,
        victim: data.vn,
        time: performance.now(),
    });
    // Keep only last 5 kills
    if (state.killFeed.length > 5) {
        state.killFeed.shift();
    }

    // Add explosion at victim location (could be player or mob)
    const victim = state.players.get(data.vid) || state.mobs.get(data.vid);
    if (victim) {
        addExplosion(victim.x, victim.y);
    }
}

function handleDeath(data) {
    state.deathInfo = { killerName: data.kn };
    state.phase = 'dead';
}

let lastTime = 0;
function gameLoop(timestamp) {
    const dt = Math.min((timestamp - lastTime) / 1000, 0.05);
    lastTime = timestamp;

    if (state.phase === 'playing' || state.phase === 'dead') {
        render(dt);
    }

    requestAnimationFrame(gameLoop);
}

// Auto-init when DOM ready
if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', init);
} else {
    init();
}
