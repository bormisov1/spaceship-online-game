import { state } from './state.js';
import { SHIP_COLORS } from './constants.js';

const MAX_PARTICLES = 200;
const MAX_DAMAGE_NUMBERS = 30;
const HIT_MARKER_DURATION = 0.25; // seconds

// Engine particles
export function addEngineParticles(x, y, rotation, speed, shipType) {
    if (speed < 20) return; // Only show when moving
    if (state.particles.length >= MAX_PARTICLES) return;

    const colors = SHIP_COLORS[shipType] || SHIP_COLORS[0];
    const count = Math.min(Math.floor(speed / 50), 5);

    for (let i = 0; i < count; i++) {
        if (state.particles.length >= MAX_PARTICLES) break;
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
    // Update particles with swap-and-pop removal
    const particles = state.particles;
    let i = 0;
    while (i < particles.length) {
        const p = particles[i];
        p.x += p.vx * dt;
        p.y += p.vy * dt;
        p.life -= dt;
        p.vx *= 0.98;
        p.vy *= 0.98;

        if (p.life <= 0) {
            // Swap with last element and pop
            particles[i] = particles[particles.length - 1];
            particles.pop();
            // Don't increment i — re-check the swapped element
        } else {
            i++;
        }
    }

    // Update explosions (shockwaves) with swap-and-pop
    const explosions = state.explosions;
    let j = 0;
    while (j < explosions.length) {
        const e = explosions[j];
        e.life -= dt;
        e.radius = e.maxRadius * (1 - e.life / e.maxLife);

        if (e.life <= 0) {
            explosions[j] = explosions[explosions.length - 1];
            explosions.pop();
        } else {
            j++;
        }
    }
}

export function renderParticles(ctx, offsetX, offsetY, vw, vh) {
    for (const p of state.particles) {
        const sx = p.x - offsetX;
        const sy = p.y - offsetY;
        if (sx < -20 || sx > vw + 20 || sy < -20 || sy > vh + 20) continue;

        const alpha = Math.max(0, p.life / p.maxLife);
        const size = p.size * (p.type === 'explosion' ? (1 + (1 - alpha) * 0.5) : alpha);

        ctx.globalAlpha = alpha;
        ctx.fillStyle = p.color;

        if (size < 3) {
            // Use rect for small particles — much cheaper than arc
            ctx.fillRect(sx - size, sy - size, size * 2, size * 2);
        } else {
            ctx.beginPath();
            ctx.arc(sx, sy, size, 0, Math.PI * 2);
            ctx.fill();
        }
    }
    ctx.globalAlpha = 1;
}

export function renderExplosions(ctx, offsetX, offsetY, vw, vh) {
    for (const e of state.explosions) {
        const sx = e.x - offsetX;
        const sy = e.y - offsetY;
        if (sx < -100 || sx > vw + 100 || sy < -100 || sy > vh + 100) continue;

        const alpha = (e.life / e.maxLife) * 0.4;
        ctx.strokeStyle = `rgba(255, 200, 50, ${alpha})`;
        ctx.lineWidth = 3;
        ctx.beginPath();
        ctx.arc(sx, sy, e.radius, 0, Math.PI * 2);
        ctx.stroke();
    }
}

// --- Screen Shake ---

export function triggerShake(intensity) {
    // Stack shakes but cap intensity
    state.shakeIntensity = Math.min(state.shakeIntensity + intensity, 20);
    state.shakeDecay = state.shakeIntensity;
}

export function updateShake(dt) {
    if (state.shakeIntensity <= 0) {
        state.shakeX = 0;
        state.shakeY = 0;
        return;
    }
    // Random offset proportional to intensity
    const angle = Math.random() * Math.PI * 2;
    state.shakeX = Math.cos(angle) * state.shakeIntensity;
    state.shakeY = Math.sin(angle) * state.shakeIntensity;
    // Decay quickly
    state.shakeIntensity -= state.shakeDecay * dt * 6;
    if (state.shakeIntensity < 0.5) {
        state.shakeIntensity = 0;
        state.shakeX = 0;
        state.shakeY = 0;
    }
}

// --- Damage Numbers ---

export function addDamageNumber(x, y, dmg, isHeal) {
    if (state.damageNumbers.length >= MAX_DAMAGE_NUMBERS) {
        state.damageNumbers.shift();
    }
    state.damageNumbers.push({
        x,
        y,
        text: isHeal ? `+${dmg}` : `-${dmg}`,
        color: isHeal ? '#44ff44' : '#ff4444',
        life: 1.0,
        maxLife: 1.0,
        vy: -60, // float upward in world units/s
        offsetX: (Math.random() - 0.5) * 20,
    });
}

export function updateDamageNumbers(dt) {
    let i = 0;
    while (i < state.damageNumbers.length) {
        const dn = state.damageNumbers[i];
        dn.life -= dt;
        dn.y += dn.vy * dt;
        if (dn.life <= 0) {
            state.damageNumbers.splice(i, 1);
        } else {
            i++;
        }
    }
}

export function renderDamageNumbers(ctx, offsetX, offsetY, vw, vh) {
    ctx.textAlign = 'center';
    for (const dn of state.damageNumbers) {
        const sx = dn.x + dn.offsetX - offsetX;
        const sy = dn.y - offsetY;
        if (sx < -50 || sx > vw + 50 || sy < -50 || sy > vh + 50) continue;

        const alpha = Math.max(0, dn.life / dn.maxLife);
        const scale = 1 + (1 - alpha) * 0.3; // grow slightly as fading

        ctx.globalAlpha = alpha;
        ctx.font = `bold ${Math.round(14 * scale)}px monospace`;
        // Shadow for readability
        ctx.fillStyle = '#000000';
        ctx.fillText(dn.text, sx + 1, sy + 1);
        ctx.fillStyle = dn.color;
        ctx.fillText(dn.text, sx, sy);
    }
    ctx.globalAlpha = 1;
}

// --- Hit Markers (screen-space) ---

export function addHitMarker() {
    state.hitMarkers.push({
        life: HIT_MARKER_DURATION,
        maxLife: HIT_MARKER_DURATION,
    });
}

export function updateHitMarkers(dt) {
    let i = 0;
    while (i < state.hitMarkers.length) {
        state.hitMarkers[i].life -= dt;
        if (state.hitMarkers[i].life <= 0) {
            state.hitMarkers.splice(i, 1);
        } else {
            i++;
        }
    }
}

export function renderHitMarkers(ctx) {
    if (state.hitMarkers.length === 0) return;

    // Draw an X at screen center (crosshair position)
    const cx = state.screenW / 2;
    const cy = state.screenH / 2;

    for (const hm of state.hitMarkers) {
        const alpha = Math.max(0, hm.life / hm.maxLife);
        const size = 10 + (1 - alpha) * 4; // expand slightly
        const gap = 3;

        ctx.globalAlpha = alpha;
        ctx.strokeStyle = '#ffffff';
        ctx.lineWidth = 2.5;
        ctx.beginPath();
        // Top-left to center
        ctx.moveTo(cx - size, cy - size);
        ctx.lineTo(cx - gap, cy - gap);
        // Top-right to center
        ctx.moveTo(cx + size, cy - size);
        ctx.lineTo(cx + gap, cy - gap);
        // Bottom-left to center
        ctx.moveTo(cx - size, cy + size);
        ctx.lineTo(cx - gap, cy + gap);
        // Bottom-right to center
        ctx.moveTo(cx + size, cy + size);
        ctx.lineTo(cx + gap, cy + gap);
        ctx.stroke();
    }
    ctx.globalAlpha = 1;
}
