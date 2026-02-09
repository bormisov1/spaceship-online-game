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

    // Camera offset (world coords -> screen coords)
    const offsetX = state.camX - w / 2;
    const offsetY = state.camY - h / 2;

    // Render background stars
    renderStarfield();

    // Clear foreground
    ctx.clearRect(0, 0, w, h);

    // Draw world boundary
    drawWorldBounds(ctx, offsetX, offsetY);

    // Update and render particles
    updateParticles(dt);
    renderParticles(ctx, offsetX, offsetY);
    renderExplosions(ctx, offsetX, offsetY);

    // Render projectiles
    renderProjectiles(ctx, offsetX, offsetY);

    // Render players
    for (const [id, player] of state.players) {
        if (!player.a) continue;

        const sx = player.x - offsetX;
        const sy = player.y - offsetY;

        // Skip if off screen (with margin)
        if (sx < -100 || sx > w + 100 || sy < -100 || sy > h + 100) continue;

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
        drawHitboxes(ctx, offsetX, offsetY);
    }

    // HUD overlay
    renderHUD(ctx);
}

function drawHitboxes(ctx, offsetX, offsetY) {
    // Player hitboxes
    for (const [, player] of state.players) {
        if (!player.a) continue;
        const sx = player.x - offsetX;
        const sy = player.y - offsetY;
        if (sx < -100 || sx > state.screenW + 100 || sy < -100 || sy > state.screenH + 100) continue;

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
        if (sx < -50 || sx > state.screenW + 50 || sy < -50 || sy > state.screenH + 50) continue;

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
