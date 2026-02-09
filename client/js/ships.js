import { SHIP_COLORS, SHIP_SIZE } from './constants.js';

// Cache generated ship sprites
const shipCanvases = [];

export function initShips() {
    for (let i = 0; i < 4; i++) {
        shipCanvases[i] = generateShipSprite(i);
    }
}

export function getShipSprite(type) {
    if (!shipCanvases[type]) {
        shipCanvases[type] = generateShipSprite(type);
    }
    return shipCanvases[type];
}

function generateShipSprite(type) {
    const size = SHIP_SIZE * 2; // higher res for quality
    const c = document.createElement('canvas');
    c.width = size;
    c.height = size;
    const ctx = c.getContext('2d');
    const cx = size / 2;
    const cy = size / 2;
    const s = size / 2;
    const colors = SHIP_COLORS[type];

    ctx.save();
    ctx.translate(cx, cy);

    switch (type) {
        case 0: drawFighter(ctx, s, colors); break;
        case 1: drawInterceptor(ctx, s, colors); break;
        case 2: drawBomber(ctx, s, colors); break;
        case 3: drawScout(ctx, s, colors); break;
    }

    ctx.restore();
    return c;
}

function drawFighter(ctx, s, colors) {
    // X-Wing style
    ctx.fillStyle = colors.main;
    ctx.strokeStyle = colors.accent;
    ctx.lineWidth = 2;

    // Main body
    ctx.beginPath();
    ctx.moveTo(s * 0.9, 0);
    ctx.lineTo(-s * 0.5, -s * 0.2);
    ctx.lineTo(-s * 0.7, -s * 0.15);
    ctx.lineTo(-s * 0.7, s * 0.15);
    ctx.lineTo(-s * 0.5, s * 0.2);
    ctx.closePath();
    ctx.fill();
    ctx.stroke();

    // Wings
    ctx.fillStyle = colors.accent;
    ctx.beginPath();
    ctx.moveTo(s * 0.2, -s * 0.15);
    ctx.lineTo(-s * 0.5, -s * 0.7);
    ctx.lineTo(-s * 0.7, -s * 0.65);
    ctx.lineTo(-s * 0.4, -s * 0.15);
    ctx.closePath();
    ctx.fill();
    ctx.stroke();

    ctx.beginPath();
    ctx.moveTo(s * 0.2, s * 0.15);
    ctx.lineTo(-s * 0.5, s * 0.7);
    ctx.lineTo(-s * 0.7, s * 0.65);
    ctx.lineTo(-s * 0.4, s * 0.15);
    ctx.closePath();
    ctx.fill();
    ctx.stroke();

    // Cockpit
    ctx.fillStyle = '#aaddff';
    ctx.beginPath();
    ctx.ellipse(s * 0.3, 0, s * 0.15, s * 0.08, 0, 0, Math.PI * 2);
    ctx.fill();
}

function drawInterceptor(ctx, s, colors) {
    // TIE-fighter style
    ctx.fillStyle = colors.main;
    ctx.strokeStyle = colors.accent;
    ctx.lineWidth = 2;

    // Central pod
    ctx.beginPath();
    ctx.arc(0, 0, s * 0.25, 0, Math.PI * 2);
    ctx.fill();
    ctx.stroke();

    // Cockpit window
    ctx.fillStyle = '#aaddff';
    ctx.beginPath();
    ctx.arc(s * 0.05, 0, s * 0.12, 0, Math.PI * 2);
    ctx.fill();

    // Side panels
    ctx.fillStyle = colors.accent;
    ctx.strokeStyle = colors.main;

    // Top panel
    ctx.beginPath();
    ctx.moveTo(s * 0.3, -s * 0.25);
    ctx.lineTo(-s * 0.3, -s * 0.25);
    ctx.lineTo(-s * 0.4, -s * 0.8);
    ctx.lineTo(s * 0.4, -s * 0.8);
    ctx.closePath();
    ctx.fill();
    ctx.stroke();

    // Bottom panel
    ctx.beginPath();
    ctx.moveTo(s * 0.3, s * 0.25);
    ctx.lineTo(-s * 0.3, s * 0.25);
    ctx.lineTo(-s * 0.4, s * 0.8);
    ctx.lineTo(s * 0.4, s * 0.8);
    ctx.closePath();
    ctx.fill();
    ctx.stroke();

    // Struts
    ctx.strokeStyle = colors.main;
    ctx.lineWidth = 3;
    ctx.beginPath();
    ctx.moveTo(0, -s * 0.25);
    ctx.lineTo(0, -s * 0.8);
    ctx.moveTo(0, s * 0.25);
    ctx.lineTo(0, s * 0.8);
    ctx.stroke();
}

