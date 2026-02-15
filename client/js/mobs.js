import { state } from './state.js';
import { drawShip } from './ships.js';
import { drawPlayerHealthBar } from './hud.js';
import { addEngineParticles } from './effects.js';
import { MOB_RADIUS } from './constants.js';

export function renderMobs(ctx, offsetX, offsetY, vw, vh) {
    for (const [, mob] of state.mobs) {
        if (!mob.a) continue;

        const sx = mob.x - offsetX;
        const sy = mob.y - offsetY;

        // Skip if off viewport
        if (sx < -100 || sx > vw + 100 || sy < -100 || sy > vh + 100) continue;

        // Engine particles
        const shipType = mob.s !== undefined ? mob.s : 3;
        const speed = Math.sqrt(mob.vx * mob.vx + mob.vy * mob.vy);
        addEngineParticles(mob.x, mob.y, mob.r, speed, shipType);

        // Draw ship using mob's ship type
        drawShip(ctx, sx, sy, mob.r, shipType, 0.85);

        // Health bar
        drawPlayerHealthBar(ctx, sx, sy, mob.hp, mob.mhp, 'MOB', false);
    }
}
