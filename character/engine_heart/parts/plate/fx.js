(function () {
  const parts=window.pumpParts=window.pumpParts||{};
  const point=(r,a)=>`${(r*Math.cos(a)).toFixed(4)} ${(r*Math.sin(a)).toFixed(4)}`;
  const arc=(r,a,b)=>`M${point(r,a)}A${r} ${r} 0 0 1 ${point(r,b)}`;
  const panel=(inner,outer,a,b)=>`M${point(outer,a)}A${outer} ${outer} 0 0 1 ${point(outer,b)}L${point(inner,b)}A${inner} ${inner} 0 0 0 ${point(inner,a)}Z`;
  const rad=degrees=>degrees*Math.PI/180;
  const face=panel(25,31,rad(-54),rad(54));
  parts[5]={
    name:"护心夹板",
    color:"#d4af69",
    site:"前置回转装甲与承力导轨",
    description:"前置108°实体钢板沿承力导轨回转，黑铁甲面、黄铜包边与滚轮支座组成完整挡弹总成",
    mount:{x:0,y:0,layer:"overlay"},
    motion:"frame",
    replaces:[],
    svg:`<g data-part="5" stroke-linejoin="round">
      <g class="ph-motion-frame">
        <defs>
          <linearGradient id="ph-guard-steel" x1="0" y1="0" x2="1" y2=".7">
            <stop stop-color="#a1afaa"/><stop offset=".15" stop-color="#5a6b6d"/>
            <stop offset=".46" stop-color="#35474c"/><stop offset="1" stop-color="#17252b"/>
          </linearGradient>
        </defs>
        <g data-guard-space>
          <g class="ph-guard-race" fill="none">
            <circle r="23.7" stroke="#060e12" stroke-width="2.5"/>
            <circle r="23.7" stroke="url(#ph-brass)" stroke-width="1.15"/>
            <circle r="22.9" stroke="#82928a" stroke-width=".35"/>
          </g>
          <g class="ph-guard-saddles">
            ${[35,145,215,325].map(a=>`<g transform="rotate(${a})">
              <path d="M18-1.6H24.7V1.6H18Z" fill="url(#ph-metal)" stroke="#071015" stroke-width=".7"/>
              <rect x="20" y="-1.1" width="3.7" height="2.2" rx=".45" fill="url(#ph-brass)" stroke="#42351c" stroke-width=".4"/>
              <circle cx="21.5" r=".65" fill="#cbd0bd" stroke="#1a2526" stroke-width=".35"/>
            </g>`).join('')}
            <path d="M-17-11H-21V-5" fill="none" stroke="#0a1114" stroke-width="3.1"/>
            <path d="M-17-11H-21V-5" fill="none" stroke="url(#ph-brass)" stroke-width="1.7"/>
            <rect x="-22.5" y="-7" width="3" height="3.5" rx=".6" fill="url(#ph-elbow)" stroke="#3a321f" stroke-width=".5"/>
          </g>
          <g data-guard-rotor>
            ${[-38,38].map(a=>`<g transform="rotate(${a})">
              <path d="M22-1.6H27V1.6H22Z" fill="url(#ph-metal)" stroke="#081116" stroke-width=".8"/>
              <rect x="23" y="-.7" width="4.3" height="1.4" fill="url(#ph-brass)"/>
              <circle cx="23.7" r="1.65" fill="url(#ph-brass)" stroke="#0b161b" stroke-width=".65"/>
              <circle cx="23.7" r=".75" fill="#c8d0bd" stroke="#263235" stroke-width=".4"/>
            </g>`).join('')}
            <path d="${face}" fill="url(#ph-metal)" stroke="#050c10" stroke-width="1.2"/>
            ${[[-53,-18.6],[-17.4,17.4],[18.6,53]].map(([a,b])=>`<path d="${panel(25.8,30.2,rad(a),rad(b))}" fill="url(#ph-guard-steel)" stroke="#16252a" stroke-width=".55"/>`).join('')}
            <path d="${arc(30.65,rad(-53),rad(53))}" fill="none" stroke="url(#ph-brass)" stroke-width="1.05"/>
            <path d="${arc(25.35,rad(-53),rad(53))}" fill="none" stroke="url(#ph-brass)" stroke-width=".8"/>
            <path d="${arc(29.7,rad(-51),rad(51))}" fill="none" stroke="#d3dbce" stroke-width=".45"/>
            ${[-54,-18,18,54].map(a=>`<g transform="rotate(${a})"><path d="M25.1-.6H31V.6H25.1Z" fill="url(#ph-brass)" stroke="#1c292b" stroke-width=".4"/></g>`).join('')}
            ${[-46,-30,-10,10,30,46].map(a=>`<g transform="rotate(${a})">
              <circle cx="28" r=".9" fill="url(#ph-brass)" stroke="#101b1f" stroke-width=".45"/>
              <path d="M27.55 0H28.45" stroke="#e1dec2" stroke-width=".35"/>
            </g>`).join('')}
          </g>
        </g>
      </g>
    </g>`,
    update(node,d,ctx){
      // Bypass the machine's decorative tilt to match the real blocking arc.
      const guard=d.guards?.find(g=>g.key==="plate");
      const angle=guard?.angle??(ctx.time||0)*Math.PI/6;
      const scale=ctx.worldToArt||1;
      node.querySelector('[data-guard-space]').setAttribute('transform',`scale(${scale} ${-scale})`);
      node.querySelector('[data-guard-rotor]').setAttribute('transform',`rotate(${angle*180/Math.PI})`);
    }
  };
})();
