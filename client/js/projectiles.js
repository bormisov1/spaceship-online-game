import { state } from './state.js';
import { LASER_COLORS, PROJECTILE_RADIUS } from './constants.js';

export function renderProjectiles(ctx, offsetX, offsetY) {
    for (const [, proj] of state.projectiles) {
        const sx = proj.x - offsetX;
        const sy = proj.y - offsetY;

        // Skip if off screen
        if (sx < -50 || sx > state.screenW + 50 || sy < -50 || sy > state.screenH + 50) continue;

        // Determine color from owner's ship type
        let color = LASER_COLORS[0];
        const owner = state.players.get(proj.o);
        if (owner) {
            color = LASER_COLORS[owner.s] || LASER_COLORS[0];
        }

        // Draw glow
        const glowSize = PROJECTILE_RADIUS * 6;
        const grad = ctx.createRadialGradient(sx, sy, 0, sx, sy, glowSize);
        grad.addColorStop(0, color + '66');
        grad.addColorStop(1, color + '00');
        ctx.fillStyle = grad;
        ctx.fillRect(sx - glowSize, sy - glowSize, glowSize * 2, glowSize * 2);

        // Draw laser bolt
        ctx.save();
        ctx.translate(sx, sy);
        ctx.rotate(proj.r);

        // Elongated laser shape
        ctx.fillStyle = '#ffffff';
        ctx.beginPath();
        ctx.ellipse(0, 0, PROJECTILE_RADIUS * 3, PROJECTILE_RADIUS * 0.8, 0, 0, Math.PI * 2);
        ctx.fill();

        ctx.fillStyle = color;
        ctx.beginPath();
        ctx.ellipse(0, 0, PROJECTILE_RADIUS * 2.5, PROJECTILE_RADIUS * 0.6, 0, 0, Math.PI * 2);
        ctx.fill();

        ctx.restore();
    }
}
