// Phone controller mode — joystick + fire pad over WebSocket
import { INPUT_RATE, RECONNECT_DELAY } from './constants.js';

const JOYSTICK_SCALE = 3;
const DEAD_ZONE = 8; // pixels

let ws = null;
let sid = '';
let pid = '';
let connected = false;
let attached = false;

// Player position from state broadcasts (for world-coord conversion)
let playerX = 0;
let playerY = 0;
let playerR = 0;
let screenW = 0;
let screenH = 0;

// Auto-aim state
const AIM_ORBIT_R = 360;
const AIM_DETECT_R = 150;
let enemies = [];       // {id, x, y} from other players + mobs
let lockTargetId = null;

// Joystick state
let joystickTouchId = null;
let joystickStartX = 0;
let joystickStartY = 0;
let joystickDX = 0;
let joystickDY = 0;

// Fire state
let fireTouchId = null;
let firing = false;

// Boost state
let boostTouchId = null;
let boosting = false;
let boostLockedR = null; // rotation locked at boost start

const BOOST_COLUMN_HALF = 50;

let inputInterval = null;

export function initController(sessionID, playerID) {
    sid = sessionID;
    pid = playerID;

    // Hide all normal game UI
    document.getElementById('lobby').style.display = 'none';
    document.getElementById('bgCanvas').style.display = 'none';
    document.getElementById('gameCanvas').style.display = 'none';
    document.getElementById('fullscreenBtn').style.display = 'none';

    // Build controller UI
    buildUI();

    // Handle orientation
    checkOrientation();
    window.addEventListener('resize', checkOrientation);
    if (screen.orientation) {
        screen.orientation.addEventListener('change', checkOrientation);
    }

    // Connect
    connectWS();
}

function buildUI() {
    const el = document.createElement('div');
    el.id = 'controllerRoot';
    el.innerHTML = `
        <div id="ctrlRotateMsg">
            <div class="rotate-icon">
                <svg width="80" height="80" viewBox="0 0 80 80" fill="none" stroke="#6688aa" stroke-width="2">
                    <rect x="20" y="10" width="40" height="60" rx="4" stroke-dasharray="4 2"/>
                    <path d="M50 70 L70 50 L70 30 L30 30 L10 50 L10 70 Z" fill="rgba(50,100,200,0.1)" stroke="#4488ff" stroke-dasharray="4 2"/>
                    <path d="M55 25 C60 15, 70 20, 65 28" stroke="#ffcc00" stroke-width="2" fill="none"/>
                    <path d="M63 22 L65 28 L59 27" stroke="#ffcc00" stroke-width="2" fill="none"/>
                </svg>
            </div>
            <p>Rotate your phone to landscape</p>
        </div>
        <div id="ctrlPad" style="display:none;">
            <div id="ctrlStatus">Connecting...</div>
            <div class="ctrl-divider-left"></div>
            <div class="ctrl-divider-right"></div>
            <div class="ctrl-center">
                <div class="ctrl-boost-indicator" id="boostIndicator"></div>
                <div class="ctrl-label">BOOST</div>
            </div>
            <div class="ctrl-left">
                <div class="ctrl-label">Drag to navigate</div>
                <div class="ctrl-joystick-ring" id="joystickRing">
                    <div class="ctrl-joystick-knob" id="joystickKnob"></div>
                </div>
            </div>
            <div class="ctrl-right">
                <div class="ctrl-label">Tap to fire</div>
                <div class="ctrl-fire-indicator" id="fireIndicator"></div>
            </div>
        </div>
    `;
    document.body.appendChild(el);

    // Add styles
    const style = document.createElement('style');
    style.textContent = `
        #controllerRoot {
            position: fixed; top: 0; left: 0; width: 100%; height: 100%;
            z-index: 100; background: #0a0a1a;
            font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif;
            color: #fff;
            touch-action: none;
            -webkit-user-select: none;
            user-select: none;
        }
        #ctrlRotateMsg {
            display: flex; flex-direction: column;
            align-items: center; justify-content: center;
            height: 100%;
            text-align: center;
        }
        #ctrlRotateMsg p {
            color: #6688aa; font-size: 18px; margin-top: 20px;
        }
        .rotate-icon { margin-bottom: 10px; }
        #ctrlPad {
            width: 100%; height: 100%; position: relative;
            overflow: hidden;
        }
        #ctrlStatus {
            position: absolute; top: 8px; left: 50%; transform: translateX(-50%);
            font-size: 12px; color: #556677; z-index: 2;
            letter-spacing: 2px; text-transform: uppercase;
        }
        .ctrl-divider-left, .ctrl-divider-right {
            position: absolute; top: 10%; bottom: 10%;
            width: 0;
            border-left: 2px dashed rgba(255,255,255,0.12);
            z-index: 1;
        }
        .ctrl-divider-left { left: calc(50% - 50px); }
        .ctrl-divider-right { left: calc(50% + 50px); }
        .ctrl-center {
            position: absolute; top: 0; bottom: 0;
            left: calc(50% - 50px); width: 100px;
            display: flex; flex-direction: column;
            align-items: center; justify-content: center;
            z-index: 1; pointer-events: none;
        }
        .ctrl-boost-indicator {
            width: 40px; height: 40px;
            border: 2px solid rgba(100, 180, 255, 0.2);
            border-radius: 50%;
            pointer-events: none;
            transition: background 0.1s, border-color 0.1s;
            margin-bottom: 8px;
        }
        .ctrl-boost-indicator.active {
            background: rgba(80, 160, 255, 0.4);
            border-color: rgba(100, 200, 255, 0.8);
        }
        .ctrl-left, .ctrl-right {
            position: absolute; top: 0; bottom: 0; width: calc(50% - 50px);
            display: flex; flex-direction: column;
            align-items: center; justify-content: center;
        }
        .ctrl-left { left: 0; }
        .ctrl-right { right: 0; }
        .ctrl-label {
            color: #334455; font-size: 13px; text-transform: uppercase;
            letter-spacing: 2px; margin-bottom: 20px;
            pointer-events: none;
        }
        .ctrl-joystick-ring {
            width: 140px; height: 140px;
            border: 2px solid rgba(255,255,255,0.1);
            border-radius: 50%;
            position: relative;
            pointer-events: none;
        }
        .ctrl-joystick-knob {
            width: 50px; height: 50px;
            background: rgba(68, 136, 255, 0.3);
            border: 2px solid rgba(68, 136, 255, 0.5);
            border-radius: 50%;
            position: absolute;
            top: 50%; left: 50%;
            transform: translate(-50%, -50%);
            transition: background 0.1s;
            pointer-events: none;
        }
        .ctrl-fire-indicator {
            width: 100px; height: 100px;
            border: 2px solid rgba(255,68,68,0.2);
            border-radius: 50%;
            pointer-events: none;
            transition: background 0.1s, border-color 0.1s;
        }
        .ctrl-fire-indicator.active {
            background: rgba(255,68,68,0.3);
            border-color: rgba(255,68,68,0.6);
        }
    `;
    document.head.appendChild(style);

    // Touch handlers on the pad
    const pad = document.getElementById('ctrlPad');
    pad.addEventListener('touchstart', onTouchStart, { passive: false });
    pad.addEventListener('touchmove', onTouchMove, { passive: false });
    pad.addEventListener('touchend', onTouchEnd, { passive: false });
    pad.addEventListener('touchcancel', onTouchEnd, { passive: false });
}

