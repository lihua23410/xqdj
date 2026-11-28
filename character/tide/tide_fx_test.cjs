const fs = require('fs');
const path = require('path');
const vm = require('vm');
const assert = require('assert');

function createMockElement(tagName) {
  const classes = new Set();
  const children = [];
  const el = {
    tagName: (tagName || 'div').toUpperCase(),
    className: '',
    innerHTML: '',
    parentNode: null,
    parentElement: null,
    children,
    style: {
      vars: {},
      setProperty(k, v) { this.vars[k] = v; },
      getPropertyValue(k) { return this.vars[k] || ''; },
    },
    classList: {
      add(...xs) { xs.forEach((x) => classes.add(x)); },
      remove(...xs) { xs.forEach((x) => classes.delete(x)); },
      toggle(x, force) {
        if (force === undefined) {
          if (classes.has(x)) { classes.delete(x); return false; }
          classes.add(x); return true;
        }
        if (force) classes.add(x); else classes.delete(x);
        return !!force;
      },
      contains(x) { return classes.has(x); },
    },
    appendChild(child) {
      child.parentNode = this;
      child.parentElement = this;
      children.push(child);
      return child;
    },
    addEventListener() {},
    remove() {
      if (this.parentNode) this.parentNode.removeChild(this);
    },
    removeChild(child) {
      const idx = children.indexOf(child);
      if (idx >= 0) children.splice(idx, 1);
      child.parentNode = null;
      child.parentElement = null;
      return child;
    },
    querySelector(sel) {
      if (sel === 'canvas') return children.find((c) => c.tagName === 'CANVAS') || null;
      if (sel === '.tide-art') return children.find((c) => c.className === 'tide-art') || null;
      return null;
    },
    closest(sel) {
      if (sel === '.ball') return this.parentElement;
      return null;
    },
  };

  if (el.tagName === 'CANVAS') {
    let _w = 0;
    let _h = 0;
    el.resizeCount = 0;
    Object.defineProperty(el, 'width', {
      get() { return _w; },
      set(v) { _w = v; el.resizeCount++; },
    });
    Object.defineProperty(el, 'height', {
      get() { return _h; },
      set(v) { _h = v; },
    });
    const stats = {
      createImageData: 0,
      putImageData: 0,
      createPattern: 0,
      setTransform: 0,
      clip: 0,
      quadraticCurveTo: 0,
      bezierCurveTo: 0,
      fillRect: 0,
      fill: 0,
      stroke: 0,
      arc: 0,
      destOutCount: 0,
    };
    const ctx2d = {
      stats,
      _gco: 'source-over',
      get globalCompositeOperation() { return this._gco; },
      set globalCompositeOperation(v) {
        this._gco = v;
        if (v === 'destination-out') stats.destOutCount++;
      },
      globalAlpha: 1,
      fillStyle: '',
      strokeStyle: '',
      lineWidth: 1,
      lineCap: 'butt',
      lineJoin: 'miter',
      save() {},
      restore() { this._gco = 'source-over'; },
      translate() {},
      rotate() {},
      scale() {},
      setTransform() { stats.setTransform++; },
      clearRect() {},
      beginPath() {},
      closePath() {},
      moveTo() {},
      lineTo() {},
      quadraticCurveTo() { stats.quadraticCurveTo++; },
      bezierCurveTo() { stats.bezierCurveTo++; },
      arc() { stats.arc++; },
      clip() { stats.clip++; },
      fill() { stats.fill++; },
      stroke() { stats.stroke++; },
      fillRect() { stats.fillRect++; },
      createRadialGradient() {
        return { addColorStop() {} };
      },
      createLinearGradient() {
        return { addColorStop() {} };
      },
      createImageData(w, h) {
        stats.createImageData++;
        return { width: w, height: h, data: new Uint8ClampedArray(w * h * 4) };
      },
      putImageData() {
        stats.putImageData++;
      },
      createPattern(imgCanvas, repeat) {
        stats.createPattern++;
        return { _src: imgCanvas, _repeat: repeat };
      },
    };
    el.getContext = (kind) => (kind === '2d' ? ctx2d : null);
    el._ctx2d = ctx2d;
  }

  return el;
}

