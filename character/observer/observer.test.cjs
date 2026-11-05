const test = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const vm = require('node:vm');

function setup() {
  class Element {
    constructor(tag = 'div') {
      this.tagName = tag; this.children = []; this.attributes = {}; this.names = new Set();
      this.style = { values: new Map(), setProperty(k, v) { this.values.set(k, v); }, removeProperty(k) { this.values.delete(k); } };
      this.classList = {
        contains: n => this.names.has(n), add: (...ns) => ns.forEach(n => this.names.add(n)),
        remove: (...ns) => ns.forEach(n => this.names.delete(n)),
        toggle: (n, on) => on ? this.names.add(n) : this.names.delete(n),
      };
    }
    set className(n) { this.names = new Set(n.split(/\s+/).filter(Boolean)); }
    get className() { return [...this.names].join(' '); }
    get firstChild() { return this.children[0] || this.innerHTML || null; }
    appendChild(n) { n.parentElement = this; this.children.push(n); return n; }
    remove() { if (this.parentElement) this.parentElement.children = this.parentElement.children.filter(n => n !== this); }
    replaceChildren() { this.children = []; this.innerHTML = ''; }
    setAttribute(k, v) { this.attributes[k] = v; if (k === 'id') this.id = v; }
    querySelector(s) { return this.children.find(n => n.classList.contains(s.replace(':scope > .', ''))) || null; }
  }
  const body = new Element(), head = new Element(), guides = body.appendChild(new Element());
  const find = (n, id) => n.id === id ? n : n.children.map(c => find(c, id)).find(Boolean);
  const document = { body, head, documentElement: body,
    getElementById: id => find(body, id) || find(head, id),
    createElement: tag => new Element(tag), createElementNS: (_, tag) => new Element(tag),
  };
  const shots = {}, paints = [];
  const arena = { registerShot: (n, f) => { shots[n] = f; }, registerUnitPaint: f => paints.push(f),
    screenPos: (x, y, scale, cx, cy) => [cx + x * scale, cy - y * scale] };
  const box = { window: { arena }, arena, document };
  vm.runInNewContext(fs.readFileSync(__dirname + '/fx/observer.js', 'utf8'), box);
  const owners = new Map();
  function owner(id, slot = 0) {
    const el = body.appendChild(new Element()); el.className = 'ball look-observer';
    const u = { id, slot, kind: '观察者', role: 'fighter', x: 0, y: 0, radius: 18 };
    owners.set(id, { el, u }); return el;
  }
  function render(time, events = [], extra = [{ id: 90, kind: '靶子', x: 100, y: 0 }]) {
    const seen = new Set();
    const ctx = { time, effects: events, units: [...owners.values()].map(o => o.u).concat(extra), scale: 1, cx: 100, cy: 100,
      seenGuides: seen, ensureGuide(id, cls) {
        let n = document.getElementById(id);
        if (!n) { n = guides.appendChild(new Element()); n.id = id; n.className = 'guide ' + cls; }
        return n;
      }, placeSeg: () => {} };
    for (const { u } of owners.values()) box.window.lookFX.observer.guide(u, ctx);
    for (const n of [...guides.children]) if (!seen.has(n.id)) n.remove();
    for (const { el, u } of owners.values()) box.window.lookFX.observer.tick(el, u, ctx);
    return ctx;
  }
  const fx = (name, values = {}) => ({ name: 'observer-' + name, kind: '观察者', unitId: 1, x: 0, y: 0, ...values });
  const state = id => document.getElementById('observer-state-' + id)?._observer;
  return { box, body, guides, document, owner, owners, render, fx, state, shots, paints };
}
const child = (el, name) => el.querySelector(':scope > .' + name);

test('special film counts ordinary projectiles when it contains no beams', () => {
  const s = setup(), el = s.owner(1);
  s.render(1, [s.fx('state', { vx: 4, vy: 90 }), s.fx('film-state', { amount: 0, vx: 5, vy: 0 })]);
  const echo = child(child(el, 'observer-films'), 'observer-echo');
  assert.equal(echo.classList.contains('active'), true);
  assert.equal(echo.classList.contains('has-beams'), false);
  assert.equal(child(echo, 'observer-echo-count').textContent, '×5');
  assert.match(echo.attributes['aria-label'], /共 5 发，其中视线 0 发/);
  assert.equal(echo.attributes.title, echo.attributes['aria-label']);
  assert.equal(echo.style.values.get('--film-layers'), '3');
});

test('gauge percentage and film report actual archived beams and pending shots', () => {
  const s = setup(), el = s.owner(1);
  s.render(1, [s.fx('state', { amount: 75, vx: 6, vy: 90 }), s.fx('film-state', { amount: 3, vx: 8, vy: 17 })]);
  const gauge = child(el, 'observer-gauge-wrap');
  assert.equal(child(gauge, 'observer-gauge-value').textContent, '75%');
  assert.equal(child(child(gauge, 'observer-gauge-track'), 'observer-gauge-fill').style.transform, 'scaleX(0.75)');
  assert.equal(child(gauge, 'observer-replay-status').textContent, '复现 · 17 发');
  const echo = child(child(el, 'observer-films'), 'observer-echo');
  assert.equal(child(echo, 'observer-echo-count').textContent, '×8');
  assert.match(echo.attributes['aria-label'], /共 8 发，其中视线 3 发/);
});

