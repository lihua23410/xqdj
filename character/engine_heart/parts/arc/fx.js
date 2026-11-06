(function(){
 const parts=window.pumpParts=window.pumpParts||{};
 parts[14]={
  name:"接触电弧线圈",
  color:"#d4af69",
  site:"顶部蒸汽发电线圈",
  description:"近身两次搏动储电，真正碰撞才放电麻痹，向接触目标附近传导",
  attack:true,
  mount:{"x": -8, "y": -34, "layer": "front"},
  motion:"crown",
  replaces:["chimney-cap"],
  svg:`<g data-part="14" stroke-linejoin="round">
  <g class="ph-motion-crown">
    <rect x="-9" y="1" width="18" height="6" rx="1" fill="url(#ph-metal)" stroke="#0b1216" stroke-width=".7"/>
    <path d="M0 5V12" fill="none" stroke="#281f0f" stroke-width="5.4"/>
    <path d="M0 5V12" fill="none" stroke="url(#ph-brass)" stroke-width="4"/>
    <rect x="-7.5" y="-9" width="4" height="11" rx="1" fill="#0b151b" stroke="#0b1216" stroke-width=".7"/>
    <rect x="-9.0" y="-8" width="7" height="1.6" rx="0.6" fill="url(#ph-brass)" stroke="#0b1216" stroke-width=".7"/>
    <rect x="-9.0" y="-5.5" width="7" height="1.6" rx="0.6" fill="url(#ph-brass)" stroke="#0b1216" stroke-width=".7"/>
    <rect x="-9.0" y="-3" width="7" height="1.6" rx="0.6" fill="url(#ph-brass)" stroke="#0b1216" stroke-width=".7"/>
    <rect x="-9.0" y="-0.5" width="7" height="1.6" rx="0.6" fill="url(#ph-brass)" stroke="#0b1216" stroke-width=".7"/>
    <circle cx="-5.5" cy="-11" r="3" fill="url(#ph-brass)" stroke="#0a1115" stroke-width="0.6"/>
    <rect x="3.5" y="-9" width="4" height="11" rx="1" fill="#0b151b" stroke="#0b1216" stroke-width=".7"/>
    <rect x="2.0" y="-8" width="7" height="1.6" rx="0.6" fill="url(#ph-brass)" stroke="#0b1216" stroke-width=".7"/>
    <rect x="2.0" y="-5.5" width="7" height="1.6" rx="0.6" fill="url(#ph-brass)" stroke="#0b1216" stroke-width=".7"/>
    <rect x="2.0" y="-3" width="7" height="1.6" rx="0.6" fill="url(#ph-brass)" stroke="#0b1216" stroke-width=".7"/>
    <rect x="2.0" y="-0.5" width="7" height="1.6" rx="0.6" fill="url(#ph-brass)" stroke="#0b1216" stroke-width=".7"/>
    <circle cx="5.5" cy="-11" r="3" fill="url(#ph-brass)" stroke="#0a1115" stroke-width="0.6"/>
    <rect x="-11" y="2" width="22" height="3" rx="1" fill="url(#ph-brass)" stroke="#0b1216" stroke-width=".7"/>
    <path d="M-3-11-1-13 1-9 3-11" fill="none" stroke="#f4e6a9" stroke-width="0.8"/>
    <circle cx="-8" cy="3.5" r="0.65" fill="#c7c6ac" stroke="#0a1115" stroke-width="0.6"/>
    <circle cx="8" cy="3.5" r="0.65" fill="#c7c6ac" stroke="#0a1115" stroke-width="0.6"/>
  </g>
</g>`
 };
 const effects=window.pumpAttackFX=window.pumpAttackFX||{};
 effects["heart-arc-charge"]={channel:"arc-charge",duration:.12,draw(c,f,age){
   c.globalAlpha=1-age/.12;
   for(let i=0;i<2;i++){c.beginPath();c.arc(f.x-30,f.y+4+i*7,2.3,0,Math.PI*2);c.fillStyle=(f.amount||0)>i*.5?"#a6f2ef":"#3f6874";c.fill();}
   if(f.amount>=1){c.beginPath();c.moveTo(f.x-21,f.y+40);c.lineTo(f.x-16,f.y+44);c.lineTo(f.x-13,f.y+39);c.lineTo(f.x-8,f.y+43);c.strokeStyle="#b3f4f0";c.lineWidth=1.3;c.stroke();}
 }};
 effects["heart-arc-off"]={channel:"arc-charge",duration:0};
 effects["heart-arc"]={duration:.3,draw(c,f,age){
   c.globalAlpha=(1-age/.3)*(f.amount||1);
   const dx=f.vx-f.x,dy=f.vy-f.y,len=Math.hypot(dx,dy)||1,nx=-dy/len,ny=dx/len;
   c.beginPath();c.moveTo(f.x,f.y);for(let i=1;i<9;i++){const t=i/9,bend=Math.sin(i*2.7+age*100)*8;c.lineTo(f.x+dx*t+nx*bend,f.y+dy*t+ny*bend);}c.lineTo(f.vx,f.vy);
   c.strokeStyle="#5dbac95c";c.lineWidth=11;c.stroke();c.strokeStyle="#c7ffff";c.lineWidth=2;c.stroke();
   c.beginPath();c.arc(f.vx,f.vy,5+age*40,0,Math.PI*2);c.strokeStyle="#82e4ec";c.lineWidth=2;c.stroke();
 }};
 for(const name of ["heart-arc-charge","heart-arc-off","heart-arc"])arena.registerShot(name,()=>{});
})();
