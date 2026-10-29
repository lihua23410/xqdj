const test = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const vm = require('node:vm');

function setup(mode) {
  const voices = [];
  function element() {
    return {
      children: [], dataset: {}, classList: { add() {} },
      get childElementCount() { return this.children.length; },
      appendChild(el) { this.children.push(el); el.parent = this; },
      remove() { if (this.parent) this.parent.children = this.parent.children.filter(el => el !== this); },
    };
  }
  const sandbox = {
    window: {},
    document: { createElement: element, currentScript: { dataset: { pack: '/ball/非因果' } } },
  };
  vm.createContext(sandbox);
  vm.runInContext(fs.readFileSync(__dirname + '/../../internal/web/presentation.js', 'utf8'), sandbox);
  const player = sandbox.window.presentationPlayer;
  sandbox.arena = {
    SOUND_MODE: { FORCE: 3 },
    registerPresentation(type, renderer) { player.register('/ball/非因果', type, renderer); },
  };
  vm.runInContext(fs.readFileSync(__dirname + '/fx/rewind.js', 'utf8'), sandbox);
  const context = {
    host: element(), mode, scale: 1, cx: 0, cy: 0,
    createVoice(src, options) {
      const voice = { src, options, playing: true, disposals: 0,
        dispose() { this.playing = false; this.disposals++; },
      };
      voices.push(voice);
      return voice;
    },
  };
  const list = [{ id: 'probe-1', pack: '非因果', type: 'rewind', duration: 7, audio: true,
    data: { death: 60, frames: [{ t: 0, b: [] }] },
  }];
  return { player, context, list, voices };
}

for (const mode of ['live', 'export']) {
  test(`${mode}: probe sound loops in mode 3 and stops at the exact segment boundary`, () => {
    const { player, context, list, voices } = setup(mode);
    assert.equal(player.draw(list, 0, context), true);
    assert.equal(voices.length, 1);
    const voice = voices[0];
    assert.equal(voice.src, '/ball/非因果/fx/probe-hum.mp3');
    assert.equal(voice.options.mode, 3);
    assert.equal(voice.options.loop, true);
    // 经过素材原长 4.85 秒后仍是同一个循环声音，不能逐帧重开。
    player.draw(list, 5, context);
    player.draw(list, 6.999, context);
    assert.equal(voices.length, 1);
    assert.equal(voice.playing, true);
    assert.equal(player.draw(list, 7, context), false);
    assert.equal(voice.playing, false);
    assert.equal(voice.disposals, 1);
    assert.equal(context.host.children.length, 0);
    player.clear();
    assert.equal(voice.disposals, 1);
  });
}

test('export preview is silent; playback remounts and early cancellation cuts the loop', () => {
  const { player, context, list, voices } = setup('export');
  player.draw(list, 0, { ...context, preview: true });
  assert.equal(voices.length, 0);
  player.draw(list, 0, context);
  assert.equal(voices.length, 1);
  player.clear();
  assert.equal(voices[0].playing, false);
  assert.equal(voices[0].disposals, 1);
});

test('switching away from a probe stops its mode 3 voice before mounting the next segment', () => {
  const { player, context, list, voices } = setup('export');
  let nextMounted = false;
  player.register('/ball/test', 'next', {
    mount() { assert.equal(voices[0].playing, false); nextMounted = true; },
    render() {}, destroy() {},
  });
  list.push({ id: 'next-1', pack: 'test', type: 'next', duration: 1 });
  player.draw(list, 0, context);
  assert.equal(player.draw(list, 7, context), true);
  assert.equal(nextMounted, true);
  assert.equal(voices[0].disposals, 1);
  player.clear();
});

test('export keeps looping audio active across source boundaries while one-shot audio ends', () => {
  const source = fs.readFileSync(__dirname + '/../../internal/web/export.js', 'utf8');
  const start = source.indexOf('  function tickMedia()');
  const end = source.indexOf('  function audioPath(', start);
  assert.ok(start >= 0 && end > start);
  const loop = { loop: true, duration: 4.85, __exportPlaying: true,
    __exportAnchorFrame: 0, __exportAnchorTime: 0, currentTime: 0 };
  const once = { ...loop, loop: false };
  const playingMedia = new Set([loop, once]);
  const sandbox = vm.createContext({ playingMedia, frameNo: 360, FPS: 60 });
  vm.runInContext(source.slice(start, end), sandbox);
  vm.runInContext('tickMedia()', sandbox);
  assert.equal(loop.__exportPlaying, true);
  assert.equal(playingMedia.has(loop), true);
  assert.ok(Math.abs(loop.currentTime - (6 % 4.85)) < 1e-9);
  assert.equal(once.__exportPlaying, false);
  assert.equal(playingMedia.has(once), false);
  assert.equal(once.currentTime, 4.85);
  sandbox.frameNo = 582;
  vm.runInContext('tickMedia()', sandbox);
  assert.equal(loop.__exportPlaying, true);
  assert.ok(Math.abs(loop.currentTime) < 1e-9);
});
