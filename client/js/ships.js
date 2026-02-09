import { SHIP_SIZE } from './constants.js';

const SHIP_FILES = [
    'assets/ships/Fighter.png',
    'assets/ships/Artillery.png',
    'assets/ships/Cruiser.png',
    'assets/ships/Destroyer.png',
];

// Cache ship sprite canvases
const shipCanvases = [];
let loaded = false;

export function initShips() {
    SHIP_FILES.forEach((src, i) => {
        const img = new Image();
        img.onload = () => {
            const size = SHIP_SIZE * 2;
            const c = document.createElement('canvas');
            c.width = size;
            c.height = size;
            const ctx = c.getContext('2d');
            ctx.drawImage(img, 0, 0, size, size);
            shipCanvases[i] = c;
            if (shipCanvases.filter(Boolean).length === 4) loaded = true;
        };
        img.src = src;
    });
}

export function getShipSprite(type) {
    return shipCanvases[type] || null;
}

export function drawShip(ctx, x, y, rotation, shipType, alpha = 1) {
    const sprite = getShipSprite(shipType);
    if (!sprite) return;
    ctx.save();
    ctx.globalAlpha = alpha;
    ctx.translate(x, y);
    ctx.rotate(rotation);
    ctx.drawImage(sprite, -SHIP_SIZE / 2, -SHIP_SIZE / 2, SHIP_SIZE, SHIP_SIZE);
    ctx.restore();
}