const createdCanvases = [];
const sandboxShots = {};
const domDocument = {
  createElement(tag) {
    const el = createMockElement(tag);
    if (tag === 'canvas') createdCanvases.push(el);
    return el;
  },
};

const appSource = fs.readFileSync(path.join(__dirname, '..', '..', 'internal', 'web', 'app.js'), 'utf8');
function extractFunc(src, name) {
  const norm = src.replace(/\r\n/g, '\n');
  const start = norm.indexOf('function ' + name + '(');
  assert(start >= 0, 'Could not locate function ' + name + ' in app.js');
  const end = norm.indexOf('\n}\n', start);
  assert(end >= 0, 'Could not locate end of function ' + name);
  return norm.slice(start, end + 2);
}
const appFxSnippet = extractFunc(appSource, 'spawnFx') + '\n' + extractFunc(appSource, 'playEffects');

const context = {
  Math,
  Number,
  String,
  Array,
  Object,
  Set,
  Map,
  JSON,
  Uint8ClampedArray,
  document: domDocument,
  fxRoot: createMockElement('div'),
  seenEffects: new Set(),
  IMMEDIATE_AUDIO: new Set(),
  ENGINE_FX: new Set(['heal', 'faction', 'hurt', 'paper-hurt', 'swap', 'wall-spawn', 'wall-fade', 'wall', 'impact', 'hit']),
  EXPORT: false,
  sandboxEffects: false,
  ownHurtKinds: new Set(),
  effectTime: () => 0,
  screenPos: (x, y, scale, cx, cy) => [cx + x * scale, cy - y * scale],
  kindColor: () => '#38bdf8',
  dmgPop: () => ({}),
  playDefaultSfx: () => {},
  lookOf: () => ({ base: 'tide', fx: ['tide'] }),
  shotFX: {},
  arena: {
    registerShot(name, fn) {
      if (!context.shotFX.tide) context.shotFX.tide = {};
      context.shotFX.tide[name] = fn;
    },
  },
  window: null,
};
context.window = context;
context.window.devicePixelRatio = 2;

vm.createContext(context);
vm.runInContext(appFxSnippet, context);
const tideJs = fs.readFileSync(path.join(__dirname, 'fx', 'tide.js'), 'utf8');
vm.runInContext(tideJs, context);

assert(typeof context.window.lookFX?.tide?.tick === 'function', 'tick exported');
assert(typeof context.window.lookFX?.tide?.guide === 'function', 'guide exported');
assert(typeof context.window.lookFX?.tide?.unmount === 'function', 'unmount exported');

// (a) Fallback DOM suppression via window.shotFX.tide
for (const name of ['tide-state', 'tide-ebb', 'tide-launch', 'tide-surf', 'tide-shed']) {
  assert(typeof context.shotFX.tide?.[name] === 'function', `shot ${name} must be registered`);
}
context.playEffects([
  { id: 1, unitId: 7, kind: '潮', name: 'tide-state', x: 0, y: 0, data: { rev: 1, ebb: { active: false }, segs: [] } },
  { id: 2, unitId: 7, kind: '潮', name: 'tide-ebb', x: 0, y: 0, vx: 90, vy: 120, amount: 80 },
  { id: 3, unitId: 7, kind: '潮', name: 'tide-launch', x: 10, y: 0, vx: 1, vy: 0, amount: 220 },
  { id: 4, unitId: 7, kind: '潮', name: 'tide-surf', x: 20, y: 0, vx: 1, vy: 0, amount: 40 },
  { id: 5, unitId: 7, kind: '潮', name: 'tide-shed', x: 25, y: 0, vx: 80, amount: 15 },
], 1, 400, 300);
assert.strictEqual(context.fxRoot.children.length, 0, 'registered no-op shots must produce 0 fallback DOM nodes');

