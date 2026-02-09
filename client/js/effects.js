import { state } from './state.js';
import { SHIP_COLORS } from './constants.js';

// Engine particles
export function addEngineParticles(x, y, rotation, speed, shipType) {
    if (speed < 20) return; // Only show when moving

    const colors = SHIP_COLORS[shipType] || SHIP_COLORS[0];
    const count = Math.min(Math.floor(speed / 50), 5);

    for (let i = 0; i < count; i++) {
        const angle = rotation + Math.PI + (Math.random() - 0.5) * 0.6;
        const spd = speed * 0.3 + Math.random() * 80;
        state.particles.push({
            x: x - Math.cos(rotation) * 15 + (Math.random() - 0.5) * 6,
            y: y - Math.sin(rotation) * 15 + (Math.random() - 0.5) * 6,
            vx: Math.cos(angle) * spd,
            vy: Math.sin(angle) * spd,
            life: 0.3 + Math.random() * 0.3,
            maxLife: 0.3 + Math.random() * 0.3,
            size: 2 + Math.random() * 3,
            color: colors.engine,
            type: 'engine',
        });
    }
}

// Explosion effect
export function addExplosion(x, y) {
    const count = 30;
    for (let i = 0; i < count; i++) {
        const angle = (Math.PI * 2 * i) / count + (Math.random() - 0.5) * 0.5;
        const spd = 100 + Math.random() * 300;
        const colors = ['#ff4400', '#ff8800', '#ffcc00', '#ffffff', '#ff2200'];
        state.particles.push({
            x: x + (Math.random() - 0.5) * 10,
            y: y + (Math.random() - 0.5) * 10,
            vx: Math.cos(angle) * spd,
            vy: Math.sin(angle) * spd,
            life: 0.5 + Math.random() * 0.8,
            maxLife: 0.5 + Math.random() * 0.8,
            size: 3 + Math.random() * 5,
            color: colors[Math.floor(Math.random() * colors.length)],
            type: 'explosion',
        });
    }

    // Add shockwave
    state.explosions.push({
        x,
        y,
        radius: 0,
        maxRadius: 80,
        life: 0.4,
        maxLife: 0.4,
    });
}

export function updateParticles(dt) {
    // Update particles
    for (let i = state.particles.length - 1; i >= 0; i--) {
        const p = state.particles[i];
        p.x += p.vx * dt;
        p.y += p.vy * dt;
        p.life -= dt;
        p.vx *= 0.98;
        p.vy *= 0.98;

        if (p.life <= 0) {
            state.particles.splice(i, 1);
        }
    }

    // Update explosions (shockwaves)
    for (let i = state.explosions.length - 1; i >= 0; i--) {
        const e = state.explosions[i];
        e.life -= dt;
        e.radius = e.maxRadius * (1 - e.life / e.maxLife);

        if (e.life <= 0) {
            state.explosions.splice(i, 1);
        }
    }
}

export function renderParticles(ctx, offsetX, offsetY) {
    for (const p of state.particles) {
        const sx = p.x - offsetX;
        const sy = p.y - offsetY;
        if (sx < -20 || sx > state.screenW + 20 || sy < -20 || sy > state.screenH + 20) continue;

        const alpha = Math.max(0, p.life / p.maxLife);
        const size = p.size * (p.type === 'explosion' ? (1 + (1 - alpha) * 0.5) : alpha);

        ctx.globalAlpha = alpha;
        ctx.fillStyle = p.color;
        ctx.beginPath();
        ctx.arc(sx, sy, size, 0, Math.PI * 2);
        ctx.fill();
    }
    ctx.globalAlpha = 1;
}

export function renderExplosions(ctx, offsetX, offsetY) {
    for (const e of state.explosions) {
        const sx = e.x - offsetX;
        const sy = e.y - offsetY;
        if (sx < -100 || sx > state.screenW + 100 || sy < -100 || sy > state.screenH + 100) continue;

        const alpha = (e.life / e.maxLife) * 0.4;
        ctx.strokeStyle = `rgba(255, 200, 50, ${alpha})`;
        ctx.lineWidth = 3;
        ctx.beginPath();
        ctx.arc(sx, sy, e.radius, 0, Math.PI * 2);
        ctx.stroke();
    }
}
