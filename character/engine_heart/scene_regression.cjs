// Standalone renderer regression: node character/engine_heart/scene_regression.cjs
const fs=require('node:fs'),vm=require('node:vm'),assert=require('node:assert/strict');
let arcs=[];
const c={setTransform(){},clearRect(){},beginPath(){},lineTo(){},moveTo(){},closePath(){},stroke(){},fill(){},fillRect(){},strokeRect(){},setLineDash(){},save(){},restore(){},translate(){},rotate(){},arc(x,y,r){arcs.push({x,y,r});}};
const canvas={width:0,height:0,getContext:()=>c};
const el={style:{},appendChild(){}};
const sandbox={window:{},document:{currentScript:null,createElement:()=>canvas},arena:{registerShot(){}},Math,Set,Map};
vm.runInNewContext(fs.readFileSync(__dirname+'/fx/heart.js','utf8'),sandbox);
const data={parts:[],stack:[],phase:.6,radius:108,winch:-1,guards:[]};
sandbox.window.lookFX['pump-floor'].guide({ownerId:1},{time:10.1,scale:1,cx:300,cy:300,units:[{id:1,x:100,y:0}],effects:[{name:'heart-state',unitId:1,data},{name:'heart-beat',unitId:1,x:0,y:0,vx:108,amount:1}],ensureGuide:()=>el,seenGuides:new Set()});
assert(arcs.some(a=>Math.abs(a.x)<1&&Math.abs(a.y)<1&&a.r>=18),'attack pulse was not rendered at its fixed world origin');
arcs=[];
sandbox.window.lookFX['pump-floor'].guide({ownerId:1},{time:10.2,scale:1,cx:300,cy:300,units:[{id:1,x:160,y:0}],effects:[],ensureGuide:()=>el,seenGuides:new Set()});
assert(arcs.some(a=>Math.abs(a.x)<1&&Math.abs(a.y)<1&&a.r>=18),'attack pulse followed the moving body');
console.log('attack world origin: PASS');
const pose=sandbox.window.lookFX['pump-heart'].mechanicalPose;
assert(Object.values(pose({phase:.2})).every(y=>y===0),'relaxed machine did not return to its stops');
const top=pose({phase:.39});
assert(top.crown>0&&top.chest===0&&top.drive===0&&top.keel===0,'contraction did not start at the crown');
const chest=pose({phase:.43});
assert(chest.chest>0&&chest.drive===0&&chest.keel===0,'contraction skipped the chest stage');
const drive=pose({phase:.47});
assert(drive.drive<0&&drive.keel===0,'lower transmission did not close before the keel');
assert(pose({phase:.53}).keel<0,'keel did not retract at the end of the wave');
assert(Object.values(pose({phase:.9})).every(y=>y===0),'machine failed to release pressure');
assert(pose({phase:.7,echo:true}).crown>pose({phase:.7}).crown,'double valve did not produce a second mechanical wave');
assert(Object.values(pose({phase:.53,echo:true,suspended:true})).every(y=>y===0),'stunned machine continued contracting');
console.log('top-to-bottom mechanical contraction: PASS');

let painted=[];
sandbox.window.pumpAttackFX={
 'test-aim':{channel:'test-beam',duration:.2,draw(){painted.push('aim');}},
 'test-burn':{channel:'test-beam',duration:.2,draw(){painted.push('burn');}},
 'test-off':{channel:'test-beam',duration:0},
};
function frame(time,name){sandbox.window.lookFX['pump-floor'].guide({ownerId:1},{time,scale:1,cx:300,cy:300,units:[{id:1,x:0,y:0}],effects:[{name:'heart-state',unitId:1,data},...(name?[{name,unitId:1,x:0,y:0,vx:100,vy:0}]:[])],ensureGuide:()=>el,seenGuides:new Set()});}
painted=[];frame(11,'test-aim');assert.deepEqual(painted,['aim']);
painted=[];frame(11.02,'test-burn');assert.deepEqual(painted,['burn'],'burning beam left its old warning/geometry behind');
painted=[];frame(11.03,'test-off');assert.deepEqual(painted,[],'interruption did not clear an ongoing beam');
painted=[];frame(11.04);assert.deepEqual(painted,[],'cancelled beam reappeared');
console.log('continuous FX replacement and interruption: PASS');
