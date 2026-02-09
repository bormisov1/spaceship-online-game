import { state } from './state.js';
import { connect, setMessageHandler } from './network.js';
import { setupInput } from './input.js';
import { initLobby, hideLobby, updateSessions } from './lobby.js';
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

    // Connect to server
    connect();

    // Start render loop
    requestAnimationFrame(gameLoop);
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
        case 'error':
            console.error('Server error:', msg.d.msg);
            break;
    }
}

function handleState(data) {
    // Store previous state for interpolation
    state.prevPlayers = new Map(state.players);
    state.prevProjectiles = new Map(state.projectiles);
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

    // Add explosion at victim location
    const victim = state.players.get(data.vid);
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
