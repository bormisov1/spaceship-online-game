import { state } from './state.js';
import { SHIP_COLORS, PLAYER_MAX_HP, WORLD_W, WORLD_H } from './constants.js';

export function renderHUD(ctx) {
    const me = state.players.get(state.myID);

    // Health bar
    if (me && me.a) {
        const minDim = Math.min(state.screenW, state.screenH);
        const barW = Math.max(120, Math.min(200, minDim * 0.28));
        drawHealthBar(ctx, state.screenW / 2, state.screenH - 40, barW, 16, me.hp, me.mhp);
    }

    // Minimap (top right)
    drawMinimap(ctx);

    // Kill feed (below minimap)
    drawKillFeed(ctx);

    // Scoreboard (top left)
    drawScoreboard(ctx);

    // Death screen
    if (state.phase === 'dead' && state.deathInfo) {
        drawDeathScreen(ctx);
    }

    // Crosshair (desktop only)
    if (state.phase === 'playing' && !state.isMobile) {
        drawCrosshair(ctx);
    }

    // Mobile touch controls overlay
    if (state.isMobile && (state.phase === 'playing' || state.phase === 'dead')) {
        drawMobileControls(ctx);
    }

    // Connection status
    if (!state.connected) {
        ctx.fillStyle = '#ff4444';
        ctx.font = '16px monospace';
        ctx.textAlign = 'center';
        ctx.fillText('DISCONNECTED - Reconnecting...', state.screenW / 2, 30);
    }
}

function drawHealthBar(ctx, x, y, w, h, hp, maxHp) {
    const ratio = hp / maxHp;

    // Background
    ctx.fillStyle = 'rgba(0, 0, 0, 0.5)';
    ctx.fillRect(x - w / 2 - 2, y - 2, w + 4, h + 4);

    // Bar
    let color;
    if (ratio > 0.6) color = '#44ff44';
    else if (ratio > 0.3) color = '#ffaa00';
    else color = '#ff4444';

    ctx.fillStyle = color;
    ctx.fillRect(x - w / 2, y, w * ratio, h);

    // Border
    ctx.strokeStyle = '#ffffff44';
    ctx.lineWidth = 1;
    ctx.strokeRect(x - w / 2, y, w, h);

    // Text
    ctx.fillStyle = '#ffffff';
    ctx.font = 'bold 12px monospace';
    ctx.textAlign = 'center';
    ctx.fillText(`${hp}/${maxHp}`, x, y + h - 3);
}

function drawMinimap(ctx) {
    const minDim = Math.min(state.screenW, state.screenH);
    const size = Math.max(80, Math.min(180, minDim * 0.22));
    const margin = 10;
    const x = state.screenW - size - margin;
    const y = margin;

    // Background
    ctx.fillStyle = 'rgba(0, 40, 0, 0.5)';
    ctx.fillRect(x, y, size, size);

    // Border
    ctx.strokeStyle = '#00ff00';
    ctx.lineWidth = 1;
    ctx.strokeRect(x, y, size, size);

    // Draw all alive players as dots
    for (const p of state.players.values()) {
        if (!p.a) continue;
        const isMe = p.id === state.myID;
        const colors = SHIP_COLORS[p.s] || SHIP_COLORS[0];
        const dotX = x + (p.x / WORLD_W) * size;
        const dotY = y + (p.y / WORLD_H) * size;
        const radius = isMe ? 3 : 2;

        ctx.beginPath();
        ctx.arc(dotX, dotY, radius, 0, Math.PI * 2);
        ctx.fillStyle = isMe ? '#ffffff' : colors.main;
        ctx.fill();
    }
}

function drawKillFeed(ctx) {
    const now = performance.now();
    const x = state.screenW - 20;
    const minDim = Math.min(state.screenW, state.screenH);
    const mapSize = Math.max(80, Math.min(180, minDim * 0.22));
    let y = mapSize + 30; // below minimap

    ctx.textAlign = 'right';
    const fontSize = Math.max(10, Math.min(13, minDim * 0.018)) | 0;
    ctx.font = `${fontSize}px monospace`;

    for (let i = state.killFeed.length - 1; i >= 0; i--) {
        const kill = state.killFeed[i];
        const age = (now - kill.time) / 1000;
        if (age > 8) {
            state.killFeed.splice(i, 1);
            continue;
        }

        const alpha = age > 6 ? (8 - age) / 2 : 1;
        ctx.globalAlpha = alpha;

        ctx.fillStyle = '#ffaa00';
        ctx.fillText(`${kill.killer}`, x - ctx.measureText(` killed ${kill.victim}`).width, y);
        ctx.fillStyle = '#ffffff';
        ctx.fillText(` killed `, x - ctx.measureText(`${kill.victim}`).width, y);
        ctx.fillStyle = '#ff4444';
        ctx.fillText(kill.victim, x, y);

        y += 20;
    }
    ctx.globalAlpha = 1;
}

