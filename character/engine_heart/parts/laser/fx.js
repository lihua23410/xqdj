(function(){
 const parts=window.pumpParts=window.pumpParts||{};
 parts[11]={
  name:"蓄压光棱镜",
  color:"#d4af69",
  site:"炉窗聚焦光路",
  description:"远距蓄射：预警后锁死方向，持续灼烧射线上的敌人；横移可躲",
  attack:true,
  mount:{"x": 2, "y": 2, "layer": "front"},
  motion:"chest",
  replaces:[],
  svg:`<g data-part="11" stroke-linejoin="round">
  <g class="ph-motion-chest">
    <circle cx="0" cy="0" r="11.3" fill="none" stroke="#0b1216" stroke-width="3.8"/>
    <circle cx="0" cy="0" r="11.1" fill="none" stroke="url(#ph-brass)" stroke-width="2.2"/>
    <g transform="rotate(0)">
      <path d="M-3-11-2-7H2L3-11Z" fill="url(#ph-metal)" stroke="#9b8a53" stroke-width="0.5"/>
      <circle cx="0" cy="-11" r="0.7" fill="#e2d4ab" stroke="#0a1115" stroke-width="0.6"/>
    </g>
    <g transform="rotate(60)">
      <path d="M-3-11-2-7H2L3-11Z" fill="url(#ph-metal)" stroke="#9b8a53" stroke-width="0.5"/>
      <circle cx="0" cy="-11" r="0.7" fill="#e2d4ab" stroke="#0a1115" stroke-width="0.6"/>
    </g>
    <g transform="rotate(120)">
      <path d="M-3-11-2-7H2L3-11Z" fill="url(#ph-metal)" stroke="#9b8a53" stroke-width="0.5"/>
      <circle cx="0" cy="-11" r="0.7" fill="#e2d4ab" stroke="#0a1115" stroke-width="0.6"/>
    </g>
    <g transform="rotate(180)">
      <path d="M-3-11-2-7H2L3-11Z" fill="url(#ph-metal)" stroke="#9b8a53" stroke-width="0.5"/>
      <circle cx="0" cy="-11" r="0.7" fill="#e2d4ab" stroke="#0a1115" stroke-width="0.6"/>
    </g>
    <g transform="rotate(240)">
      <path d="M-3-11-2-7H2L3-11Z" fill="url(#ph-metal)" stroke="#9b8a53" stroke-width="0.5"/>
      <circle cx="0" cy="-11" r="0.7" fill="#e2d4ab" stroke="#0a1115" stroke-width="0.6"/>
    </g>
    <g transform="rotate(300)">
      <path d="M-3-11-2-7H2L3-11Z" fill="url(#ph-metal)" stroke="#9b8a53" stroke-width="0.5"/>
      <circle cx="0" cy="-11" r="0.7" fill="#e2d4ab" stroke="#0a1115" stroke-width="0.6"/>
    </g>
    <path d="M-10-4H-14V6H-9" fill="none" stroke="#281f0f" stroke-width="3.6"/>
    <path d="M-10-4H-14V6H-9" fill="none" stroke="url(#ph-brass)" stroke-width="2.2"/>
    <path d="M10-4H14V6H9" fill="none" stroke="#281f0f" stroke-width="3.6"/>
    <path d="M10-4H14V6H9" fill="none" stroke="url(#ph-brass)" stroke-width="2.2"/>
    <rect x="-3" y="-15" width="6" height="5" rx="1" fill="url(#ph-brass)" stroke="#0b1216" stroke-width=".7"/>
    <circle cx="0" cy="-13" r="1.3" fill="#e5dbb6" stroke="#0a1115" stroke-width="0.6"/>
  </g>
</g>`
 };
 const effects=window.pumpAttackFX=window.pumpAttackFX||{};
 effects["heart-laser-aim"]={channel:"laser",duration:.12,draw(c,f,age){
   c.globalAlpha=1-age/.12;c.beginPath();c.moveTo(f.x,f.y);c.lineTo(f.vx,f.vy);
   c.setLineDash([5,5]);c.strokeStyle="#ea976b";c.lineWidth=1;c.stroke();c.setLineDash([]);
   c.beginPath();c.arc(f.x,f.y,20+(1-(f.amount||0))*13,0,Math.PI*2);c.strokeStyle="#f6bf85";c.lineWidth=1.5;c.stroke();
   c.beginPath();c.moveTo(f.vx-5,f.vy);c.lineTo(f.vx+5,f.vy);c.moveTo(f.vx,f.vy-5);c.lineTo(f.vx,f.vy+5);c.stroke();
 }};
 effects["heart-laser"]={channel:"laser",duration:.12,draw(c,f,age){
   c.globalAlpha=1-age/.12;
   for(const [width,color] of [[19,"#cf571f44"],[9,"#f09e4b"],[3,"#fff1c7"]]){c.beginPath();c.moveTo(f.x,f.y);c.lineTo(f.vx,f.vy);c.lineWidth=width;c.strokeStyle=color;c.stroke();}
   c.beginPath();c.arc(f.vx,f.vy,6,0,Math.PI*2);c.fillStyle="#ffdc8b";c.fill();
 }};
 effects["heart-laser-off"]={channel:"laser",duration:0};
 for(const name of ["heart-laser-aim","heart-laser","heart-laser-off"])arena.registerShot(name,()=>{});
})();
