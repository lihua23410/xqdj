const test = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const vm = require('node:vm');
const path = require('node:path');

function setup() {
  const guides = new Map(), hooks = new Map(), calls = [];
  let voices = 0, plays = 0;
  const pen = new Proxy({}, { get: (_, key) => (...args) => calls.push([key, ...args]), set: () => true });
  const api = { SOUND_MODE: { NORMAL: 0 }, registerShot: (k, fn) => hooks.set(k, fn), audio: () => {
    voices++; return { stop() {}, reset() {}, play() { plays++; } };
  } };
  const sandbox = { window: { arena: api, devicePixelRatio: 1 }, document: {
    currentScript: { dataset: { pack: '/ball/闪烁' } },
    createElement: () => ({ style: {}, getContext: () => pen }),
    getElementById: (id) => guides.get(id),
  } };
  vm.runInNewContext(fs.readFileSync(path.join(__dirname, 'fx/reflex.js'), 'utf8'), sandbox);
  const ctx = { time: 1, scale: 1, cx: 280, cy: 280, seenGuides: new Set(), effects: [], ensureGuide(id) {
    if (!guides.has(id)) guides.set(id, { appendChild() {} });
    return guides.get(id);
  } };
  const u = { id: 1, x: 100, y: 80, radius: 18 };
  function draw(st, time = 1, unit = u) {
    calls.length = 0; ctx.time = time;
    ctx.effects = st ? [{ name: 'flicker-state', unitId: unit.id, data: st }] : [];
    sandbox.window.lookFX.flicker.guide(unit, ctx);
    return JSON.parse(JSON.stringify(calls));
  }
  return { draw, guides, ctx, u, hooks, voices: () => voices, plays: () => plays, skin: sandbox.window.lookFX.flicker };
}

test('pause freezes actual trail positions and expiry removes them', () => {
 const r=setup(); const st={Trail:[{X:0,Y:20,T:1}]}; const first=r.draw(st);
 assert.deepEqual(r.draw(null),first);
 assert.ok(!r.draw(null,1.3).some(c=>c[0]==='fillRect'));
});

test('bounded canvas per owner; focus expires without a new state during stun', () => {
  const r = setup();
  for (let i = 0; i < 100; i++) r.draw({ Phase: 2, FocusUntil: 1.1 });
  assert.equal(r.guides.size, 1);
  r.draw({}, 1, { ...r.u, id: 2 }); assert.equal(r.guides.size, 2);
  const classes = new Map(); const el = { classList: { toggle(k, v) { classes.set(k, v); } } };
  r.skin.tick(el, r.u, { time: 1 }); assert.equal(classes.get('flicker-focus'), true);
  r.skin.tick(el, r.u, { time: 1.2 }); assert.equal(classes.get('flicker-focus'), false);
});

test('persistent state never plays audio; live visual pass does not double shots; pools are bounded', () => {
  const r = setup();
  for (let i = 0; i < 120; i++) r.hooks.get('flicker-state')({});
  assert.equal(r.voices(), 0);
  r.hooks.get('shot')({}, { audioOnly: true });
  r.hooks.get('shot')({}, { visualOnly: true });
  assert.equal(r.plays(), 1);
  for (let i = 0; i < 20; i++) r.hooks.get('shot')({}, {});
  assert.equal(r.voices(), 3); assert.equal(r.plays(), 21);
});
