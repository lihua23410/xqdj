const test = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const vm = require('node:vm');

function setup() {
  const timers = new Map();
  let timerID = 0;
  function element(classes = []) {
    const names = new Set(classes);
    return {
      children: [], dataset: {}, isConnected: true,
      style: { values: new Map(), setProperty(k, v) { this.values.set(k, v); } },
      classList: {
        contains: k => names.has(k),
        toggle(k, on) { if (on) names.add(k); else names.delete(k); },
        remove: k => names.delete(k),
      },
      appendChild(el) { this.children.push(el); el.parentElement = this; },
      remove() { if (this.parentElement) this.parentElement.children = this.parentElement.children.filter(x => x !== this); },
    };
  }
  const arena = { screenPos: (x, y) => [x, y], registerShot() {} };
  const sandbox = {
    window: { arena }, document: { createElement: () => element() },
    setTimeout(fn, delay) { const id = ++timerID; timers.set(id, { fn, delay }); return id; },
    clearTimeout(id) { timers.delete(id); },
  };
  vm.runInNewContext(fs.readFileSync(__dirname + '/fx/echo.js', 'utf8'), sandbox);
  const root = element(), owner = element(['look-echo']);
  const tick = (time, marks) => sandbox.window.lookFX.echo.tick(owner, { id: 1, slot: 0 }, {
    time, scale: 1, cx: 0, cy: 0, fxRoot: root,
    units: [{ id: 9, slot: 1, radius: 18, x: 50, y: 0, marks }],
  });
  return { root, owner, tick, timers };
}

test('full ring stays bright for six seconds, clears, then displays the next scan cycle', () => {
  const { root, owner, tick, timers } = setup();
  const scan = (stacks, nextDecayAt) => ({ kind: '回声计数1', stacks, lifetime: 10, nextDecayAt });
  tick(1, [scan(9, 11)]);
  const ring = root.children[0].children[0];
  assert.equal(ring.style.values.get('--scan'), '324deg');
  const frozen = { kind: '回声计数1锁定', stacks: 10, lifetime: 6, nextDecayAt: 7.1 };
  tick(1.1, [frozen, scan(1, 11.12)]);
  assert.equal(ring.style.values.get('--scan'), '360deg');
  assert.equal(ring.style.values.get('--left'), '1.000');
  assert.ok(ring.classList.contains('locked'));
  assert.ok(owner.classList.contains('echo-locked'));
  assert.equal(timers.size, 1);
  assert.ok(Math.abs([...timers.values()][0].delay - 6080) < 1e-8);
  tick(6.9, [frozen]);
  assert.equal(ring.style.values.get('--left'), '1.000');
  // 清理尚未落地的旧扫描层不能在到期时闪成下一轮蓝环。
  tick(7.1, [scan(1, 11.12)]);
  assert.equal(ring.style.display, 'none');
  assert.equal(owner.classList.contains('echo-locked'), false);
  tick(7.12, []);
  assert.equal(root.children[0].children.length, 0);
  assert.equal(timers.size, 0);
  tick(7.2, [scan(1, 17.2)]);
  const next = root.children[0].children[0];
  assert.equal(next.style.values.get('--scan'), '36deg');
  assert.equal(next.classList.contains('locked'), false);
});

test('another echo lock does not draw or light this echo ring', () => {
  const { root, owner, tick } = setup();
  tick(1, [{ kind: '回声计数2锁定', stacks: 10, lifetime: 6, nextDecayAt: 7 }]);
  assert.equal(root.children[0].children.length, 0);
  assert.equal(owner.classList.contains('echo-locked'), false);
});