function drawScoreboard(ctx) {
    const minDim = Math.min(state.screenW, state.screenH);
    const scale = Math.max(0.7, Math.min(1, minDim / 800));
    const fontSize = (13 * scale) | 0;
    const headerSize = (12 * scale) | 0;
    const lineH = (18 * scale) | 0;
    const panelW = (180 * scale) | 0;
    const scoreX = (150 * scale) | 0;
    const maxPlayers = minDim < 500 ? 5 : 8;

    // Collect and sort players by score
    const players = Array.from(state.players.values())
        .sort((a, b) => b.sc - a.sc || a.id.localeCompare(b.id))
        .slice(0, maxPlayers);

    ctx.textAlign = 'left';
    ctx.font = `${fontSize}px monospace`;

    const x = 15;
    let y = 60 * scale;

    ctx.fillStyle = 'rgba(0, 0, 0, 0.4)';
    ctx.fillRect(x - 5, y - lineH, panelW, players.length * (lineH + 2) + lineH + 6);

    ctx.fillStyle = '#ffffff88';
    ctx.font = `bold ${headerSize}px monospace`;
    ctx.fillText('SCOREBOARD', x, y - 2);
    y += lineH;

    ctx.font = `${fontSize}px monospace`;
    for (const p of players) {
        const isMe = p.id === state.myID;
        const colors = SHIP_COLORS[p.s] || SHIP_COLORS[0];

        ctx.fillStyle = isMe ? '#ffffff' : '#aaaaaa';
        const maxNameLen = minDim < 500 ? 8 : 12;
        const nameStr = p.n.length > maxNameLen ? p.n.slice(0, maxNameLen) + '..' : p.n;
        ctx.fillText(`${nameStr}`, x, y);

        ctx.fillStyle = colors.main;
        ctx.fillText(`${p.sc}`, x + scoreX, y);
        y += lineH;
    }
}

function drawDeathScreen(ctx) {
    ctx.fillStyle = 'rgba(0, 0, 0, 0.5)';
    ctx.fillRect(0, 0, state.screenW, state.screenH);

    ctx.textAlign = 'center';
    ctx.fillStyle = '#ff4444';
    ctx.font = 'bold 36px monospace';
    ctx.fillText('DESTROYED', state.screenW / 2, state.screenH / 2 - 30);

    if (state.deathInfo) {
        ctx.fillStyle = '#ffffff';
        ctx.font = '20px monospace';
        ctx.fillText(`by ${state.deathInfo.killerName}`, state.screenW / 2, state.screenH / 2 + 10);
    }

    ctx.fillStyle = '#aaaaaa';
    ctx.font = '16px monospace';
    ctx.fillText('Respawning...', state.screenW / 2, state.screenH / 2 + 50);
}

function drawCrosshair(ctx) {
    const mx = state.mouseX;
    const my = state.mouseY;
    const size = 12;

    ctx.strokeStyle = 'rgba(255, 255, 255, 0.6)';
    ctx.lineWidth = 1.5;

    ctx.beginPath();
    ctx.moveTo(mx - size, my);
    ctx.lineTo(mx - size / 3, my);
    ctx.moveTo(mx + size / 3, my);
    ctx.lineTo(mx + size, my);
    ctx.moveTo(mx, my - size);
    ctx.lineTo(mx, my - size / 3);
    ctx.moveTo(mx, my + size / 3);
    ctx.lineTo(mx, my + size);
    ctx.stroke();

    ctx.beginPath();
    ctx.arc(mx, my, 2, 0, Math.PI * 2);
    ctx.fillStyle = 'rgba(255, 255, 255, 0.6)';
    ctx.fill();
}

function drawMobileControls(ctx) {
    // Draw joystick when active
    if (state.touchJoystick) {
        const { startX, startY, currentX, currentY } = state.touchJoystick;
        const maxRadius = 60;

        // Base circle
        ctx.beginPath();
        ctx.arc(startX, startY, maxRadius, 0, Math.PI * 2);
        ctx.strokeStyle = 'rgba(255, 255, 255, 0.2)';
        ctx.lineWidth = 2;
        ctx.stroke();

        // Inner dead zone circle
        ctx.beginPath();
        ctx.arc(startX, startY, 12, 0, Math.PI * 2);
        ctx.strokeStyle = 'rgba(255, 255, 255, 0.1)';
        ctx.lineWidth = 1;
        ctx.stroke();

        // Thumb position (clamped to base radius for visual)
        let dx = currentX - startX;
        let dy = currentY - startY;
        const dist = Math.sqrt(dx * dx + dy * dy);
        if (dist > maxRadius) {
            dx = (dx / dist) * maxRadius;
            dy = (dy / dist) * maxRadius;
        }

        ctx.beginPath();
        ctx.arc(startX + dx, startY + dy, 18, 0, Math.PI * 2);
        ctx.fillStyle = 'rgba(255, 255, 255, 0.25)';
        ctx.fill();
        ctx.strokeStyle = 'rgba(255, 255, 255, 0.4)';
        ctx.lineWidth = 1.5;
        ctx.stroke();
    }
}

// Draw health bar above other players
export function drawPlayerHealthBar(ctx, x, y, hp, maxHp, name, isMe) {
    const barW = 40;
    const barH = 4;
    const barY = y - 30;

    if (!isMe) {
        // Name tag
        ctx.fillStyle = '#ffffff99';
        ctx.font = '11px monospace';
        ctx.textAlign = 'center';
        ctx.fillText(name, x, barY - 8);
    }

    // Health bar
    const ratio = hp / maxHp;
    ctx.fillStyle = 'rgba(0,0,0,0.5)';
    ctx.fillRect(x - barW / 2, barY, barW, barH);

    let color;
    if (ratio > 0.6) color = '#44ff44';
    else if (ratio > 0.3) color = '#ffaa00';
    else color = '#ff4444';

    ctx.fillStyle = color;
    ctx.fillRect(x - barW / 2, barY, barW * ratio, barH);
}