function checkOrientation() {
    screenW = window.innerWidth;
    screenH = window.innerHeight;
    const landscape = screenW > screenH;
    const rotateMsg = document.getElementById('ctrlRotateMsg');
    const pad = document.getElementById('ctrlPad');
    if (rotateMsg && pad) {
        rotateMsg.style.display = landscape ? 'none' : 'flex';
        pad.style.display = landscape ? 'block' : 'none';
    }
}

function connectWS() {
    const proto = location.protocol === 'https:' ? 'wss:' : 'ws:';
    const url = `${proto}//${location.host}/ws`;
    ws = new WebSocket(url);

    ws.onopen = () => {
        connected = true;
        updateStatus('Attaching...');
        ws.send(JSON.stringify({ t: 'control', d: { sid, pid } }));
    };

    ws.onmessage = (event) => {
        try {
            const msg = JSON.parse(event.data);
            handleMessage(msg);
        } catch (e) { /* ignore */ }
    };

    ws.onclose = () => {
        connected = false;
        attached = false;
        stopInputLoop();
        updateStatus('Disconnected. Reconnecting...');
        setTimeout(connectWS, RECONNECT_DELAY);
    };

    ws.onerror = () => {};
}

function handleMessage(msg) {
    switch (msg.t) {
        case 'control_ok':
            attached = true;
            updateStatus('Connected');
            startInputLoop();
            break;
        case 'state':
            handleState(msg.d);
            break;
        case 'error':
            updateStatus('Error: ' + msg.d.msg);
            break;
    }
}

function handleState(data) {
    const newEnemies = [];

    if (data.p) {
        for (const p of data.p) {
            if (p.id === pid) {
                playerX = p.x;
                playerY = p.y;
                playerR = p.r;
            } else if (p.a) {
                newEnemies.push({ id: 'p_' + p.id, x: p.x, y: p.y });
            }
        }
    }

    if (data.m) {
        for (const m of data.m) {
            if (m.a) {
                newEnemies.push({ id: 'm_' + m.id, x: m.x, y: m.y });
            }
        }
    }

    enemies = newEnemies;
}

