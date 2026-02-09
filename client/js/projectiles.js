import { state } from './state.js';
import { LASER_COLORS, PROJECTILE_RADIUS } from './constants.js';

// Cached glow sprites per laser color
const glowSprites = new Map();
const GLOW_SIZE = PROJECTILE_RADIUS * 6;
const SPRITE_DIM = GLOW_SIZE * 2;

function getGlowSprite(color) {
    let sprite = glowSprites.get(color);
    if (sprite) return sprite;

    // Create offscreen canvas with the radial gradient glow
    const canvas = document.createElement('canvas');
    canvas.width = SPRITE_DIM;
    canvas.height = SPRITE_DIM;
    const ctx = canvas.getContext('2d');

    const cx = GLOW_SIZE;
    const cy = GLOW_SIZE;
    const grad = ctx.createRadialGradient(cx, cy, 0, cx, cy, GLOW_SIZE);
    grad.addColorStop(0, color + '66');
    grad.addColorStop(1, color + '00');
    ctx.fillStyle = grad;
    ctx.fillRect(0, 0, SPRITE_DIM, SPRITE_DIM);

    glowSprites.set(color, canvas);
    return canvas;
}

export function renderProjectiles(ctx, offsetX, offsetY, vw, vh) {
    for (const [, proj] of state.projectiles) {
        const sx = proj.x - offsetX;
        const sy = proj.y - offsetY;

        // Skip if off viewport
        if (sx < -50 || sx > vw + 50 || sy < -50 || sy > vh + 50) continue;

        // Determine color from owner's ship type
        let color = LASER_COLORS[0];
        const owner = state.players.get(proj.o);
        if (owner) {
            color = LASER_COLORS[owner.s] || LASER_COLORS[0];
        }

        // Draw cached glow sprite
        const sprite = getGlowSprite(color);
        ctx.drawImage(sprite, sx - GLOW_SIZE, sy - GLOW_SIZE);

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