// Build realistic rich sample state (30 wake segments + ebb + bursts + walls)
function makeSampleSegs(n) {
  const segs = [];
  for (let i = 0; i < n; i++) {
    segs.push({
      id: i + 1,
      x: -180 + i * 11,
      y: Math.sin(i * 0.28) * 18,
      fx: 0.96,
      fy: 0.28,
      hl: 7,
      nw: 34 + (i % 5) * 4,
      pw: 34 + ((i + 2) % 5) * 4,
      born: 1.0 + i * 0.04,
      life: 1.7,
      flare: i > 20 ? 2.8 : null,
    });
  }
  return segs;
}

const guideDomMap = new Map();
const makeCtx = (time, sandbox, extraEffects = []) => ({
  scale: 1.05,
  cx: 400,
  cy: 300,
  time,
  sandbox,
  walls: [
    { x1: -60, y1: -80, x2: 60, y2: -80, radius: 12, square: true },
    { x1: 80, y1: 40, x2: 140, y2: 110, radius: 10, square: false },
  ],
  effects: [
    {
      id: 100,
      unitId: 7,
      kind: '潮',
      name: 'tide-state',
      time: 2.9,
      data: {
        rev: 12,
        surfUntil: 3.4,
        ebb: { active: true, start: 2.75, until: 3.25, r: 105, power: 140 },
        segs: makeSampleSegs(30),
      },
    },
    ...extraEffects,
  ],
  seenGuides: new Set(),
  ensureGuide(key, cls) {
    if (!guideDomMap.has(key)) {
      const g = createMockElement('div');
      g.id = key;
      g.className = 'guide ' + cls;
      guideDomMap.set(key, g);
    }
    return guideDomMap.get(key);
  },
});

const unit = { id: 7, x: 150, y: 12, vx: 295, vy: 80, radius: 18 };

// First render
const ctx1 = makeCtx(3.0, false, [
  { id: 101, unitId: 7, kind: '潮', name: 'tide-shed', x: 150, y: 12, vx: 80, amount: 18, time: 2.88 },
  { id: 102, unitId: 7, kind: '潮', name: 'tide-launch', x: 150, y: 12, vx: 1, vy: 0, amount: 240, time: 2.9 },
  { id: 103, unitId: 7, kind: '潮', name: 'tide-surf', x: 140, y: 10, vx: 0.95, vy: 0.3, amount: 45, time: 2.92 },
]);
context.window.lookFX.tide.guide(unit, ctx1);
assert(ctx1.seenGuides.has('tide-floor-7'), 'guide must register id in ctx.seenGuides Set');

const floorGuide = guideDomMap.get('tide-floor-7');
const floorCanvas = floorGuide.querySelector('canvas');
assert(floorCanvas && floorCanvas.width >= 16 && floorCanvas.height >= 16, 'full-Retina floor canvas sized');
const c2d = floorCanvas._ctx2d;

// (b) Check Voronoi caustic tile generation + vector spline / clip / wall carve usage
assert.strictEqual(createdCanvases.length, 2, 'only 1 caustic tile canvas + 1 floor canvas created');
const tileCanvas = createdCanvases.find((cn) => cn !== floorCanvas);
assert.strictEqual(tileCanvas.width, 160);
assert.strictEqual(tileCanvas.height, 160);
assert.strictEqual(tileCanvas._ctx2d.stats.createImageData, 1, 'Voronoi tile created once');
assert.strictEqual(tileCanvas._ctx2d.stats.putImageData, 1, 'Voronoi tile uploaded once');
assert.strictEqual(c2d.stats.createPattern, 1, 'caustic pattern created once per floor canvas');
assert(c2d.stats.quadraticCurveTo > 50, `expected smooth quadraticCurveTo spline calls, got ${c2d.stats.quadraticCurveTo}`);
assert(c2d.stats.clip >= 3, `expected clipped caustic draws across wake/wings/ebb, got ${c2d.stats.clip}`);
assert(c2d.stats.destOutCount >= 1, 'expected destination-out wall carve pass when walls exist');

