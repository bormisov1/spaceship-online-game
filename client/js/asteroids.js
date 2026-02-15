import { state } from './state.js';
import { ASTEROID_RADIUS, ASTEROID_RENDER_SIZE } from './constants.js';

const ASTEROID_FILES = [
    'assets/asteroid-1.png',
    'assets/asteroid-2.png',
    'assets/asteroid-3.png',
    'assets/asteroid-4.png',
];

const asteroidSprites = [];
let loaded = false;

export function initAsteroid() {
    ASTEROID_FILES.forEach((src, i) => {
        const img = new Image();
        img.onload = () => {
            const size = ASTEROID_RENDER_SIZE;
            const c = document.createElement('canvas');
            c.width = size;
            c.height = size;
            const ctx = c.getContext('2d');
            ctx.drawImage(img, 0, 0, size, size);
            asteroidSprites[i] = c;
            if (asteroidSprites.filter(Boolean).length === ASTEROID_FILES.length) loaded = true;
        };
        img.src = src;
    });
}

// Simple hash of asteroid ID to pick a variant
function idToVariant(id) {
    let h = 0;
    for (let i = 0; i < id.length; i++) {
        h = (h * 31 + id.charCodeAt(i)) | 0;
    }
    return ((h % ASTEROID_FILES.length) + ASTEROID_FILES.length) % ASTEROID_FILES.length;
}

export function renderAsteroids(ctx, offsetX, offsetY, vw, vh) {
    if (!loaded) return;

    for (const [id, ast] of state.asteroids) {
        const sx = ast.x - offsetX;
        const sy = ast.y - offsetY;

        // Skip if off viewport
        if (sx < -ASTEROID_RENDER_SIZE || sx > vw + ASTEROID_RENDER_SIZE ||
            sy < -ASTEROID_RENDER_SIZE || sy > vh + ASTEROID_RENDER_SIZE) continue;

        const variant = idToVariant(id);
        const sprite = asteroidSprites[variant];
        if (!sprite) continue;

        ctx.save();
        ctx.translate(sx, sy);
        ctx.rotate(ast.r);
        const half = ASTEROID_RENDER_SIZE / 2;
        ctx.drawImage(sprite, -half, -half, ASTEROID_RENDER_SIZE, ASTEROID_RENDER_SIZE);
        ctx.restore();
    }
}
