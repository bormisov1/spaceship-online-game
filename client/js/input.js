import { state } from './state.js';

export function setupInput() {
    const canvas = state.canvas;

    canvas.addEventListener('mousemove', (e) => {
        const rect = canvas.getBoundingClientRect();
        state.mouseX = e.clientX - rect.left;
        state.mouseY = e.clientY - rect.top;
        // Convert to world coords
        state.mouseWorldX = state.mouseX + state.camX - state.screenW / 2;
        state.mouseWorldY = state.mouseY + state.camY - state.screenH / 2;
    });

    document.addEventListener('keydown', (e) => {
        if (state.phase !== 'playing') return;
        if (e.key === 'w' || e.key === 'W') {
            state.firing = true;
        }
        if (e.key === 'Shift') {
            state.boosting = true;
        }
    });

    document.addEventListener('keyup', (e) => {
        if (e.key === 'w' || e.key === 'W') {
            state.firing = false;
        }
        if (e.key === 'Shift') {
            state.boosting = false;
        }
    });

    // Also support mouse click for firing
    canvas.addEventListener('mousedown', (e) => {
        if (state.phase !== 'playing') return;
        if (e.button === 0) {
            state.firing = true;
        }
    });

    canvas.addEventListener('mouseup', (e) => {
        if (e.button === 0) {
            state.firing = false;
        }
    });

    // Prevent context menu on right click
    canvas.addEventListener('contextmenu', (e) => e.preventDefault());
}
