import { state } from './state.js';
import { ASTEROID_RADIUS, ASTEROID_RENDER_SIZE } from './constants.js';

let asteroidSprite = null;

export function initAsteroid() {
    const img = new Image();
    img.onload = () => {
        const size = ASTEROID_RENDER_SIZE;
        const c = document.createElement('canvas');
        c.width = size;
        c.height = size;
        const ctx = c.getContext('2d');
        ctx.drawImage(img, 0, 0, size, size);
        asteroidSprite = c;
    };
    img.src = 'assets/asteroid.png';
}

export function renderAsteroids(ctx, offsetX, offsetY, vw, vh) {
    if (!asteroidSprite) return;

    for (const [, ast] of state.asteroids) {
        const sx = ast.x - offsetX;
        const sy = ast.y - offsetY;

        // Skip if off viewport
        if (sx < -ASTEROID_RENDER_SIZE || sx > vw + ASTEROID_RENDER_SIZE ||
            sy < -ASTEROID_RENDER_SIZE || sy > vh + ASTEROID_RENDER_SIZE) continue;

        ctx.save();
        ctx.translate(sx, sy);
        ctx.rotate(ast.r);
        const half = ASTEROID_RENDER_SIZE / 2;
        ctx.drawImage(asteroidSprite, -half, -half, ASTEROID_RENDER_SIZE, ASTEROID_RENDER_SIZE);
        ctx.restore();
    }
}
