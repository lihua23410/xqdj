(function(){
 const parts=window.pumpParts=window.pumpParts||{};
 parts[12]={
  name:"破阵撞角",
  color:"#d4af69",
  site:"下胸承力撞架",
  description:"每三次收缩锁向敌人冲撞，每名敌人每次冲撞只受击一次",
  attack:true,
  mount:{"x": 0, "y": 4, "layer": "front"},
  motion:"drive",
  replaces:[],
  svg:`<g data-part="12" stroke-linejoin="round">
  <g class="ph-motion-drive">
    <path d="M-26 0-19 3-16 9H16L19 3 26 0 30 5 27 14 16 17H-16L-27 14-30 5Z" fill="url(#ph-metal)" stroke="#060d11" stroke-width="1.2"/>
    <path d="M-27 5-24 11-16 13H16L24 11 27 5" fill="none" stroke="#a8ae95" stroke-width="1"/>
    <path d="M-19 5V11H19V5" fill="none" stroke="#281f0f" stroke-width="4.199999999999999"/>
    <path d="M-19 5V11H19V5" fill="none" stroke="url(#ph-brass)" stroke-width="2.8"/>
    <path d="M-30 4-33 10-26 8M30 4 33 10 26 8" fill="url(#ph-brass)" stroke="#3c301a" stroke-width="0.7"/>
    <rect x="-10" y="9" width="20" height="6" rx="1" fill="url(#ph-metal)" stroke="#0b1216" stroke-width=".7"/>
    <circle cx="-24" cy="7" r="0.65" fill="#c7c6ac" stroke="#0a1115" stroke-width="0.6"/>
    <circle cx="-16" cy="14" r="0.65" fill="#c7c6ac" stroke="#0a1115" stroke-width="0.6"/>
    <circle cx="0" cy="12" r="0.65" fill="#c7c6ac" stroke="#0a1115" stroke-width="0.6"/>
    <circle cx="16" cy="14" r="0.65" fill="#c7c6ac" stroke="#0a1115" stroke-width="0.6"/>
    <circle cx="24" cy="7" r="0.65" fill="#c7c6ac" stroke="#0a1115" stroke-width="0.6"/>
  </g>
</g>`
 };
 const effects=window.pumpAttackFX=window.pumpAttackFX||{};
 effects["heart-ram"]={duration:0.3,draw(c,f,age){c.globalAlpha=(1-age/0.3)*(f.amount||1);const a=Math.atan2(f.vy-f.y,f.vx-f.x),t=age/.3;c.translate(f.x+(f.vx-f.x)*t,f.y+(f.vy-f.y)*t);c.rotate(a);for(let i=0;i<3;i++){c.beginPath();c.moveTo(-i*11-8,-14);c.lineTo(-i*11+4,0);c.lineTo(-i*11-8,14);c.strokeStyle=i===0?"#e9c984":"#bda26c88";c.lineWidth=3-i*.7;c.stroke();}}};
 arena.registerShot("heart-ram",()=>{});
})();
