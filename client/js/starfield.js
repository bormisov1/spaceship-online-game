import { state } from './state.js';
import { STAR_COUNT, PARALLAX_FACTOR, WORLD_W, WORLD_H } from './constants.js';

let stars = [];
let lastCamX = -1;
let lastCamY = -1;

// Offscreen canvases for each parallax layer + nebula
let layerCanvases = []; // indices 0, 1, 2 for star layers
let nebulaCanvas = null;
const LAYER_FACTORS = [PARALLAX_FACTOR * 0.3, PARALLAX_FACTOR * 0.6, PARALLAX_FACTOR];
const NEBULA_FACTOR = PARALLAX_FACTOR * 0.2;

// Dimensions of the offscreen canvases (screen size, for tiling)
let cachedW = 0;
let cachedH = 0;

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
    // Force rebuild on next render
    cachedW = 0;
    cachedH = 0;
}

function buildOffscreenCanvases(w, h) {
    cachedW = w;
    cachedH = h;
    const minDim = Math.min(w, h);
    const sizeScale = Math.max(0.4, Math.min(1, minDim / 900));

    // Build one offscreen canvas per star layer
    layerCanvases = [];
    for (let layer = 0; layer < 3; layer++) {
        const offscreen = document.createElement('canvas');
        offscreen.width = w;
        offscreen.height = h;
        const offCtx = offscreen.getContext('2d');

        // Draw stars for this layer
        for (const star of stars) {
            if (star.layer !== layer) continue;
            // Place stars at their position mod canvas size (they tile)
            const sx = ((star.x) % w + w) % w;
            const sy = ((star.y) % h + h) % h;

            offCtx.fillStyle = `rgba(255,255,255,${star.brightness})`;
            offCtx.beginPath();
            offCtx.arc(sx, sy, star.size * sizeScale, 0, Math.PI * 2);
            offCtx.fill();
        }

        layerCanvases.push(offscreen);
    }

    // Build nebula offscreen canvas (large enough to hold all nebulae)
    // We use a canvas the size of the world so nebulae stay in fixed positions
    const nebulae = [
        { x: 1000, y: 1000, r: 200, color: '40, 20, 80' },
        { x: 3000, y: 2000, r: 150, color: '80, 20, 40' },
        { x: 2000, y: 3500, r: 180, color: '20, 40, 80' },
    ];

    nebulaCanvas = document.createElement('canvas');
    // Size to cover the nebulae region (with padding)
    const nebulaW = WORLD_W;
    const nebulaH = WORLD_H;
    nebulaCanvas.width = nebulaW;
    nebulaCanvas.height = nebulaH;
    const nCtx = nebulaCanvas.getContext('2d');

    for (const n of nebulae) {
        const grad = nCtx.createRadialGradient(n.x, n.y, 0, n.x, n.y, n.r);
        grad.addColorStop(0, `rgba(${n.color}, 0.08)`);
        grad.addColorStop(1, `rgba(${n.color}, 0)`);
        nCtx.fillStyle = grad;
        nCtx.fillRect(n.x - n.r, n.y - n.r, n.r * 2, n.r * 2);
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

    // Rebuild offscreen canvases if screen size changed
    if (w !== cachedW || h !== cachedH) {
        buildOffscreenCanvases(w, h);
    }

    // Clear background
    ctx.fillStyle = '#0a0a1a';
    ctx.fillRect(0, 0, w, h);

    // Blit each star layer with parallax offset
    for (let layer = 0; layer < 3; layer++) {
        const factor = LAYER_FACTORS[layer];
        const ox = ((cx * factor) % w + w) % w;
        const oy = ((cy * factor) % h + h) % h;

        // Draw the offscreen canvas 4 times to handle wrapping at edges
        ctx.drawImage(layerCanvases[layer], -ox, -oy);
        ctx.drawImage(layerCanvases[layer], w - ox, -oy);
        ctx.drawImage(layerCanvases[layer], -ox, h - oy);
        ctx.drawImage(layerCanvases[layer], w - ox, h - oy);
    }

    // Blit nebula layer with parallax
    const nebulaOffX = cx * NEBULA_FACTOR + (cx - w / 2);
    const nebulaOffY = cy * NEBULA_FACTOR + (cy - h / 2);
    ctx.drawImage(nebulaCanvas, -nebulaOffX, -nebulaOffY);
}
