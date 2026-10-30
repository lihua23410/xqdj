const test = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const vm = require('node:vm');

function setup() {
  const shots = {}, voices = [], events = [], visuals = [];
  const root = {};
  const arena = {
    SOUND_MODE: { NORMAL:0, PREEMPT:1, IMMUNE:2, FORCE:3 },
    registerShot(name, handler) { shots[name] = handler; },
    spawnFx(name, x, y, kind, style) { visuals.push({ name, style }); },
    burst() {},
    voice(audio, options) {
      const voice = {
        audio, options,
        stop() { events.push(['stop', audio.src]); },
        reset() { audio.currentTime = 0; events.push(['reset', audio.src]); },
        play() { events.push(['play', audio.src, audio.volume, audio.currentTime]); },
      };
      voices.push(voice);
      return voice;
    },
  };
  const sandbox = {
    window:{ arena },
    document:{ currentScript:{ dataset:{ pack:'/ball/反重力猫咪' } }, getElementById:()=>root },
    Audio:class { constructor(src) { this.src = src; this.volume = 1; this.currentTime = 0.5; } },
  };
  vm.runInNewContext(fs.readFileSync(__dirname+'/fx/shot.js','utf8'), sandbox);
  const fire = (name, time, payload = {}) => shots[name](payload, { x:10, y:20, kind:'反重力猫咪', scale:1, time });
  return { fire, voices, events, visuals };
}

test('cat calls and star cues follow the actual events and restart from the beginning', () => {
  const { fire, voices, events, visuals } = setup();
  const cases = [
    ['hajimi-meow',['meow']], ['hajimi-fire',['twinkle']],
    ['gravity-field',['meow','starlight']], ['gravity-twist',['twinkle']],
    ['gravity-convert',['twinkle']], ['gravity-capture',['twinkle']],
    ['gravity-release',['starlight']],
  ];
  cases.forEach(([event, expected], i) => {
    events.length = 0;
    fire(event, i * 2);
    assert.deepEqual(events.filter(e=>e[0]==='play').map(e=>e[1].split('/').pop()), expected.map(n=>n+'.wav'), event);
    for (let j=0;j<events.length;j+=3) {
      assert.deepEqual(events.slice(j,j+3).map(e=>e[0]), ['stop','reset','play']);
      assert.equal(events[j+2][3], 0);
    }
  });
  assert.ok(voices.every(v=>v.audio.preload==='auto' && v.audio.src.startsWith('/ball/反重力猫咪/fx/')));
  assert.ok(voices.filter(v=>v.audio.src.endsWith('/meow.wav')).every(v=>v.options.mode===2));
  assert.ok(voices.filter(v=>v.audio.src.endsWith('/twinkle.wav')).every(v=>v.options.mode===0 && v.options.priority===205));
  const field = visuals.find(v=>v.name==='fx-hajimi-field');
  assert.equal(field.style['--r'], '840px');
  assert.equal(field.style['--duration'], '3s');
});

test('a batch of captured, converted and released stars stays quiet enough', () => {
  const { fire, events } = setup();
  for (let i=0;i<16;i++) {
    fire('gravity-convert',20);
    fire('gravity-capture',20);
    fire('gravity-release',20);
  }
  assert.equal(events.filter(e=>e[0]==='play' && e[1].endsWith('/twinkle.wav')).length,1);
  assert.equal(events.filter(e=>e[0]==='play' && e[1].endsWith('/starlight.wav')).length,1);
  fire('hajimi-fire',20.12);
  assert.equal(events.filter(e=>e[0]==='play' && e[1].endsWith('/twinkle.wav')).length,2);
  fire('hajimi-fire',0);
  assert.equal(events.filter(e=>e[0]==='play' && e[1].endsWith('/twinkle.wav')).length,3);
});
