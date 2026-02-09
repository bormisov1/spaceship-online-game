import { state } from './state.js';
import { STAR_COUNT, PARALLAX_FACTOR, WORLD_W, WORLD_H } from './constants.js';

let stars = [];
let lastCamX = -1;
let lastCamY = -1;

export function initStarfield() {
    stars = [];
    for (let i = 0; i < STAR_COUNT; i++) {
        stars.push({
            x: Math.random() * WORLD_W * 1.5,
            y: Math.random() * WORLD_H * 1.5,
            size: Math.random() * 2 + 0.5,
            brightness: Math.random() * 0.7 + 0.3,
            layer: Math.random() < 0.3 ? 2 : (Math.random() < 0.5 ? 1 : 0),
        });
    }
}

export function renderStarfield() {
    const ctx = state.bgCtx;
    const w = state.screenW;
    const h = state.screenH;
    const cx = state.camX;
    const cy = state.camY;

    // Only redraw if camera moved enough
    const dx = Math.abs(cx - lastCamX);
    const dy = Math.abs(cy - lastCamY);
    if (dx < 2 && dy < 2 && lastCamX !== -1) return;
    lastCamX = cx;
    lastCamY = cy;

    ctx.fillStyle = '#0a0a1a';
    ctx.fillRect(0, 0, w, h);

    const factors = [PARALLAX_FACTOR * 0.3, PARALLAX_FACTOR * 0.6, PARALLAX_FACTOR];

    for (const star of stars) {
        const factor = factors[star.layer];
        const sx = ((star.x - cx * factor) % w + w) % w;
        const sy = ((star.y - cy * factor) % h + h) % h;

        const alpha = star.brightness;
        ctx.fillStyle = `rgba(255, 255, 255, ${alpha})`;
        ctx.beginPath();
        ctx.arc(sx, sy, star.size, 0, Math.PI * 2);
        ctx.fill();
    }

    // Add a few colored nebula spots for atmosphere
    drawNebula(ctx, w, h, cx, cy);
}

function drawNebula(ctx, w, h, cx, cy) {
    const nebulae = [
        { x: 1000, y: 1000, r: 200, color: '40, 20, 80' },
        { x: 3000, y: 2000, r: 150, color: '80, 20, 40' },
        { x: 2000, y: 3500, r: 180, color: '20, 40, 80' },
    ];

    for (const n of nebulae) {
        const factor = PARALLAX_FACTOR * 0.2;
        const sx = n.x - cx * factor - (cx - w / 2);
        const sy = n.y - cy * factor - (cy - h / 2);

        // Only draw if on screen
        if (sx < -n.r * 2 || sx > w + n.r * 2 || sy < -n.r * 2 || sy > h + n.r * 2) continue;

        const grad = ctx.createRadialGradient(sx, sy, 0, sx, sy, n.r);
        grad.addColorStop(0, `rgba(${n.color}, 0.08)`);
        grad.addColorStop(1, `rgba(${n.color}, 0)`);
        ctx.fillStyle = grad;
        ctx.fillRect(sx - n.r, sy - n.r, n.r * 2, n.r * 2);
    }
}
