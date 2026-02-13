import { state } from './state.js';
import { renderStarfield } from './starfield.js';
import { drawShip, initShips } from './ships.js';
import { renderProjectiles } from './projectiles.js';
import { updateParticles, renderParticles, renderExplosions, addEngineParticles } from './effects.js';
import { renderHUD, drawPlayerHealthBar } from './hud.js';
import { WORLD_W, WORLD_H, PLAYER_RADIUS, PROJECTILE_RADIUS, MOB_RADIUS, ASTEROID_RADIUS, PICKUP_RADIUS } from './constants.js';
import { renderMobs } from './mobs.js';
import { initAsteroid, renderAsteroids } from './asteroids.js';
import { initPickups, renderPickups } from './pickups.js';
import { initFog, renderFog } from './fog.js';

let shipsInited = false;

export function render(dt) {
    if (!shipsInited) {
        initShips();
        initAsteroid();
        initPickups();
        initFog();
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

    // Fog (behind everything)
    renderFog(ctx, offsetX, offsetY);

    // Update and render particles
    updateParticles(dt);
    renderParticles(ctx, offsetX, offsetY, vw, vh);
    renderExplosions(ctx, offsetX, offsetY, vw, vh);

    // Render projectiles
    renderProjectiles(ctx, offsetX, offsetY, vw, vh);

    // Render pickups, asteroids, mobs
    renderPickups(ctx, offsetX, offsetY, vw, vh);
    renderAsteroids(ctx, offsetX, offsetY, vw, vh);
    renderMobs(ctx, offsetX, offsetY, vw, vh);

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

        // Controller aim reticle
        if (isMe && state.controllerAttached) {
            drawControllerAim(ctx, sx, sy, player.r);
        }
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

    // Mob hitboxes (orange)
    for (const [, mob] of state.mobs) {
        if (!mob.a) continue;
        const sx = mob.x - offsetX;
        const sy = mob.y - offsetY;
        if (sx < -100 || sx > vw + 100 || sy < -100 || sy > vh + 100) continue;

        ctx.beginPath();
        ctx.arc(sx, sy, MOB_RADIUS, 0, Math.PI * 2);
        ctx.fillStyle = 'rgba(255, 165, 0, 0.15)';
        ctx.fill();
        ctx.strokeStyle = 'rgba(255, 165, 0, 0.6)';
        ctx.lineWidth = 1;
        ctx.stroke();
    }

    // Asteroid hitboxes (brown)
    for (const [, ast] of state.asteroids) {
        const sx = ast.x - offsetX;
        const sy = ast.y - offsetY;
        if (sx < -150 || sx > vw + 150 || sy < -150 || sy > vh + 150) continue;

        ctx.beginPath();
        ctx.arc(sx, sy, ASTEROID_RADIUS, 0, Math.PI * 2);
        ctx.fillStyle = 'rgba(139, 90, 43, 0.15)';
        ctx.fill();
        ctx.strokeStyle = 'rgba(139, 90, 43, 0.6)';
        ctx.lineWidth = 1;
        ctx.stroke();
    }

    // Pickup hitboxes (green)
    for (const [, pk] of state.pickups) {
        const sx = pk.x - offsetX;
        const sy = pk.y - offsetY;
        if (sx < -50 || sx > vw + 50 || sy < -50 || sy > vh + 50) continue;

        ctx.beginPath();
        ctx.arc(sx, sy, PICKUP_RADIUS, 0, Math.PI * 2);
        ctx.fillStyle = 'rgba(0, 255, 0, 0.15)';
        ctx.fill();
        ctx.strokeStyle = 'rgba(0, 255, 0, 0.6)';
        ctx.lineWidth = 1;
        ctx.stroke();
    }
}

const AIM_ORBIT_R = 300;  // world units from ship center
const AIM_CIRCLE_R = 8;   // small circle radius
const AIM_LINE_LEN = 20;  // line length from circle edge
const AIM_GAP = 3;        // gap between circle and line start

function drawControllerAim(ctx, shipSX, shipSY, rotation) {
    // Aim position on the orbit
    const ax = shipSX + Math.cos(rotation) * AIM_ORBIT_R;
    const ay = shipSY + Math.sin(rotation) * AIM_ORBIT_R;

    // The aim rotates so one arm always points back at the ship
    // rotation + PI points from aim back toward ship
    const facing = rotation + Math.PI;

    ctx.save();
    ctx.translate(ax, ay);
    ctx.rotate(facing);

    ctx.strokeStyle = 'rgba(255, 255, 255, 0.35)';
    ctx.lineWidth = 1.5;

    // Small circle around center
    ctx.beginPath();
    ctx.arc(0, 0, AIM_CIRCLE_R, 0, Math.PI * 2);
    ctx.stroke();

    // 4 lines from circle edges outward (at 0, 90, 180, 270 degrees in local space)
    for (let i = 0; i < 4; i++) {
        const a = (i * Math.PI) / 2;
        const cos = Math.cos(a);
        const sin = Math.sin(a);
        const startR = AIM_CIRCLE_R + AIM_GAP;
        ctx.beginPath();
        ctx.moveTo(cos * startR, sin * startR);
        ctx.lineTo(cos * (startR + AIM_LINE_LEN), sin * (startR + AIM_LINE_LEN));
        ctx.stroke();
    }

    ctx.restore();
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
