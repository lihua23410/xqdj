const test=require('node:test');
const assert=require('node:assert/strict');
const fs=require('node:fs');
const vm=require('node:vm');

function setup() {
  const events=[];
  function element() {
    return {children:[],dataset:{},appendChild(el){this.children.push(el);el.parent=this;},remove(){if(this.parent)this.parent.children=this.parent.children.filter(x=>x!==this);},animate(){return {pause(){},cancel(){events.push('cancel animation');},currentTime:0};}};
  }
  const host=element();
  const sandbox={window:{},document:{createElement:element}};
  vm.createContext(sandbox);
  vm.runInContext(fs.readFileSync(__dirname+'/presentation.js','utf8'),sandbox);
  const bus={addEventListener(){events.push('listen');},removeEventListener(){events.push('unlisten');}};
  const context={host,mode:'export',scale:1,createVoice(){events.push('audio');return {dispose(){events.push('stop audio');}}}};
  return {player:sandbox.window.presentationPlayer,host,context,events,bus,sandbox};
}

test('different payloads and pack namespaces use the same scheduler',()=>{
  const {player,host,context,events,bus}=setup();
  let mounts=0;
  player.register('/ball/one','demo',{
    mount(ctx,data){mounts++;ctx.own(()=>events.push('dispose text'));ctx.listen(bus,'click',()=>{});ctx.playAudio('muted');ctx.animate(ctx.root,[],{});return {text:data.message};},
    render(instance,t,ctx){events.push(`${instance.text}:${t}:${ctx.scale}`);},
    destroy(){events.push('destroy text');},
  });
  player.register('/ball/two','demo',{
    mount(ctx,data){ctx.playAudio('enabled');return data.values;},
    render(values,t){events.push(`values:${values.join(',')}:${t}`);},
    destroy(){events.push('destroy values');},
  });
  const list=[{id:'1',pack:'one',type:'demo',duration:1,data:{message:'hello'},audio:false},{id:'2',pack:'two',type:'demo',duration:2,data:{values:[2,4]},audio:true}];
  assert.equal(player.draw(list,0,context),true);
  player.draw(list,.5,{...context,scale:2});
  assert.equal(mounts,1);
  assert.equal(events.includes('audio'),false);
  player.draw(list,1.25,context);
  assert.equal(host.children.length,1);
  assert.ok(events.includes('hello:0.5:2'));
  assert.ok(events.includes('values:2,4:0.25'));
  assert.ok(events.includes('unlisten')&&events.includes('cancel animation')&&events.includes('dispose text'));
  assert.equal(player.draw(list,3,context),false);
  assert.equal(host.children.length,0);
  assert.equal(events.filter(x=>x==='stop audio').length,1);
  player.clear();
  assert.equal(events.filter(x=>x==='destroy values').length,1);
});

test('static preview is silent and remounts when playback actually starts',()=>{
  const {player,context,events}=setup();
  player.register('/ball/test','text',{mount(ctx){events.push('mount');ctx.playAudio('sound');},render(){},destroy(){events.push('destroy');}});
  const list=[{id:'1',pack:'test',type:'text',duration:1,audio:true,data:'a string payload'}];
  player.draw(list,0,{...context,preview:true});
  assert.equal(events.includes('audio'),false);
  player.draw(list,0,context);
  assert.equal(events.filter(x=>x==='mount').length,2);
  assert.equal(events.filter(x=>x==='audio').length,1);
  player.clear();
  assert.ok(events.includes('stop audio'));
});

test('renderer failures still dispose resources and remove the layer',()=>{
  const {player,context,host,events}=setup();
  player.register('/ball/test','broken',{mount(ctx){ctx.own(()=>events.push('disposed'));return {};},render(){throw Error('broken');},destroy(){events.push('destroy');}});
  assert.throws(()=>player.draw([{id:'x',pack:'test',type:'broken',duration:1}],0,context),/broken/);
  assert.equal(host.children.length,0);
  assert.deepEqual(events,['destroy','disposed']);
  assert.throws(()=>player.draw([{id:'y',pack:'test',type:'unknown',duration:1}],0,context),/找不到/);
});

test('live adapter waits for packs, plays sequentially and sends one last-ID receipt',()=>{
 const {player,host,context,events,sandbox}=setup();
 let now=0;
 const sent=[];
 Object.assign(sandbox,{EXPORT:false,hex:host,performance:{now:()=>now},state:{presentations:[]},packLoads:new Set(['loading']),send:(m)=>sent.push(m)});
 for (const name of ['screenPos','kindColor','paintKind','applyLookFX','runLookFX','unmountLookFX']) sandbox[name]=()=>{};
 sandbox.window.arena={playAudio:context.createVoice};
 const source=fs.readFileSync(__dirname+'/app.js','utf8');
 vm.runInContext(source.slice(source.indexOf('let openingStart = null;'),source.indexOf('function renderArena()')),sandbox);
 player.register('/ball/test','text',{mount(ctx,data){events.push('mount '+data.text);return data;},render(data,t){events.push(data.text+':'+t);},destroy(data){events.push('destroy '+data.text);}});
 const descriptor=(id,text)=>({id,pack:'test',type:'text',duration:1,data:{text}});
 sandbox.state.presentations=[descriptor('1','first'),descriptor('2','second')];
 const draw=()=>vm.runInContext('playOpening(1,0,0)',sandbox);
 assert.equal(draw(),true);assert.equal(events.length,0);
 now=10000;sandbox.packLoads.clear();draw();
 assert.ok(events.includes('first:0'));
 now=11000;draw();assert.ok(events.includes('second:0'));
 now=12000;assert.equal(draw(),false);draw();
 assert.equal(sent.length,1);assert.equal(sent[0].type,'presentation-end');assert.equal(sent[0].id,'2');
 sandbox.state.presentations=[descriptor('3','next')];draw();
 assert.ok(events.includes('next:0'));
 sandbox.state.presentations=[];draw();
 assert.ok(events.includes('destroy next'));assert.equal(host.children.length,0);assert.equal(sent.length,1);
});
