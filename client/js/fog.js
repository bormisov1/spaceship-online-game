import { WORLD_W, WORLD_H } from './constants.js';

let fogCanvas = null;

export function initFog() {
    // Build offscreen canvas with fog patches at fixed world positions
    fogCanvas = document.createElement('canvas');
    fogCanvas.width = WORLD_W;
    fogCanvas.height = WORLD_H;
    const ctx = fogCanvas.getContext('2d');

    // 8 fog patches at fixed positions
    const patches = [
        { x: 600, y: 500, r: 400, color: [40, 20, 80] },
        { x: 1500, y: 800, r: 350, color: [20, 40, 90] },
        { x: 3200, y: 600, r: 450, color: [30, 15, 70] },
        { x: 800, y: 2500, r: 380, color: [15, 30, 85] },
        { x: 2000, y: 2000, r: 500, color: [25, 25, 75] },
        { x: 3400, y: 2800, r: 420, color: [35, 20, 80] },
        { x: 1200, y: 3500, r: 360, color: [20, 35, 90] },
        { x: 3000, y: 3600, r: 400, color: [30, 25, 85] },
    ];

    for (const p of patches) {
        const grad = ctx.createRadialGradient(p.x, p.y, 0, p.x, p.y, p.r);
        const [r, g, b] = p.color;
        grad.addColorStop(0, `rgba(${r}, ${g}, ${b}, 0.06)`);
        grad.addColorStop(0.5, `rgba(${r}, ${g}, ${b}, 0.03)`);
        grad.addColorStop(1, `rgba(${r}, ${g}, ${b}, 0)`);
        ctx.fillStyle = grad;
        ctx.beginPath();
        ctx.arc(p.x, p.y, p.r, 0, Math.PI * 2);
        ctx.fill();
    }
}

export function renderFog(ctx, offsetX, offsetY) {
    if (!fogCanvas) return;
    ctx.drawImage(fogCanvas, -offsetX, -offsetY);
}
