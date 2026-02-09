// Browser-based test runner
// Run by calling runTests() in browser console

import { WORLD_W, WORLD_H, PLAYER_RADIUS, PROJECTILE_RADIUS, PLAYER_MAX_HP } from './constants.js';

let passed = 0;
let failed = 0;

function assert(condition, msg) {
    if (condition) {
        passed++;
        console.log(`  PASS: ${msg}`);
    } else {
        failed++;
        console.error(`  FAIL: ${msg}`);
    }
}

function testConstants() {
    console.log('--- Constants ---');
    assert(WORLD_W === 4000, 'World width is 4000');
    assert(WORLD_H === 4000, 'World height is 4000');
    assert(PLAYER_RADIUS === 20, 'Player radius is 20');
    assert(PROJECTILE_RADIUS === 4, 'Projectile radius is 4');
    assert(PLAYER_MAX_HP === 100, 'Player max HP is 100');
}

function testStateStructure() {
    console.log('--- State structure ---');
    import('./state.js').then(({ state }) => {
        assert(state.players instanceof Map, 'players is a Map');
        assert(state.projectiles instanceof Map, 'projectiles is a Map');
        assert(Array.isArray(state.killFeed), 'killFeed is an array');
        assert(Array.isArray(state.particles), 'particles is an array');
        assert(state.phase === 'lobby', 'initial phase is lobby');
    });
}

function testShipGeneration() {
    console.log('--- Ship generation ---');
    import('./ships.js').then(({ getShipSprite }) => {
        for (let i = 0; i < 4; i++) {
            const sprite = getShipSprite(i);
            assert(sprite instanceof HTMLCanvasElement, `Ship ${i} generates canvas`);
            assert(sprite.width > 0, `Ship ${i} has width`);
            assert(sprite.height > 0, `Ship ${i} has height`);
        }
    });
}

export function runTests() {
    passed = 0;
    failed = 0;
    console.log('=== Running client tests ===');

    testConstants();
    testStateStructure();
    testShipGeneration();

    setTimeout(() => {
        console.log(`\n=== Results: ${passed} passed, ${failed} failed ===`);
    }, 500);
}

// Expose globally
window.runTests = runTests;
