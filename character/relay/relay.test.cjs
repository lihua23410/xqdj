const test=require('node:test');
const assert=require('node:assert/strict');
const fs=require('node:fs');
const vm=require('node:vm');

function setup() {
  const shots={}, voices=[], events=[], visuals=[];
  class Element {
    constructor(tag='div') { this.tag=tag;this.children=[];this.attrs={};this.style={setProperty:(k,v)=>this.attrs[k]=v,removeProperty:k=>delete this.attrs[k]};this.classes=new Set();this.classList={toggle:(k,v)=>v?this.classes.add(k):this.classes.delete(k),remove:(...ks)=>ks.forEach(k=>this.classes.delete(k))}; }
    setAttribute(k,v) { this.attrs[k]=v; }
    appendChild(c) { c.parent=this;this.children.push(c); }
    remove() { if(this.parent)this.parent.children=this.parent.children.filter(c=>c!==this); }
    set innerHTML(value) { this.html=value; }
    querySelector(s) { if(s==='svg')return this.children.find(c=>c.tag==='svg');if(s==='.relay-face')return this.html?.includes('relay-face')?{}:null;if(s==='.relay-datagram')return this.html?.includes('relay-datagram')?{}:null; }
    querySelectorAll() { return []; }
  }
  const arena={
    SOUND_MODE:{NORMAL:0,IMMUNE:2},registerShot:(n,f)=>shots[n]=f,
    spawnFx(name,x,y,kind,style){const el=new Element();visuals.push({name,style,el});return el;},
    voice(audio,opts){const v={audio,opts,stop(){events.push(['stop',audio.src]);},reset(){audio.currentTime=0;events.push(['reset',audio.src]);},play(){events.push(['play',audio.src,audio.currentTime]);}};voices.push(v);return v;}
  };
  const sandbox={window:{arena},document:{currentScript:{dataset:{pack:'/ball/中转站'}},createElementNS:(ns,tag)=>new Element(tag)},Audio:class{constructor(src){this.src=src;this.currentTime=1;}}};
  vm.createContext(sandbox);
  for(const file of ['relay.js','shot.js'])vm.runInContext(fs.readFileSync(__dirname+'/fx/'+file,'utf8'),sandbox);
  const fire=(name,time,fx={})=>shots['relay-'+name](fx,{time,x:10,y:20,scale:1,kind:'中转站'});
  const guides=new Map();
  const context=(time,units)=>({time,units,scale:2,cx:100,cy:100,seenGuides:new Set(),ensureGuide(id){if(!guides.has(id)){const el=new Element();el.id=id;guides.set(id,el);}return guides.get(id);}});
  return {hooks:sandbox.window.lookFX,fire,events,voices,visuals,Element,context,guides};
}

test('routing sounds share a traffic budget and each retrigger starts at zero',()=>{
  const {fire,events,voices,visuals}=setup();
  for(let i=0;i<100;i++){fire('forward',1,{unitId:4,amount:9});fire('arrive',1,{unitId:9});fire('join',1,{unitId:4});fire('hit',1);}
  assert.equal(events.filter(e=>e[0]==='play').length,3);
  fire('arrive',1.16,{unitId:9});
  assert.equal(events.filter(e=>e[0]==='play').length,4);
  fire('forward',0,{unitId:4,amount:9});
  assert.equal(events.filter(e=>e[0]==='play').length,5,'new match resets traffic cooldown');
  for(let i=0;i<events.length;i+=3){assert.deepEqual(events.slice(i,i+3).map(e=>e[0]),['stop','reset','play']);assert.equal(events[i+2][2],0);}
  assert.ok(voices.every(v=>v.audio.preload==='auto'));
  assert.ok(voices.filter(v=>v.audio.src.endsWith('neighbor.wav')).every(v=>v.opts.mode===2));
  assert.equal(visuals.filter(v=>v.name==='fx-relay-tx').length,2,'one transient per router in a dense batch, reset on new match');
});

test('active links follow the actual target, isolate owners, expire and reset',()=>{
  const {hooks,fire,context,guides}=setup();
  const body={id:1,kind:'中转站',x:0,y:0};
  const a={id:4,kind:'中转站随从',ownerId:1,x:40,y:20};
  const b={id:9,kind:'中转站随从',ownerId:1,x:-70,y:0};
  const foreign={id:12,kind:'中转站随从',ownerId:8,x:5,y:0};
  const units=[body,a,b,foreign];
  fire('forward',1,{unitId:4,amount:9});
  hooks.relay.guide(body,context(1,units));
  const map=guides.get('guide-relay-1').querySelector('svg');
  assert.equal(map.children.length,3);
  const route=map.children.find(e=>e.attrs.class==='relay-route');
  assert.equal(route.attrs.x1,180);assert.equal(route.attrs.y1,60);assert.equal(route.attrs.x2,-40);
  hooks.relay.guide(body,context(1.1,units));
  assert.equal(map.children.find(e=>e.attrs.class==='relay-route'),route,'reuse DOM on each frame');
  hooks.relay.guide(body,context(1.8,units));assert.equal(map.children.length,2);
  hooks.relay.guide(body,context(0,units));assert.equal(map.children.length,2,'rewind clears old transmission');
  fire('forward',.1,{unitId:4,amount:12});hooks.relay.guide(body,context(.1,units));
  assert.equal(map.children.length,2,'foreign node never becomes next hop');
  hooks.relay.guide(body,context(.2,[body,a]));assert.equal(map.children.length,1,'deleted nodes lose edges');
});

test('router activity and blinking binary trails freeze on match time and follow velocity',()=>{
  const {hooks,fire,Element}=setup();
  const el=new Element(),u={id:4};
  fire('join',1,{unitId:4});fire('arrive',1,{unitId:4});
  hooks['relay-node'].tick(el,u,{time:1.1});
  assert.ok(el.classes.has('relay-online'));assert.ok(el.classes.has('relay-receiving'));
  const frozen=JSON.stringify(el.attrs);
  hooks['relay-node'].tick(el,u,{time:1.1});assert.equal(JSON.stringify(el.attrs),frozen);
  hooks['relay-node'].tick(el,u,{time:2});assert.ok(!el.classes.has('relay-online')&&!el.classes.has('relay-receiving'));
  hooks['relay-shot'].tick(el,{vx:0,vy:520},{time:0,scale:1});assert.equal(el.attrs['--relay-angle'],`${-Math.PI/2}rad`);
  assert.ok(el.html.includes('class="relay-bits"><span>0</span><span>1</span>'));
  assert.ok(el.html.includes('class="relay-packet-tail"'));
  const first=el.attrs['--relay-bit-0'];
  hooks['relay-shot'].tick(el,{vx:0,vy:520},{time:.1,scale:1});
  assert.notEqual(el.attrs['--relay-bit-0'],first,'digits alternate brightness');
  const packetFrozen=JSON.stringify(el.attrs);
  hooks['relay-shot'].tick(el,{vx:0,vy:520},{time:.1,scale:1});
  assert.equal(JSON.stringify(el.attrs),packetFrozen,'pause freezes digits and tail');
  hooks['relay-shot'].tick(el,{radius:5,vx:-520,vy:0},{time:.2,scale:2});
  assert.equal(el.attrs['--relay-font'],'20px','binary size scales with the arena');
});
