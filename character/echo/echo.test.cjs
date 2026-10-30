const test = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const vm = require('node:vm');

function setup() {
  const timers = new Map();
  const shots = {}, audioEvents = [], voices = [];
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
  const arena = {
    SOUND_MODE: { NORMAL: 0, PREEMPT: 1, IMMUNE: 2, FORCE: 3 },
    screenPos: (x, y) => [x, y],
    registerShot(name, fn) { shots[name] = fn; },
    spawnFx: () => element(),
    voice(audio, options) {
      const voice = {
        audio, options,
        stop() { audioEvents.push(['stop', audio.src]); },
        reset() { audio.currentTime = 0; audioEvents.push(['reset', audio.src]); },
        play() { audioEvents.push(['play', audio.src, audio.volume, audio.currentTime]); },
      };
      voices.push(voice);
      return voice;
    },
  };
  const sandbox = {
    window: { arena }, document: { createElement: () => element() },
    Audio: class { constructor(src) { this.src = src; this.currentTime = 0; this.volume = 1; } },
    setTimeout(fn, delay) { const id = ++timerID; timers.set(id, { fn, delay }); return id; },
    clearTimeout(id) { timers.delete(id); },
  };
  vm.runInNewContext(fs.readFileSync(__dirname + '/fx/echo.js', 'utf8'), sandbox);
  const root = element(), owner = element(['look-echo']);
  const tick = (time, marks) => sandbox.window.lookFX.echo.tick(owner, { id: 1, slot: 0 }, {
    time, scale: 1, cx: 0, cy: 0, fxRoot: root,
    units: [{ id: 9, slot: 1, radius: 18, x: 50, y: 0, marks }],
  });
  return { root, owner, tick, timers, shots, audioEvents, voices };
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

test('enabled sounds retrigger from the start while dash and release stay silent', () => {
  const { shots, audioEvents, voices } = setup();
  const expected = {
    'echo-scan':'scan', 'echo-fire':'fire', 'echo-burst':'burst', 'echo-ping':'ping',
    'echo-found':'lock', 'echo-lock':'aim', 'echo-ram':'ram',
    'echo-wall':'wall', 'echo-bounce':'bounce', 'echo-fade':'fade',
  };
  for (const voice of voices) voice.audio.currentTime = 0.2;
  for (const [event, file] of Object.entries(expected)) {
    audioEvents.length = 0;
    shots[event]({ amount: 3 }, { x: 0, y: 0, kind: '回声', time: 1 });
    assert.deepEqual(audioEvents.map(e => e[0]), ['stop', 'reset', 'play'], event);
    assert.ok(audioEvents[2][1].endsWith(`/fx/${file}.wav`), event);
    assert.equal(audioEvents[2][3], 0, event);
    assert.ok(audioEvents[2][2] > 0 && audioEvents[2][2] <= 1, event);
  }
  assert.ok(voices.filter(v => v.audio.src.endsWith('/lock.wav')).every(v => v.options.mode === 2));
  assert.ok(voices.filter(v => v.audio.src.endsWith('/ram.wav')).every(v => v.options.mode === 1 && v.options.priority === 130));
  audioEvents.length = 0;
  shots['echo-dash']({}, { x: 0, y: 0, time: 2 });
  assert.equal(audioEvents.length, 0);
  assert.equal(shots['echo-release'], undefined);
  assert.equal(voices.some(v => /\/(dash|release)\.wav$/.test(v.audio.src)), false);
});

test('sonar storms are limited by match time without hiding lock and ram cues', () => {
  const { shots, audioEvents } = setup();
  const ctx = time => ({ x: 0, y: 0, kind: '回声', time });
  for (let i = 0; i < 16; i++) shots['echo-ping']({ amount: 9 }, ctx(1));
  assert.equal(audioEvents.filter(e => e[0] === 'play').length, 1);
  shots['echo-ping']({ amount: 9 }, ctx(1.06));
  shots['echo-found']({}, ctx(1.06));
  shots['echo-ram']({}, ctx(1.06));
  assert.equal(audioEvents.filter(e => e[0] === 'play').length, 4);
  // A tenth visual ping must not duplicate the authoritative lock sound.
  shots['echo-ping']({ amount: 10 }, ctx(1.2));
  assert.equal(audioEvents.filter(e => e[0] === 'play').length, 4);
  // New matches reset simulation time and must not inherit the previous cooldown.
  shots['echo-ping']({ amount: 1 }, ctx(0));
  assert.equal(audioEvents.filter(e => e[0] === 'play').length, 5);
});

test('scan decay makes a quiet release cue, without replaying a frozen lock', () => {
  const { tick, audioEvents } = setup();
  const scan = n => [{ kind: '回声计数1', stacks:n, lifetime:10, nextDecayAt:11 }];
  tick(1, scan(3));
  tick(2, scan(2));
  assert.equal(audioEvents.filter(e => e[0] === 'play' && e[1].endsWith('/fade.wav')).length, 1);
  audioEvents.length = 0;
  tick(3, [{kind:'回声计数1锁定',stacks:10,lifetime:6,nextDecayAt:9}]);
  tick(4, [{kind:'回声计数1锁定',stacks:10,lifetime:6,nextDecayAt:9}]);
  assert.equal(audioEvents.length, 0);
});
