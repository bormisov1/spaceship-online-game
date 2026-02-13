// Global game state
export const state = {
    // Connection
    connected: false,
    myID: null,
    myShip: 0,
    sessionID: null,
    urlSessionID: null, // UUID extracted from URL path

    // Game state from server
    players: new Map(),     // id -> PlayerState
    projectiles: new Map(), // id -> ProjectileState
    tick: 0,

    // New entities from server
    mobs: new Map(),
    asteroids: new Map(),
    pickups: new Map(),

    // Previous state for interpolation
    prevPlayers: new Map(),
    prevProjectiles: new Map(),
    prevMobs: new Map(),
    prevAsteroids: new Map(),
    prevPickups: new Map(),
    lastStateTime: 0,
    stateInterval: 1000 / 30, // server broadcasts at 30Hz

    // Local
    canvas: null,
    bgCanvas: null,
    ctx: null,
    bgCtx: null,
    screenW: 0,
    screenH: 0,

    // Camera (centered on local player)
    camX: 0,
    camY: 0,
    camZoom: 1, // <1 on small screens to show more world

    // Input
    mouseX: 0,
    mouseY: 0,
    mouseWorldX: 0,
    mouseWorldY: 0,
    firing: false,
    boosting: false,

    // UI state
    phase: 'lobby', // 'lobby' | 'playing' | 'dead'
    sessions: [],
    killFeed: [],   // { killer, victim, time }
    deathInfo: null, // { killerName }

    // Mobile
    isMobile: false,
    touchJoystick: null, // { startX, startY, currentX, currentY }

    // Debug
    debugHitboxes: false,

    // Effects
    particles: [],
    explosions: [],
};
