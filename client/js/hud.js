import { state } from './state.js';
import { SHIP_COLORS, PLAYER_MAX_HP } from './constants.js';

export function renderHUD(ctx) {
    const me = state.players.get(state.myID);

    // Health bar
    if (me && me.a) {
        drawHealthBar(ctx, state.screenW / 2, state.screenH - 40, 200, 16, me.hp, me.mhp);
    }

    // Kill feed (top right)
    drawKillFeed(ctx);

    // Scoreboard (top left)
    drawScoreboard(ctx);

    // Death screen
    if (state.phase === 'dead' && state.deathInfo) {
        drawDeathScreen(ctx);
    }

    // Crosshair
    if (state.phase === 'playing') {
        drawCrosshair(ctx);
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

function drawKillFeed(ctx) {
    const now = performance.now();
    const x = state.screenW - 20;
    let y = 60;

    ctx.textAlign = 'right';
    ctx.font = '13px monospace';

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
    // Collect and sort players by score
    const players = Array.from(state.players.values())
        .sort((a, b) => b.sc - a.sc)
        .slice(0, 8);

    ctx.textAlign = 'left';
    ctx.font = '13px monospace';

    const x = 15;
    let y = 60;

    ctx.fillStyle = 'rgba(0, 0, 0, 0.4)';
    ctx.fillRect(x - 5, y - 18, 180, players.length * 20 + 24);

    ctx.fillStyle = '#ffffff88';
    ctx.font = 'bold 12px monospace';
    ctx.fillText('SCOREBOARD', x, y - 2);
    y += 18;

    ctx.font = '13px monospace';
    for (const p of players) {
        const isMe = p.id === state.myID;
        const colors = SHIP_COLORS[p.s] || SHIP_COLORS[0];

        ctx.fillStyle = isMe ? '#ffffff' : '#aaaaaa';
        const nameStr = p.n.length > 12 ? p.n.slice(0, 12) + '..' : p.n;
        ctx.fillText(`${nameStr}`, x, y);

        ctx.fillStyle = colors.main;
        ctx.fillText(`${p.sc}`, x + 150, y);
        y += 18;
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
