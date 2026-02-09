import { state } from './state.js';
import { renderStarfield } from './starfield.js';
import { drawShip, initShips } from './ships.js';
import { renderProjectiles } from './projectiles.js';
import { updateParticles, renderParticles, renderExplosions, addEngineParticles } from './effects.js';
import { renderHUD, drawPlayerHealthBar } from './hud.js';
import { WORLD_W, WORLD_H, PLAYER_RADIUS, PROJECTILE_RADIUS } from './constants.js';

let shipsInited = false;

export function render(dt) {
    if (!shipsInited) {
        initShips();
        shipsInited = true;
    }

    const ctx = state.ctx;
    const w = state.screenW;
    const h = state.screenH;
    const zoom = state.camZoom;

    // Virtual viewport in world units (what the camera sees)
    const vw = w / zoom;
    const vh = h / zoom;

    // Camera offset in world coords
    const offsetX = state.camX - vw / 2;
    const offsetY = state.camY - vh / 2;

    // Render background stars (screen-space, no zoom)
    renderStarfield();

    // Clear foreground
    ctx.clearRect(0, 0, w, h);

    // Apply zoom transform for all world rendering
    ctx.save();
    ctx.scale(zoom, zoom);

    // Draw world boundary
    drawWorldBounds(ctx, offsetX, offsetY);

    // Update and render particles
    updateParticles(dt);
    renderParticles(ctx, offsetX, offsetY, vw, vh);
    renderExplosions(ctx, offsetX, offsetY, vw, vh);

    // Render projectiles
    renderProjectiles(ctx, offsetX, offsetY, vw, vh);

    // Render players
    for (const [id, player] of state.players) {
        if (!player.a) continue;

        const sx = player.x - offsetX;
        const sy = player.y - offsetY;

        // Skip if off virtual viewport (with margin)
        if (sx < -100 || sx > vw + 100 || sy < -100 || sy > vh + 100) continue;

        // Engine glow particles
        const speed = Math.sqrt(player.vx * player.vx + player.vy * player.vy);
        addEngineParticles(player.x, player.y, player.r, speed, player.s);

        // Draw ship
        const isMe = id === state.myID;
        const alpha = isMe ? 1 : 0.95;
        drawShip(ctx, sx, sy, player.r, player.s, alpha);

        // Health bar + name for other players
        drawPlayerHealthBar(ctx, sx, sy, player.hp, player.mhp, player.n, isMe);
    }

    // Debug hitboxes
    if (state.debugHitboxes) {
        drawHitboxes(ctx, offsetX, offsetY, vw, vh);
    }

    ctx.restore(); // Remove zoom transform

    // HUD overlay (screen coords, no zoom)
    renderHUD(ctx);
}

function drawHitboxes(ctx, offsetX, offsetY, vw, vh) {
    // Player hitboxes
    for (const [, player] of state.players) {
        if (!player.a) continue;
        const sx = player.x - offsetX;
        const sy = player.y - offsetY;
        if (sx < -100 || sx > vw + 100 || sy < -100 || sy > vh + 100) continue;

        ctx.beginPath();
        ctx.arc(sx, sy, PLAYER_RADIUS, 0, Math.PI * 2);
        ctx.fillStyle = 'rgba(255, 255, 0, 0.15)';
        ctx.fill();
        ctx.strokeStyle = 'rgba(255, 255, 0, 0.6)';
        ctx.lineWidth = 1;
        ctx.stroke();
    }

    // Projectile hitboxes
    for (const [, proj] of state.projectiles) {
        const sx = proj.x - offsetX;
        const sy = proj.y - offsetY;
        if (sx < -50 || sx > vw + 50 || sy < -50 || sy > vh + 50) continue;

        ctx.beginPath();
        ctx.arc(sx, sy, PROJECTILE_RADIUS, 0, Math.PI * 2);
        ctx.fillStyle = 'rgba(255, 0, 0, 0.2)';
        ctx.fill();
        ctx.strokeStyle = 'rgba(255, 0, 0, 0.7)';
        ctx.lineWidth = 1;
        ctx.stroke();
    }
}

function drawWorldBounds(ctx, offsetX, offsetY) {
    // Draw boundary indicators when near edges
    ctx.strokeStyle = 'rgba(255, 100, 100, 0.3)';
    ctx.lineWidth = 2;
    ctx.setLineDash([10, 10]);

    const bx = -offsetX;
    const by = -offsetY;
    const bw = WORLD_W;
    const bh = WORLD_H;

    ctx.strokeRect(bx, by, bw, bh);
    ctx.setLineDash([]);
}
