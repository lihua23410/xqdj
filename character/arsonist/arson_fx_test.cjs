const test = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const vm = require('node:vm');
const path = require('node:path');

function setup() {
  let spawned = 0;
  const root = { appendChild() { spawned++; } };
  const sandbox = {
    window: { lookFX: {}, shotFX: {} },
    document: {
      getElementById: () => root,
      createElement: () => ({ style: { setProperty() {} }, addEventListener() {} }),
    },
  };
  sandbox.window.arena = { registerShot(name, fn) {
    (sandbox.window.shotFX['/ball/纵火者'] ||= {})[name] = fn;
  } };
  sandbox.arena = sandbox.window.arena;
  vm.createContext(sandbox);
  const app = fs.readFileSync(path.join(__dirname, '../../internal/web/app.js'), 'utf8');
  const spawn = app.slice(app.indexOf('function spawnFx('), app.indexOf('function spawnGhostFrom('));
  const play = app.slice(app.indexOf('function playEffects('), app.indexOf('const lobby ='));
  vm.runInContext(`const fxRoot=document.getElementById('fx');const ENGINE_FX=new Set();const IMMEDIATE_AUDIO=new Set();const EXPORT=false;const state={time:0};function screenPos(x,y){return[x,y]}function kindColor(){return'#fff'}function lookOf(){return{base:'/ball/纵火者'}}
    ${spawn}
    ${play}`, sandbox);
  vm.runInContext(fs.readFileSync(path.join(__dirname, 'fx/arson.js'), 'utf8'), sandbox);
  return { sandbox, spawned: () => spawned };
}

test('persistent field FX do not fall through into uncollected transient nodes', () => {
  const { sandbox, spawned } = setup();
  sandbox.effects = Array.from({length: 2000}, (_, i) => ({name: ['arson-oil','arson-fire','arson-surface','arson-burning'][i % 4], kind: '纵火者油火', unitId: 7, x: i, y: 0, amount: 18}));
  for (let frame = 0; frame < 30; frame++) vm.runInContext('playEffects(effects,1,0,0)', sandbox);
  assert.equal(spawned(), 0, 'field geometry must be consumed by its own persistent renderer');
});

function renderer() {
  const { sandbox } = setup();
  let paints = 0, writes = 0, guides = new Map(), canvases = 0, flamePaints = 0, fills = 0;
  sandbox.document.createElement = tag => {
    assert.equal(tag, 'canvas'); canvases++;
    const canvas = {width:0,height:0,style:{},setAttribute(){},getContext(){return{
      clearRect(){flamePaints++},beginPath(){},moveTo(){},bezierCurveTo(){},closePath(){},fill(){fills++},fillRect(){},
      createLinearGradient(){return{addColorStop(){}}},createRadialGradient(){return{addColorStop(){}}},
      createImageData: (w,h)=>({data:new Uint8ClampedArray(w*h*4)}),
      putImageData(image){canvas.image=image;paints++;},
    }}};
    return canvas;
  };
  const ctx = {time:0,scale:1,cx:0,cy:0,effects:[],seenGuides:new Set(),ensureGuide(key){
    if(!guides.has(key))guides.set(key,{style:new Proxy({},{set(obj,key,val){writes++;obj[key]=val;return true}}),appendChild(child){this.canvas=child;}});
    return guides.get(key);
  }};
  const draw = (id, data, time = ctx.time, burns) => {
    ctx.time=time; ctx.effects = data ? [{unitId:id,name:'arson-surface',data}] : [];
    if(burns)ctx.effects.push({unitId:id,name:'arson-burning',data:burns});
    sandbox.window.lookFX['arson-field'].guide({id},ctx);
  };
  return {ctx,draw,guides,paints:()=>paints,writes:()=>writes,canvases:()=>canvases,flamePaints:()=>flamePaints,fills:()=>fills};
}
function manyCells(revision=1) {
  return {revision,step:3,side:32,chunks:Array.from({length:4},(_,i)=>({x:i,y:0,o:[0,32,32,32,64,32,96,32,128,32,160,32,192,32,224,32,256,32,288,32,320,32,352,32,384,32,416,32,448,32,480,32]}))};
}
test('thousands of cells use one guide and canvas; static surface is not repainted',()=>{
  const r=renderer();r.draw(7,manyCells());
  const paints=r.paints(), writes=r.writes();
  for(let i=0;i<30;i++)r.draw(7,manyCells(),i/60);
  assert.equal(r.guides.size,1);assert.equal(r.canvases(),1);
  assert.equal(r.paints(),paints);assert.equal(r.writes(),writes);
  r.ctx.scale=2;r.ctx.cx=100;r.draw(7,manyCells(),1);
  assert.equal(r.paints(),paints);assert.ok(r.writes()>writes,'resize must reposition the canvas');
  r.draw(7,manyCells(2),2);assert.equal(r.paints(),paints+1);
  r.draw(7,manyCells(2),0);assert.equal(r.paints(),paints+2,'rewind must discard old revision cache');
});
test('fire/oil boundary alone receives the bright front, burned area is transparent',()=>{
  const r=renderer();
  r.draw(7,{revision:1,step:3,side:32,chunks:[{x:0,y:0,f:[0,2],o:[2,1]}]});
  const canvas=r.guides.get('arson-surface-7').arsonSurface.canvas;
  const pixel=(x,y)=>Array.from(canvas.image.data.slice((y*canvas.width+x)*4,(y*canvas.width+x+1)*4));
  assert.deepEqual(pixel(2,1),[255,218,117,255]);
  assert.notDeepEqual(pixel(1,1),[255,218,117,255]);
  assert.equal(pixel(0,1)[3],0);
  r.draw(7,{revision:2,step:3,side:32,chunks:[]});
  assert.equal(canvas.width,0);assert.equal(canvas.height,0);
});
test('separate owners have independent canvases and late snapshots paint the full surface',()=>{
  const r=renderer();r.draw(7,manyCells());r.draw(8,manyCells());
  assert.equal(r.guides.size,2);assert.equal(r.canvases(),2);assert.equal(r.paints(),2);
  const late=renderer();late.draw(7,manyCells(25),100);
  assert.ok(late.guides.get('arson-surface-7').canvas.width>0);
  r.draw(7,null,-1);assert.equal(r.guides.get('arson-surface-7').canvas.width,0);
});

