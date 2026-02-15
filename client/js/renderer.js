import { state } from './state.js';
import { renderStarfield } from './starfield.js';
import { drawShip, initShips } from './ships.js';
import { renderProjectiles } from './projectiles.js';
import { updateParticles, renderParticles, renderExplosions, addEngineParticles, updateShake, updateDamageNumbers, renderDamageNumbers, updateHitMarkers, renderHitMarkers } from './effects.js';
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

    // Update screen shake
    updateShake(dt);

    // Virtual viewport in world units (what the camera sees)
    const vw = w / zoom;
    const vh = h / zoom;

    // Camera offset in world coords (with screen shake)
    const offsetX = state.camX - vw / 2 + state.shakeX;
    const offsetY = state.camY - vh / 2 + state.shakeY;

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
    updateDamageNumbers(dt);
    updateHitMarkers(dt);
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
    }

    // Auto-aim reticle (controller or mobile direct play, not desktop)
    if ((state.controllerAttached || state.isMobile) && state.myID) {
        const me = state.players.get(state.myID);
        if (me && me.a) {
            updateAndDrawControllerAim(ctx, me, offsetX, offsetY, dt);
        }
    }

    // Debug hitboxes
    if (state.debugHitboxes) {
        drawHitboxes(ctx, offsetX, offsetY, vw, vh);
    }

    // Damage numbers (world-space, inside zoom transform)
    renderDamageNumbers(ctx, offsetX, offsetY, vw, vh);

    ctx.restore(); // Remove zoom transform

    // Hit markers (screen-space, no zoom)
    renderHitMarkers(ctx);

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

// --- Auto-aim dashed-circle reticle ---
const AIM_ORBIT_R = 360;     // world units from ship center
const AIM_DETECT_R = 150;    // detection radius (world units)
const AIM_FREE_R = 150;      // visual radius when free
const AIM_LOCK_R = 20;       // visual radius when locked
const AIM_ANIM_SPEED = 4;    // progress units/sec (~0.25s transition)
const AIM_SPIN_MAX = 8;      // rad/s when fully locked

let aimState = {
    targetId: null,
    targetX: 0,
    targetY: 0,
    progress: 0,  // 0 = free, 1 = locked
    spinAngle: 0,
};

function updateAndDrawControllerAim(ctx, me, offsetX, offsetY, dt) {
    // Orbit position in world coords
    const orbitWX = me.x + Math.cos(me.r) * AIM_ORBIT_R;
    const orbitWY = me.y + Math.sin(me.r) * AIM_ORBIT_R;

    // Build list of enemies (other players + mobs)
    const enemies = [];
    for (const [id, p] of state.players) {
        if (id === state.myID || !p.a) continue;
        enemies.push({ id: 'p_' + id, x: p.x, y: p.y });
    }
    for (const [id, m] of state.mobs) {
        if (!m.a) continue;
        enemies.push({ id: 'm_' + id, x: m.x, y: m.y });
    }

    // Sticky lock: check if current target is still in range
    let locked = false;
    if (aimState.targetId !== null) {
        const t = enemies.find(e => e.id === aimState.targetId);
        if (t) {
            const dx = t.x - orbitWX;
            const dy = t.y - orbitWY;
            if (dx * dx + dy * dy <= AIM_DETECT_R * AIM_DETECT_R) {
                locked = true;
                aimState.targetX = t.x;
                aimState.targetY = t.y;
            }
        }
    }

    // If not locked, search for nearest enemy in range
    if (!locked) {
        let bestDist = AIM_DETECT_R * AIM_DETECT_R;
        let bestEnemy = null;
        for (const e of enemies) {
            const dx = e.x - orbitWX;
            const dy = e.y - orbitWY;
            const d2 = dx * dx + dy * dy;
            if (d2 <= bestDist) {
                bestDist = d2;
                bestEnemy = e;
            }
        }
        if (bestEnemy) {
            locked = true;
            aimState.targetId = bestEnemy.id;
            aimState.targetX = bestEnemy.x;
            aimState.targetY = bestEnemy.y;
        } else {
            aimState.targetId = null;
        }
    }

    // Animate progress
    const targetProgress = locked ? 1 : 0;
    if (aimState.progress < targetProgress) {
        aimState.progress = Math.min(1, aimState.progress + AIM_ANIM_SPEED * dt);
    } else if (aimState.progress > targetProgress) {
        aimState.progress = Math.max(0, aimState.progress - AIM_ANIM_SPEED * dt);
    }

    // Spin
    const spinSpeed = aimState.progress * AIM_SPIN_MAX;
    aimState.spinAngle += spinSpeed * dt;

    // Screen positions
    const orbitSX = orbitWX - offsetX;
    const orbitSY = orbitWY - offsetY;
    const targetSX = aimState.targetX - offsetX;
    const targetSY = aimState.targetY - offsetY;

    // Interpolate position and radius
    const p = aimState.progress;
    const cx = orbitSX + (targetSX - orbitSX) * p;
    const cy = orbitSY + (targetSY - orbitSY) * p;
    const radius = AIM_FREE_R + (AIM_LOCK_R - AIM_FREE_R) * p;

    // Draw dashed circle
    ctx.save();
    ctx.translate(cx, cy);
    ctx.rotate(aimState.spinAngle);

    const alpha = 0.3 + 0.3 * p; // brighter when locked
    ctx.strokeStyle = `rgba(255, 255, 255, ${alpha})`;
    ctx.lineWidth = 1.5;
    ctx.setLineDash([8, 6]);
    ctx.beginPath();
    ctx.arc(0, 0, radius, 0, Math.PI * 2);
    ctx.stroke();
    ctx.setLineDash([]);

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
