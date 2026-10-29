const test = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const vm = require('node:vm');

test('discard presentation callbacks while preserving the battle clock', () => {
  const events = [];
  const context = vm.createContext({ URLSearchParams, console, location: {search:'?export=test'}, performance:{}, Date:class extends Date{}, window:{requestAnimationFrame(){},setTimeout(){},clearTimeout(){}} });
  vm.runInContext(fs.readFileSync(__dirname+'/export-clock.js','utf8'),context);
  const w=context.window;
  w.setTimeout(()=>events.push('battle timer'),10);
  w.requestAnimationFrame(()=>events.push('battle raf'));
  const mark=w.__clock.checkpoint();
  w.setTimeout(()=>events.push('intro timer'),0);
  w.requestAnimationFrame(()=>events.push('intro raf'));
  assert.equal(w.__clock.now,0);
  w.__clock.discardSince(mark);
  w.__clock.set(10);w.__clock.run();w.__clock.flush(10);
  assert.deepEqual(events,['battle timer','battle raf']);
});
