import { state } from './state.js';

export function setupInput() {
    const canvas = state.canvas;

    canvas.addEventListener('mousemove', (e) => {
        const rect = canvas.getBoundingClientRect();
        state.mouseX = e.clientX - rect.left;
        state.mouseY = e.clientY - rect.top;
    });

    document.addEventListener('keydown', (e) => {
        if (state.phase !== 'playing') return;
        if (e.key === 'w' || e.key === 'W') {
            state.firing = true;
        }
        if (e.key === 'Shift') {
            state.boosting = true;
        }
        if (e.key === 'd' || e.key === 'D') {
            state.debugHitboxes = !state.debugHitboxes;
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