// (c) Pause determinism + rewind/sandbox reset
const snapStats = () => ({ ...c2d.stats });
const beforePause = snapStats();
context.window.lookFX.tide.guide(unit, makeCtx(3.0, false));
const afterPause1 = snapStats();
context.window.lookFX.tide.guide(unit, makeCtx(3.0, false));
const afterPause2 = snapStats();
assert.strictEqual(afterPause2.quadraticCurveTo - afterPause1.quadraticCurveTo, afterPause1.quadraticCurveTo - beforePause.quadraticCurveTo, 'identical ctx.time must be deterministic');
assert.strictEqual(afterPause2.createPattern, afterPause1.createPattern, 'pattern cached across frames');

// Sandbox reset clears state
context.window.lookFX.tide.guide(unit, {
  scale: 1,
  cx: 400,
  cy: 300,
  time: 0.1,
  sandbox: true,
  walls: [],
  effects: [],
  seenGuides: new Set(),
  ensureGuide: ctx1.ensureGuide,
});
assert.strictEqual(floorGuide._tide.segs.length, 0, 'sandbox switch must reset cached segs');
assert.strictEqual(floorGuide._tide.rev, -1, 'sandbox switch must reset rev');

// Rewind time reset clears state
context.window.lookFX.tide.guide(unit, ctx1);
assert(floorGuide._tide.segs.length > 0, 'state populated before rewind');
context.window.lookFX.tide.guide(unit, {
  scale: 1,
  cx: 400,
  cy: 300,
  time: 0.2,
  sandbox: false,
  walls: [],
  effects: [],
  seenGuides: new Set(),
  ensureGuide: ctx1.ensureGuide,
});
assert.strictEqual(floorGuide._tide.segs.length, 0, 'time rewind must reset cached segs');
assert.strictEqual(floorGuide._tide.rev, -1, 'time rewind must reset rev');

// Early eviction smooth decay: removing an active seg[0] keeps it in segPool for 0.30s dissolve
const segsFull = makeSampleSegs(30).map((s, idx) => ({ ...s, born: 2.3 + idx * 0.02 }));
context.window.lookFX.tide.guide(unit, {
  scale: 1.05,
  cx: 400,
  cy: 300,
  time: 3.0,
  sandbox: false,
  walls: [],
  effects: [{
    id: 108,
    unitId: 7,
    kind: '潮',
    name: 'tide-state',
    time: 3.0,
    data: { rev: 12, surfUntil: 0, ebb: { active: false }, segs: segsFull },
  }],
  seenGuides: new Set(),
  ensureGuide: ctx1.ensureGuide,
});
const segsEvicted = segsFull.slice(1);
context.window.lookFX.tide.guide(unit, {
  scale: 1.05,
  cx: 400,
  cy: 300,
  time: 3.05,
  sandbox: false,
  walls: [],
  effects: [{
    id: 109,
    unitId: 7,
    kind: '潮',
    name: 'tide-state',
    time: 3.05,
    data: { rev: 13, surfUntil: 0, ebb: { active: false }, segs: segsEvicted },
  }],
  seenGuides: new Set(),
  ensureGuide: ctx1.ensureGuide,
});
assert(floorGuide._tide.segPool.some((s) => s.id === 1 && s.evictedAt >= 0), 'evicted segment retained in segPool for 0.30s smooth decay');

context.window.lookFX.tide.guide(unit, {
  scale: 1.05,
  cx: 400,
  cy: 300,
  time: 3.38,
  sandbox: false,
  walls: [],
  effects: [{
    id: 110,
    unitId: 7,
    kind: '潮',
    name: 'tide-state',
    time: 3.38,
    data: { rev: 13, surfUntil: 0, ebb: { active: false }, segs: segsEvicted },
  }],
  seenGuides: new Set(),
  ensureGuide: ctx1.ensureGuide,
});
assert(!floorGuide._tide.segPool.some((s) => s.id === 1), 'evicted segment pruned once 0.30s dissolve finishes');