function updateStatus(text) {
    const el = document.getElementById('ctrlStatus');
    if (el) el.textContent = text;
}

// --- Touch handling ---

function onTouchStart(e) {
    e.preventDefault();
    const halfW = screenW / 2;
    const centerLeft = halfW - BOOST_COLUMN_HALF;
    const centerRight = halfW + BOOST_COLUMN_HALF;
    for (const touch of e.changedTouches) {
        const cx = touch.clientX;
        if (cx < centerLeft && joystickTouchId === null) {
            joystickTouchId = touch.identifier;
            joystickStartX = cx;
            joystickStartY = touch.clientY;
            joystickDX = 0;
            joystickDY = 0;
        } else if (cx > centerRight && fireTouchId === null) {
            fireTouchId = touch.identifier;
            firing = true;
            const ind = document.getElementById('fireIndicator');
            if (ind) ind.classList.add('active');
        } else if (cx >= centerLeft && cx <= centerRight && boostTouchId === null) {
            boostTouchId = touch.identifier;
            boosting = true;
            boostLockedR = playerR; // lock current heading
            const ind = document.getElementById('boostIndicator');
            if (ind) ind.classList.add('active');
        }
    }
}

function onTouchMove(e) {
    e.preventDefault();
    for (const touch of e.changedTouches) {
        if (touch.identifier === joystickTouchId) {
            joystickDX = touch.clientX - joystickStartX;
            joystickDY = touch.clientY - joystickStartY;
            updateKnob();
        }
    }
}

function onTouchEnd(e) {
    e.preventDefault();
    for (const touch of e.changedTouches) {
        if (touch.identifier === joystickTouchId) {
            joystickTouchId = null;
            joystickDX = 0;
            joystickDY = 0;
            updateKnob();
        }
        if (touch.identifier === fireTouchId) {
            fireTouchId = null;
            firing = false;
            const ind = document.getElementById('fireIndicator');
            if (ind) ind.classList.remove('active');
        }
        if (touch.identifier === boostTouchId) {
            boostTouchId = null;
            boosting = false;
            boostLockedR = null;
            const ind = document.getElementById('boostIndicator');
            if (ind) ind.classList.remove('active');
        }
    }
}

function updateKnob() {
    const knob = document.getElementById('joystickKnob');
    if (!knob) return;
    // Clamp to ring radius (70px)
    const maxR = 45;
    let dx = joystickDX;
    let dy = joystickDY;
    const dist = Math.sqrt(dx * dx + dy * dy);
    if (dist > maxR) {
        dx = (dx / dist) * maxR;
        dy = (dy / dist) * maxR;
    }
    knob.style.transform = `translate(calc(-50% + ${dx}px), calc(-50% + ${dy}px))`;
}

// --- Input sending ---

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

function sendInput() {
    if (!ws || ws.readyState !== WebSocket.OPEN || !attached) return;

    // Determine aim direction from joystick (or ship rotation if idle)
    const dist = Math.sqrt(joystickDX * joystickDX + joystickDY * joystickDY);
    let aimAngle = playerR;
    if (dist > DEAD_ZONE) {
        aimAngle = Math.atan2(joystickDY, joystickDX);
    }

    let mx, my;
    let locked = false;

    // Only run auto-aim when joystick is actively being used
    if (dist > DEAD_ZONE) {
        // Orbit position in world coords
        const orbitX = playerX + Math.cos(aimAngle) * AIM_ORBIT_R;
        const orbitY = playerY + Math.sin(aimAngle) * AIM_ORBIT_R;

        // Sticky lock: check if current target still within detection range
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

        if (locked) {
            const t = enemies.find(e => e.id === lockTargetId);
            if (t) {
                mx = t.x;
                my = t.y;
            } else {
                lockTargetId = null;
                locked = false;
            }
        }

        if (!locked) {
            mx = playerX + joystickDX * JOYSTICK_SCALE;
            my = playerY + joystickDY * JOYSTICK_SCALE;
        }
    } else {
        // Joystick idle: maintain current heading, clear lock
        lockTargetId = null;
        mx = playerX;
        my = playerY;
    }

    // During boost, lock steering to the direction captured at boost start
    if (boosting && boostLockedR !== null) {
        mx = playerX + Math.cos(boostLockedR) * 1000;
        my = playerY + Math.sin(boostLockedR) * 1000;
    }

    ws.send(JSON.stringify({
        t: 'input',
        d: {
            mx,
            my,
            fire: firing,
            boost: boosting,
            thresh: 50,
        }
    }));
}