test('flame tongues share one bounded canvas and freeze with simulation time',()=>{
 const r=renderer();const surface=manyCells();
 for(const chunk of surface.chunks){chunk.f=chunk.o;delete chunk.o;}
 r.draw(7,surface,0);
 assert.equal(r.canvases(),2,'one floor canvas and one shared flame canvas');
 const initial=r.flamePaints(),fills=r.fills();
 for(let i=0;i<30;i++)r.draw(7,surface,0);
 assert.equal(r.flamePaints(),initial,'pause must freeze the flame animation');
 r.draw(7,surface,.1);
 assert.equal(r.flamePaints(),initial+1);
 assert.ok(r.fills()-fills<=144,'at most 72 two-layer tongues per field');
 assert.equal(r.paints(),1,'animation must not repaint the tar coverage');
 r.draw(7,{...surface,revision:2,chunks:[]},.2);
 assert.equal(r.guides.get('arson-surface-7').arsonSurface.flames.width,0);
});
test('body flames follow burn state and moving units; expired or dead targets lose their guide',()=>{
 const r=renderer();r.ctx.units=[{id:2,hp:100,radius:18,x:10,y:20}];
 const burn=[{id:2,until:1}];r.draw(7,manyCells(),0,burn);
 const key='arson-burning-2', el=r.guides.get(key);assert.ok(el);
 const left=el.style.left, count=r.flamePaints();
 r.ctx.units[0].x=60;r.draw(7,manyCells(),0,burn);
 assert.notEqual(el.style.left,left);assert.equal(r.flamePaints(),count,'moving while paused should only reposition');
 r.draw(8,manyCells(),0,burn);assert.equal(r.canvases(),3,'two sources must share the target flame canvas');
 r.ctx.seenGuides.clear();r.draw(7,manyCells(),1.1,[]);
 assert.ok(!r.ctx.seenGuides.has(key),'expiry clears the target effect');
 r.ctx.units[0].hp=0;r.ctx.seenGuides.clear();r.draw(7,manyCells(),.5,burn);
 assert.ok(!r.ctx.seenGuides.has(key),'dead targets never keep burning visuals');
});
test('glass bottle and wick mount once and pause at simulation time',()=>{
 const {sandbox}=setup();let nodes=0,writes=0;
 function element(){return{children:[],style:{setProperty(){writes++}},appendChild(child){this.children.push(child)},querySelector(){return this.children.find(c=>c.className==='arson-bottle')}};}
 sandbox.document.createElement=()=>{nodes++;return element()};
 const host=element(),u={id:9,vx:120,vy:40},ctx={time:.5};
 for(let i=0;i<100;i++)sandbox.window.lookFX['arson-fire'].tick(host,u,ctx);
 assert.equal(nodes,2,'one glass bottle and one wick');assert.equal(writes,2,'pause does not keep changing the bottle');
 ctx.time=.6;sandbox.window.lookFX['arson-fire'].tick(host,u,ctx);assert.equal(writes,4);
});