test('long volleys follow pending shots instead of a fixed 1.3 second timer; pause holds the eye', () => {
  const s = setup(), el = s.owner(1);
  s.render(1, [s.fx('state', { amount: 0, vx: 4, vy: 90 }), s.fx('replay', { amount: 40 })]);
  s.render(4, [s.fx('film-state', { amount: 9, vx: 20, vy: 20 })]);
  assert.equal(el.classList.contains('observer-replaying'), true);
  const vars = [...el.style.values], iris = child(child(el, 'observer-eye'), 'observer-iris').style.transform;
  s.render(4); s.render(4);
  assert.deepEqual([...el.style.values], vars);
  assert.equal(child(child(el, 'observer-eye'), 'observer-iris').style.transform, iris);
  s.render(4.1, [s.fx('film-state', { amount: 10, vx: 40, vy: 0 })]);
  assert.equal(el.classList.contains('observer-replaying'), false);
});

test('new-round film colors use their slot and never inherit discarded colors', () => {
  const s = setup(), el = s.owner(1);
  s.render(1, [s.fx('record', { amount: 40, vx: 1 }), s.fx('record', { amount: 120, vx: 2 })]);
  s.render(2, [s.fx('replay', { amount: 2 }), s.fx('state', { vx: 4, vy: 90 })]);
  assert.equal(s.state(1).tones.length, 0);
  s.render(3, [s.fx('record', { amount: 250, vx: 1 }), s.fx('state', { vx: 5, vy: 90 })]);
  const films = child(el, 'observer-films');
  assert.equal(child(films, 'observer-film-0').style.values.get('--hue'), '250');
  assert.equal(child(films, 'observer-film-1').classList.contains('active'), false);
});

test('transient captures deduplicate repeated frames, pause, expire by match time and use valid coordinates', () => {
  const s = setup(); s.owner(1);
  const events = [s.fx('idle'), s.fx('record', { amount: 190, vx: 1, x: 80, y: 40 })];
  s.render(1, events); s.render(1, events);
  assert.equal(s.state(1).effects.length, 1);
  s.owners.get(1).u.x = 30;
  s.render(1.2);
  const e = s.state(1).effects[0];
  assert.ok(Number.isFinite(parseFloat(e.node.style.left)));
  const before = [...e.node.style.values];
  s.render(1.2);
  assert.deepEqual([...e.node.style.values], before);
  s.render(1.4);
  assert.ok(Math.abs(parseFloat(e.node.style.left) - 130) < .2, 'capture should finish at the moving owner');
  s.render(1.5);
  assert.equal(s.state(1).effects.length, 0);
});

test('owner removal and match rewind clear state without affecting another observer', () => {
  const s = setup(), a = s.owner(1), b = s.owner(2);
  s.render(10, [s.fx('state', { amount: 90, vx: 7, vy: 90 }), s.fx('film-state', { amount: 8, vx: 12 }),
    s.fx('film-state', { unitId: 2, amount: 2, vx: 3 })]);
  assert.equal(s.state(1).beams, 8); assert.equal(s.state(2).beams, 2);
  s.owners.delete(1); a.remove(); s.render(11);
  assert.equal(s.state(1), undefined); assert.equal(s.state(2).beams, 2);
  s.render(0);
  assert.equal(s.state(2).beams, null); assert.equal(s.state(2).gauge, 0);
  const echo = child(child(b, 'observer-films'), 'observer-echo');
  assert.equal(child(echo, 'observer-echo-count').textContent, '');
  s.render(1, [s.fx('film-state', { unitId: 2, amount: 0, vx: 0 })]);
  assert.equal(child(echo, 'observer-echo-count').textContent, '×0');
  s.box.window.lookFX.observer.unmount(b);
  assert.equal(b.children.length, 0); assert.equal(b.style.values.size, 0);
});

test('beam orientation uses travel direction; copied shapes keep body tint', () => {
  const s = setup(), el = s.owner(1);
  const hook = s.box.window.lookFX['observer-beam'];
  hook.tick(el, { vx: 0, vy: 680 }, { time: 2 });
  assert.equal(child(el, 'observer-beam-img').style.transform, `rotate(${-Math.PI / 2}rad)`);
  const units = [{ id: 1, kind: '观察者' }], paint = s.paints[0];
  assert.equal(paint({ role: 'projectile', kind: '子弹', ownerId: 1 }, units), '#b9d7df');
  assert.equal(paint({ role: 'projectile', kind: '观察者苦无复现', ownerId: 1 }, units), '#b9d7df');
  assert.equal(paint({ role: 'projectile', kind: '观察者视线', ownerId: 1 }, units), '');
  assert.equal(paint({ role: 'projectile', kind: '子弹', ownerId: 2 }, units), '');
});