function drawBomber(ctx, s, colors) {
    // Y-Wing / heavy bomber style
    ctx.fillStyle = colors.main;
    ctx.strokeStyle = colors.accent;
    ctx.lineWidth = 2;

    // Main body (wider)
    ctx.beginPath();
    ctx.moveTo(s * 0.7, 0);
    ctx.lineTo(s * 0.2, -s * 0.3);
    ctx.lineTo(-s * 0.6, -s * 0.25);
    ctx.lineTo(-s * 0.7, 0);
    ctx.lineTo(-s * 0.6, s * 0.25);
    ctx.lineTo(s * 0.2, s * 0.3);
    ctx.closePath();
    ctx.fill();
    ctx.stroke();

    // Engine nacelles
    ctx.fillStyle = colors.accent;
    ctx.beginPath();
    ctx.roundRect(-s * 0.5, -s * 0.55, s * 0.8, s * 0.15, 3);
    ctx.fill();
    ctx.stroke();
    ctx.beginPath();
    ctx.roundRect(-s * 0.5, s * 0.4, s * 0.8, s * 0.15, 3);
    ctx.fill();
    ctx.stroke();

    // Cockpit
    ctx.fillStyle = '#aaddff';
    ctx.beginPath();
    ctx.ellipse(s * 0.35, 0, s * 0.12, s * 0.1, 0, 0, Math.PI * 2);
    ctx.fill();
}

function drawScout(ctx, s, colors) {
    // A-Wing / sleek scout style
    ctx.fillStyle = colors.main;
    ctx.strokeStyle = colors.accent;
    ctx.lineWidth = 2;

    // Main body (arrow shape)
    ctx.beginPath();
    ctx.moveTo(s * 0.9, 0);
    ctx.lineTo(0, -s * 0.15);
    ctx.lineTo(-s * 0.4, -s * 0.5);
    ctx.lineTo(-s * 0.6, -s * 0.45);
    ctx.lineTo(-s * 0.3, -s * 0.1);
    ctx.lineTo(-s * 0.5, 0);
    ctx.lineTo(-s * 0.3, s * 0.1);
    ctx.lineTo(-s * 0.6, s * 0.45);
    ctx.lineTo(-s * 0.4, s * 0.5);
    ctx.lineTo(0, s * 0.15);
    ctx.closePath();
    ctx.fill();
    ctx.stroke();

    // Central stripe
    ctx.fillStyle = colors.accent;
    ctx.beginPath();
    ctx.moveTo(s * 0.8, 0);
    ctx.lineTo(0, -s * 0.05);
    ctx.lineTo(-s * 0.4, -s * 0.05);
    ctx.lineTo(-s * 0.4, s * 0.05);
    ctx.lineTo(0, s * 0.05);
    ctx.closePath();
    ctx.fill();

    // Cockpit
    ctx.fillStyle = '#aaddff';
    ctx.beginPath();
    ctx.ellipse(s * 0.3, 0, s * 0.1, s * 0.06, 0, 0, Math.PI * 2);
    ctx.fill();
}

export function drawShip(ctx, x, y, rotation, shipType, alpha = 1) {
    const sprite = getShipSprite(shipType);
    ctx.save();
    ctx.globalAlpha = alpha;
    ctx.translate(x, y);
    ctx.rotate(rotation);
    ctx.drawImage(sprite, -SHIP_SIZE / 2, -SHIP_SIZE / 2, SHIP_SIZE, SHIP_SIZE);
    ctx.restore();
}
