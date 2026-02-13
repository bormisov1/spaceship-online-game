import { state } from './state.js';
import { PICKUP_RADIUS, PICKUP_RENDER_SIZE } from './constants.js';

// Procedural pulsing green glow (no external asset needed)
export function initPickups() {
    // No-op for now; purely procedural rendering
}

export function renderPickups(ctx, offsetX, offsetY, vw, vh) {
    const time = performance.now() / 1000;

    for (const [, pk] of state.pickups) {
        const sx = pk.x - offsetX;
        const sy = pk.y - offsetY;

        // Skip if off viewport
        if (sx < -50 || sx > vw + 50 || sy < -50 || sy > vh + 50) continue;

        // Pulsing glow
        const pulse = 0.7 + 0.3 * Math.sin(time * 3 + pk.x * 0.01);
        const size = PICKUP_RENDER_SIZE * pulse;

        // Outer glow
        const grad = ctx.createRadialGradient(sx, sy, 0, sx, sy, size * 1.5);
        grad.addColorStop(0, `rgba(0, 255, 100, ${0.4 * pulse})`);
        grad.addColorStop(0.5, `rgba(0, 200, 80, ${0.15 * pulse})`);
        grad.addColorStop(1, 'rgba(0, 150, 60, 0)');
        ctx.fillStyle = grad;
        ctx.beginPath();
        ctx.arc(sx, sy, size * 1.5, 0, Math.PI * 2);
        ctx.fill();

        // Core
        const coreGrad = ctx.createRadialGradient(sx, sy, 0, sx, sy, size * 0.5);
        coreGrad.addColorStop(0, '#ffffff');
        coreGrad.addColorStop(0.4, '#88ffaa');
        coreGrad.addColorStop(1, '#00cc44');
        ctx.fillStyle = coreGrad;
        ctx.beginPath();
        ctx.arc(sx, sy, size * 0.5, 0, Math.PI * 2);
        ctx.fill();

        // Cross sparkle
        ctx.strokeStyle = `rgba(150, 255, 200, ${0.5 * pulse})`;
        ctx.lineWidth = 1.5;
        const sparkLen = size * 0.8;
        ctx.beginPath();
        ctx.moveTo(sx - sparkLen, sy);
        ctx.lineTo(sx + sparkLen, sy);
        ctx.moveTo(sx, sy - sparkLen);
        ctx.lineTo(sx, sy + sparkLen);
        ctx.stroke();
    }
}
