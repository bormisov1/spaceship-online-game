import { state } from './state.js';
import { RECONNECT_DELAY, INPUT_RATE } from './constants.js';

let ws = null;
let messageHandler = null;
let inputInterval = null;

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
    // Recalculate world coords fresh every call so they stay correct
    // even when the mouse is stationary but the camera moves
    const mx = state.mouseX + state.camX - state.screenW / 2;
    const my = state.mouseY + state.camY - state.screenH / 2;
    send('input', {
        mx,
        my,
        fire: state.firing,
        boost: state.boosting,
        thresh: Math.min(state.screenW, state.screenH) / 8,
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