// Continuous live head bridge across sub-step frames (+4, +10, +16, +21 ahead of lastSeg) without changing d.segs
const bridgeBaseSegs = [{
  id: 50,
  x: 100,
  y: 0,
  fx: 1,
  fy: 0,
  hl: 11,
  nw: 45,
  pw: 45,
  born: 4.9,
  life: 1.7,
}];
for (const dx of [4, 10, 16, 21]) {
  const prevQ = c2d.stats.quadraticCurveTo;
  context.window.lookFX.tide.guide({ id: 7, x: 100 + dx, y: 0, vx: 220, vy: 0, radius: 18 }, {
    scale: 1.05,
    cx: 400,
    cy: 300,
    time: 5.0 + dx * 0.002,
    sandbox: false,
    walls: [],
    effects: [{
      id: 120,
      unitId: 7,
      kind: '潮',
      name: 'tide-state',
      time: 5.0,
      data: { rev: 20, surfUntil: 0, ebb: { active: false }, segs: bridgeBaseSegs },
    }],
    seenGuides: new Set(),
    ensureGuide: ctx1.ensureGuide,
  });
  assert(c2d.stats.quadraticCurveTo > prevQ, `live bridge renders smooth wake at dx=${dx}`);
}

// Re-populate full state for (d) & (e)
const resizesBeforeBench = floorCanvas.resizeCount;
assert.strictEqual(floorCanvas.resizeCount, 1, `canvas resized only once initially, got ${floorCanvas.resizeCount}`);

// (d) Body art tick & unmount lifecycle (matching app.js: tick(el, u, ctx) without seenGuides, and unmount(el))
context.window.lookFX.tide.guide(unit, ctx1);
const ballEl = createMockElement('div');
ballEl.className = 'ball look-tide';

context.window.lookFX.tide.tick(ballEl, unit, { time: 3.0, sandbox: false });
const artEl = ballEl.querySelector('.tide-art');
assert(artEl && artEl.innerHTML.includes('<clipPath'), 'body art must contain clipPath');
assert(artEl.style.getPropertyValue('--tide-bow-op') !== '', 'bow opacity CSS var set');
assert(artEl.style.getPropertyValue('--tide-shield-w').endsWith('px'), 'shield stroke width CSS var set');
assert(ballEl.classList.contains('tide-surfing'), 'surfing class synced cleanly via unitVis');
assert(ballEl.classList.contains('tide-ebbing'), 'ebbing class synced cleanly via unitVis');

context.window.lookFX.tide.unmount(ballEl);
assert.strictEqual(ballEl._tideArt, null, 'unmount clears body art reference');
assert.strictEqual(ballEl.classList.contains('tide-surfing'), false, 'unmount removes surfing class');

// (e) Performance benchmark: 30 frames with 30 wake segments + ebb + bursts + walls
const benchStart = process.hrtime.bigint();
for (let f = 0; f < 30; f++) {
  const t = 2.9 + f * 0.016;
  const movingUnit = { id: 7, x: 150 + f * 4, y: 12 + f * 1.5, vx: 295, vy: 80, radius: 18 };
  const bCtx = makeCtx(t, false, [
    { id: 201, unitId: 7, kind: '潮', name: 'tide-shed', x: movingUnit.x, y: movingUnit.y, vx: 80, amount: 18, time: 2.88 },
  ]);
  context.window.lookFX.tide.guide(movingUnit, bCtx);
  context.window.lookFX.tide.tick(ballEl, movingUnit, { time: t, sandbox: false });
}
const benchEnd = process.hrtime.bigint();
assert.strictEqual(floorCanvas.resizeCount, resizesBeforeBench, 'canvas must perform 0 per-frame resizes while unit moves');
const msPerFrame = Number(benchEnd - benchStart) / 1e6 / 30;
console.log(`PASS: tide_fx_test (Retina spline + Voronoi caustics, ${msPerFrame.toFixed(3)} ms/frame over 30 frames)`);
assert(msPerFrame < 4.0, `Expected < 4.0 ms/frame, got ${msPerFrame.toFixed(3)}`);
