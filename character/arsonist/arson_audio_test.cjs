const test = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const vm = require('node:vm');
const crypto = require('node:crypto');

function setup() {
 const hooks={}, voices=[], plays=[];
 const sandbox={window:{},document:{currentScript:{dataset:{pack:'/ball/纵火者'}}},Audio:class {
  constructor(src){this.src=src}
  play(){throw Error('audio must go through arena.voice')}
 }};
 sandbox.window.arena={SOUND_MODE:{NORMAL:0},registerShot(name,fn){hooks[name]=fn},voice(audio,opts){
  const voice={audio,opts,ops:[],stop(){this.ops.push('stop')},reset(){this.ops.push('reset')},play(){this.ops.push('play');plays.push(this)}};
  voices.push(voice);return voice;
 }};
 vm.createContext(sandbox);
 vm.runInContext(fs.readFileSync(path.join(__dirname,'fx/audio.js'),'utf8'),sandbox);
 return{hooks,voices,plays,observe:(time,active)=>sandbox.window.arsonSFX.observe({time},active)};
}

test('bottles have separate sounds; repeated snapshots do not repeat a shatter',()=>{
 const s=setup(),oil=s.hooks['arson-oil-break'],fire=s.hooks['arson-fire-break'];
 for(let i=0;i<60;i++)oil({unitId:3},{time:1});
 oil({unitId:3},{time:1.5});
 assert.equal(s.plays.length,1);
 assert.ok(s.plays[0].audio.src.endsWith('/oil-break.wav'));
 fire({unitId:4},{time:1.5,visualOnly:true});assert.equal(s.plays.length,1);
 fire({unitId:4},{time:1.5});assert.equal(s.plays.length,2);
 assert.ok(s.plays[1].audio.src.endsWith('/fire-break.wav'));
 for(const v of s.plays)assert.deepEqual(v.ops.slice(-3),['stop','reset','play']);
});
test('dense fire shares one rate limit, ends naturally, and replay rewind resets it',()=>{
 const s=setup();
 for(let i=0;i<2000;i++)s.observe(0,true);
 assert.equal(s.plays.length,1);
 s.observe(.5,true);assert.equal(s.plays.length,1);
 s.observe(1.01,true);assert.equal(s.plays.length,2);
 assert.notEqual(s.plays[0].audio.src,s.plays[1].audio.src,'alternate natural crackle recordings');
 s.observe(2.1,false);assert.equal(s.plays.length,2,'no fire produces no new bed');
 s.observe(0,true);assert.equal(s.plays.length,3,'rewind can play the earlier fire again');
 assert.equal(s.voices.length,6,'fixed pool, not one sound object per cell');
 for(const v of s.voices){assert.equal(v.opts.mode,0);assert.equal(v.opts.volume,1);assert.equal(v.opts.loop,false);assert.ok(v.opts.priority>=210)}
});
test('rapid bottle events are bounded and share their family cooldown',()=>{
 const s=setup();const oil=s.hooks['arson-oil-break'];
 for(let i=0;i<1000;i++)oil({unitId:i},{time:1});
 assert.equal(s.plays.length,1);
 oil({unitId:1001},{time:1.11});assert.equal(s.plays.length,2);
 assert.equal(s.voices.length,6);
});
test('all four embedded audio files match their measured provenance',()=>{
 const records=JSON.parse(fs.readFileSync(path.join(__dirname,'fx/audio-sources.json'),'utf8'));
 assert.equal(records.length,4);
 for(const record of records){
  const file=fs.readFileSync(path.join(__dirname,'fx',record.file));
  assert.equal(crypto.createHash('sha256').update(file).digest('hex'),record.sha256);
  assert.equal(file.toString('ascii',0,4),'RIFF');assert.equal(file.toString('ascii',8,12),'WAVE');
  assert.equal(record.sample_rate,44100);assert.ok(record.duration_seconds>0 && record.duration_seconds<1.2);
  assert.ok(record.true_peak_dbfs<=-1);assert.ok(record.integrated_lufs>-60);
 }
});
