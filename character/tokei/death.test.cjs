const test=require('node:test');
const assert=require('node:assert/strict');
const vm=require('node:vm');
const fs=require('node:fs');
const box={window:{}};
vm.runInNewContext(fs.readFileSync(__dirname+'/fx/tokei.js','utf8'),box);
const angles=box.window.tokeiDeathAngles;

test('death hands overclock in opposing directions for three simulated seconds',()=>{
  const a=angles(10,13,10),b=angles(10,13,11),c=angles(10,13,12.5);
  assert.equal(a.progress,0);
  assert.ok(b.hour>720&&b.minute< -720);
  assert.ok(c.hour>b.hour&&c.minute<b.minute);
  assert.equal(b.fade,0);
  assert.ok(Math.abs(angles(10,13,13).fade-1)<1e-12);
  assert.deepEqual(angles(10,13,11),b); // 暂停、导出重复绘制不会继续转针。
  assert.ok(angles(10,13,11.016).hour!==b.hour);
});

test('private FX drives the skin without extending unit snapshots',()=>{
  const shots=new Map();
  const scene={
    window:{},
    document:{currentScript:null},
    Audio:class {load(){}},
    arena:{
      SOUND_MODE:{IMMUNE:2},
      voice:()=>({}),
      registerShot:(name,fn)=>shots.set(name,fn),
    },
  };
  vm.createContext(scene);
  vm.runInContext(fs.readFileSync(__dirname+'/fx/tokei.js','utf8'),scene);
  vm.runInContext(fs.readFileSync(__dirname+'/fx/shot.js','utf8'),scene);
  const classList=()=>{
    const values=new Set();
    return {
      toggle:(name,on)=>on?values.add(name):values.delete(name),
      remove:(name)=>values.delete(name),
      contains:(name)=>values.has(name),
    };
  };
  const hour={style:{}},minute={style:{}};
  const face={
    dataset:{built:'1'},classList:classList(),remove(){},
    querySelector:(selector)=>selector==='.hand.hour'?hour:minute,
  };
  const props=new Map();
  const el={
    id:'u-1',classList:classList(),querySelector:()=>face,
    style:{setProperty:(k,v)=>props.set(k,v),removeProperty:(k)=>props.delete(k)},
  };
  const fx={unitId:1,x:10,y:13};
  shots.get('overclock')(fx);
  const skin=scene.window.lookFX.tokei;
  skin.tick(el,{id:1},{time:11});
  assert.ok(el.classList.contains('tokei-dying'));
  const first=[hour.style.transform,minute.style.transform];
  shots.get('overclock')(fx); // 补发同一窗口不会重开演出。
  skin.tick(el,{id:1},{time:11});
  assert.deepEqual([hour.style.transform,minute.style.transform],first);
  skin.tick(el,{id:1},{time:12});
  assert.notEqual(hour.style.transform,first[0]);
  assert.notEqual(minute.style.transform,first[1]);
  assert.equal(Number(props.get('--tokei-death')),2/3);
  skin.unmount(el);
  assert.equal(scene.window.tokeiSt[1],undefined);
  assert.equal(props.size,0);
  shots.get('clock')({unitId:1,amount:0,x:0,y:8});
  skin.tick(el,{id:1},{time:0});
  assert.equal(el.classList.contains('tokei-dying'),false);
});
