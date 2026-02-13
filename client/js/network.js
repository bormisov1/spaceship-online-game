import { state } from './state.js';
import { RECONNECT_DELAY, INPUT_RATE } from './constants.js';

let onConnectCallback = null;

export function onConnect(cb) {
    onConnectCallback = cb;
}

let ws = null;
let messageHandler = null;
let inputInterval = null;

// Mobile auto-aim state
const AIM_ORBIT_R = 360;
const AIM_DETECT_R = 150;
let lockTargetId = null;

export function setMessageHandler(handler) {
    messageHandler = handler;
}

export function connect() {
    const proto = location.protocol === 'https:' ? 'wss:' : 'ws:';
    const url = `${proto}//${location.host}/ws`;

    ws = new WebSocket(url);

    ws.onopen = () => {
        state.connected = true;
        console.log('WebSocket connected');
        startInputLoop();
        if (onConnectCallback) onConnectCallback();
    };

    ws.onclose = () => {
        state.connected = false;
        console.log('WebSocket closed, reconnecting...');
        stopInputLoop();
        setTimeout(connect, RECONNECT_DELAY);
    };

    ws.onerror = (err) => {
        console.error('WebSocket error:', err);
    };

    ws.onmessage = (event) => {
        try {
            const msg = JSON.parse(event.data);
            if (messageHandler) {
                messageHandler(msg);
            }
        } catch (e) {
            console.error('Parse error:', e);
        }
    };
}

export function send(type, data) {
    if (!ws || ws.readyState !== WebSocket.OPEN) return;
    ws.send(JSON.stringify({ t: type, d: data }));
}

export function sendInput() {
    if (state.phase !== 'playing' || !state.myID) return;
    if (state.controllerAttached) return; // phone controller is driving
    // Convert screen-space mouse to world coords, accounting for camera zoom.
    // Screen center = player position; offset from center is scaled by 1/zoom.
    const zoom = state.camZoom;
    let mx = (state.mouseX - state.screenW / 2) / zoom + state.camX;
    let my = (state.mouseY - state.screenH / 2) / zoom + state.camY;

    // Mobile auto-aim (same logic as phone controller)
    if (state.isMobile) {
        const me = state.players.get(state.myID);
        if (me && me.a) {
            // Determine aim direction from joystick offset (or player rotation if idle)
            const jdx = state.mouseX - state.screenW / 2;
            const jdy = state.mouseY - state.screenH / 2;
            const jdist = Math.sqrt(jdx * jdx + jdy * jdy);
            const aimAngle = jdist > 5 ? Math.atan2(jdy, jdx) : me.r;

            // Orbit position in world coords
            const orbitX = me.x + Math.cos(aimAngle) * AIM_ORBIT_R;
            const orbitY = me.y + Math.sin(aimAngle) * AIM_ORBIT_R;

            // Build enemy list
            const enemies = [];
            for (const [id, p] of state.players) {
                if (id === state.myID || !p.a) continue;
                enemies.push({ id: 'p_' + id, x: p.x, y: p.y });
            }
            for (const [id, m] of state.mobs) {
                if (!m.a) continue;
                enemies.push({ id: 'm_' + id, x: m.x, y: m.y });
            }

            // Sticky lock: check if current target still within detection range
            let locked = false;
            if (lockTargetId !== null) {
                const t = enemies.find(e => e.id === lockTargetId);
                if (t) {
                    const dx = t.x - orbitX;
                    const dy = t.y - orbitY;
                    if (dx * dx + dy * dy <= AIM_DETECT_R * AIM_DETECT_R) {
                        locked = true;
                    }
                }
                if (!locked) lockTargetId = null;
            }

            // If not locked, find closest enemy in range
            if (!locked) {
                let bestDist = AIM_DETECT_R * AIM_DETECT_R;
                for (const e of enemies) {
                    const dx = e.x - orbitX;
                    const dy = e.y - orbitY;
                    const d2 = dx * dx + dy * dy;
                    if (d2 <= bestDist) {
                        bestDist = d2;
                        lockTargetId = e.id;
                        locked = true;
                    }
                }
            }

            // Override aim target if locked
            if (locked) {
                const t = enemies.find(e => e.id === lockTargetId);
                if (t) {
                    mx = t.x;
                    my = t.y;
                } else {
                    lockTargetId = null;
                }
            }
        }
    }

    send('input', {
        mx,
        my,
        fire: state.firing,
        boost: state.boosting,
        thresh: Math.min(state.screenW, state.screenH) / (8 * zoom),
    });
}

function startInputLoop() {
    stopInputLoop();
    inputInterval = setInterval(sendInput, 1000 / INPUT_RATE);
}

function stopInputLoop() {
    if (inputInterval) {
        clearInterval(inputInterval);
        inputInterval = null;
    }
}

export function listSessions() {
    send('list', {});
}

export function createSession(name, sessionName) {
    send('create', { name, sname: sessionName });
}

export function joinSession(name, sessionID) {
    send('join', { name, sid: sessionID });
}

export function checkSession(sid) {
    send('check', { sid });
}
