(function(){
 const parts=window.pumpParts=window.pumpParts||{};
 parts[13]={
  name:"蒸汽泄压缸",
  color:"#d4af69",
  site:"背部双联压力缸",
  description:"收缩在原位置留下短暂蒸汽云，分段灼伤并减速",
  attack:true,
  mount:{"x": 0, "y": -7, "layer": "rear"},
  motion:"crown",
  replaces:[],
  svg:`<g data-part="13" stroke-linejoin="round">
  <g class="ph-motion-crown">
    <rect x="-30" y="-14" width="10" height="28" rx="3" fill="url(#ph-metal)" stroke="#0b1216" stroke-width=".7"/>
    <rect x="-31" y="-13" width="12" height="3" rx="1" fill="url(#ph-brass)" stroke="#0b1216" stroke-width=".7"/>
    <rect x="-31" y="10" width="12" height="3" rx="1" fill="url(#ph-brass)" stroke="#0b1216" stroke-width=".7"/>
    <rect x="-27" y="-9" width="4" height="17" rx="1" fill="#081316" stroke="#0b1216" stroke-width=".7"/>
    <path d="M-25-8V6" fill="none" stroke="#d7bc79" stroke-width="1.8"/>
    <circle cx="-28" cy="-11" r="0.65" fill="#c7c6ac" stroke="#0a1115" stroke-width="0.6"/>
    <circle cx="-22" cy="-11" r="0.65" fill="#c7c6ac" stroke="#0a1115" stroke-width="0.6"/>
    <circle cx="-28" cy="12" r="0.65" fill="#c7c6ac" stroke="#0a1115" stroke-width="0.6"/>
    <circle cx="-22" cy="12" r="0.65" fill="#c7c6ac" stroke="#0a1115" stroke-width="0.6"/>
    <rect x="20" y="-14" width="10" height="28" rx="3" fill="url(#ph-metal)" stroke="#0b1216" stroke-width=".7"/>
    <rect x="19" y="-13" width="12" height="3" rx="1" fill="url(#ph-brass)" stroke="#0b1216" stroke-width=".7"/>
    <rect x="19" y="10" width="12" height="3" rx="1" fill="url(#ph-brass)" stroke="#0b1216" stroke-width=".7"/>
    <rect x="23" y="-9" width="4" height="17" rx="1" fill="#081316" stroke="#0b1216" stroke-width=".7"/>
    <path d="M25-8V6" fill="none" stroke="#d7bc79" stroke-width="1.8"/>
    <circle cx="22" cy="-11" r="0.65" fill="#c7c6ac" stroke="#0a1115" stroke-width="0.6"/>
    <circle cx="28" cy="-11" r="0.65" fill="#c7c6ac" stroke="#0a1115" stroke-width="0.6"/>
    <circle cx="22" cy="12" r="0.65" fill="#c7c6ac" stroke="#0a1115" stroke-width="0.6"/>
    <circle cx="28" cy="12" r="0.65" fill="#c7c6ac" stroke="#0a1115" stroke-width="0.6"/>
    <path d="M-25-13V-19H25V-13" fill="none" stroke="#281f0f" stroke-width="5.4"/>
    <path d="M-25-13V-19H25V-13" fill="none" stroke="url(#ph-brass)" stroke-width="4"/>
    <path d="M-25 12V17H-19M25 12V17H19" fill="none" stroke="#281f0f" stroke-width="4.4"/>
    <path d="M-25 12V17H-19M25 12V17H19" fill="none" stroke="url(#ph-brass)" stroke-width="3"/>
    <rect x="-5" y="-21" width="10" height="5" rx="1" fill="url(#ph-metal)" stroke="#0b1216" stroke-width=".7"/>
    <path d="M-3-19H3" fill="none" stroke="#d9c488" stroke-width="1.2"/>
  </g>
</g>`
 };
 const effects=window.pumpAttackFX=window.pumpAttackFX||{};
 effects["heart-steam"]={duration:0.42,draw(c,f,age){c.globalAlpha=(1-age/0.42)*(f.amount||1);const t=age/.42;for(let i=0;i<12;i++){const a=i*Math.PI/6+.2,r=(16+53*t)*(i%2?.8:1);c.beginPath();c.arc(f.x+Math.cos(a)*r,f.y+Math.sin(a)*r,17+15*t,0,Math.PI*2);c.fillStyle=i%3?"#c3cfba":"#e3cf9b";c.globalAlpha=(1-t)*.2;c.fill();}c.globalAlpha=(1-t)*.25;c.beginPath();c.arc(f.x,f.y,82*t,0,Math.PI*2);c.strokeStyle="#e1d4ae";c.lineWidth=2;c.stroke();}};
 arena.registerShot("heart-steam",()=>{});
})();
