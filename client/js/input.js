import { state } from './state.js';

export function setupInput() {
    const canvas = state.canvas;

    // Detect mobile device
    state.isMobile = ('ontouchstart' in window) || (navigator.maxTouchPoints > 0);

    if (state.isMobile) {
        setupTouchInput(canvas);
        // Initialize mouse to screen center so ship stays in dead zone
        state.mouseX = state.screenW / 2;
        state.mouseY = state.screenH / 2;
        // Prevent document-level scrolling/bounce
        document.addEventListener('touchmove', (e) => e.preventDefault(), { passive: false });
    }

    canvas.addEventListener('mousemove', (e) => {
        if (state.isMobile) return;
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

    // Mouse click for firing (desktop only)
    canvas.addEventListener('mousedown', (e) => {
        if (state.isMobile) return;
        if (state.phase !== 'playing') return;
        if (e.button === 0) {
            state.firing = true;
        }
    });

    canvas.addEventListener('mouseup', (e) => {
        if (state.isMobile) return;
        if (e.button === 0) {
            state.firing = false;
        }
    });

    // Prevent context menu on right click
    canvas.addEventListener('contextmenu', (e) => e.preventDefault());
}

function setupTouchInput(canvas) {
    const JOYSTICK_SCALE = 2.5;
    let joystickTouchId = null;
    let joystickStartX = 0;
    let joystickStartY = 0;
    let fireTouchId = null;

    canvas.addEventListener('touchstart', (e) => {
        e.preventDefault();
        if (state.phase !== 'playing') return;

        for (const touch of e.changedTouches) {
            const halfW = state.screenW / 2;

            if (touch.clientX < halfW && joystickTouchId === null) {
                // Left half - virtual joystick
                joystickTouchId = touch.identifier;
                joystickStartX = touch.clientX;
                joystickStartY = touch.clientY;
                state.touchJoystick = {
                    startX: joystickStartX,
                    startY: joystickStartY,
                    currentX: joystickStartX,
                    currentY: joystickStartY,
                };
                // Start at center (no movement yet)
                state.mouseX = state.screenW / 2;
                state.mouseY = state.screenH / 2;
            } else if (touch.clientX >= halfW && fireTouchId === null) {
                // Right half - fire weapon
                fireTouchId = touch.identifier;
                state.firing = true;
            }
        }
    }, { passive: false });

    canvas.addEventListener('touchmove', (e) => {
        e.preventDefault();

        for (const touch of e.changedTouches) {
            if (touch.identifier === joystickTouchId && state.touchJoystick) {
                state.touchJoystick.currentX = touch.clientX;
                state.touchJoystick.currentY = touch.clientY;

                const dx = touch.clientX - joystickStartX;
                const dy = touch.clientY - joystickStartY;

                // Offset mouse from screen center by joystick delta (scaled)
                // This translates to world coords in sendInput naturally:
                // mx = screenW/2 + dx*scale + camX - screenW/2 = camX + dx*scale
                state.mouseX = state.screenW / 2 + dx * JOYSTICK_SCALE;
                state.mouseY = state.screenH / 2 + dy * JOYSTICK_SCALE;
            }
        }
    }, { passive: false });

    const handleTouchEnd = (e) => {
        e.preventDefault();

        for (const touch of e.changedTouches) {
            if (touch.identifier === joystickTouchId) {
                joystickTouchId = null;
                state.touchJoystick = null;
                // Reset to center = dead zone = ship stops
                state.mouseX = state.screenW / 2;
                state.mouseY = state.screenH / 2;
            }
            if (touch.identifier === fireTouchId) {
                fireTouchId = null;
                state.firing = false;
            }
        }
    };

    canvas.addEventListener('touchend', handleTouchEnd, { passive: false });
    canvas.addEventListener('touchcancel', handleTouchEnd, { passive: false });
}
