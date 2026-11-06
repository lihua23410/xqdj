(function () {
  const parts=window.pumpParts=window.pumpParts||{};
  const p=parts[1]={
    name:"分流歧管",
    color:"#d4af69",
    site:"右上肩压力桥",
    mount:{"x": 15, "y": -22, "layer": "front"},
    motion:"crown",
    replaces:["crown-bypass"],
    svg:`<g data-part="1" stroke-linejoin="round">
  <g class="ph-motion-crown">
    <path d="M-10-7 9-7 14-2V13H6V9H-10Z" fill="url(#ph-metal)" stroke="#080e11" stroke-width="1"/>
    <path d="M-7,9 -7,-4 11,-4 11,13 7,13" fill="none" stroke="#120f09" stroke-width="5.9" stroke-linecap="square" stroke-linejoin="miter"/>
    <rect x="-9.15" y="-6.15" width="4.3" height="17.3" rx="0.25" fill="url(#ph-pipeV)" stroke="none" stroke-width="0"/>
    <rect x="-9.15" y="-6.15" width="22.3" height="4.3" rx="0.25" fill="url(#ph-pipeH)" stroke="none" stroke-width="0"/>
    <rect x="8.85" y="-6.15" width="4.3" height="21.3" rx="0.25" fill="url(#ph-pipeV)" stroke="none" stroke-width="0"/>
    <rect x="4.85" y="10.85" width="8.3" height="4.3" rx="0.25" fill="url(#ph-pipeH)" stroke="none" stroke-width="0"/>
    <rect x="-9.709" y="-6.709" width="5.418" height="5.418" rx="0.65" fill="url(#ph-elbow)" stroke="#63481f" stroke-width="0.55"/>
    <path d="M-8.935 -5.376h3.311" fill="none" stroke="#ffdf9b" stroke-width="0.45"/>
    <rect x="8.291" y="-6.709" width="5.418" height="5.418" rx="0.65" fill="url(#ph-elbow)" stroke="#63481f" stroke-width="0.55"/>
    <path d="M9.065 -5.376h3.311" fill="none" stroke="#ffdf9b" stroke-width="0.45"/>
    <rect x="8.291" y="10.291" width="5.418" height="5.418" rx="0.65" fill="url(#ph-elbow)" stroke="#63481f" stroke-width="0.55"/>
    <path d="M9.065 11.624h3.311" fill="none" stroke="#ffdf9b" stroke-width="0.45"/>
    <path d="M-2,9 -2,1 6,1 6,9" fill="none" stroke="#120f09" stroke-width="5.1" stroke-linecap="square" stroke-linejoin="miter"/>
    <rect x="-3.75" y="-0.75" width="3.5" height="11.5" rx="0.25" fill="url(#ph-pipeV)" stroke="none" stroke-width="0"/>
    <rect x="-3.75" y="-0.75" width="11.5" height="3.5" rx="0.25" fill="url(#ph-pipeH)" stroke="none" stroke-width="0"/>
    <rect x="4.25" y="-0.75" width="3.5" height="11.5" rx="0.25" fill="url(#ph-pipeV)" stroke="none" stroke-width="0"/>
    <rect x="-4.205" y="-1.205" width="4.41" height="4.41" rx="0.65" fill="url(#ph-elbow)" stroke="#63481f" stroke-width="0.55"/>
    <path d="M-3.575 -0.12h2.695" fill="none" stroke="#ffdf9b" stroke-width="0.45"/>
    <rect x="3.795" y="-1.205" width="4.41" height="4.41" rx="0.65" fill="url(#ph-elbow)" stroke="#63481f" stroke-width="0.55"/>
    <path d="M4.425 -0.12h2.695" fill="none" stroke="#ffdf9b" stroke-width="0.45"/>
    <path d="M-7,9 -10,9 -10,11" fill="none" stroke="#120f09" stroke-width="4.6" stroke-linecap="square" stroke-linejoin="miter"/>
    <rect x="-11.5" y="7.5" width="6" height="3" rx="0.25" fill="url(#ph-pipeH)" stroke="none" stroke-width="0"/>
    <rect x="-11.5" y="7.5" width="3" height="5" rx="0.25" fill="url(#ph-pipeV)" stroke="none" stroke-width="0"/>
    <rect x="-11.89" y="7.109999999999999" width="3.7800000000000002" height="3.7800000000000002" rx="0.65" fill="url(#ph-elbow)" stroke="#63481f" stroke-width="0.55"/>
    <path d="M-11.35 8.04h2.31" fill="none" stroke="#ffdf9b" stroke-width="0.45"/>
    <rect x="-9.881" y="4.7" width="5.7620000000000005" height="2.6" rx="0.7" fill="url(#ph-brass)" stroke="#5c421c" stroke-width="0.5"/>
    <rect x="8.119" y="3.7" width="5.7620000000000005" height="2.6" rx="0.7" fill="url(#ph-brass)" stroke="#5c421c" stroke-width="0.5"/>
    <rect x="-8" y="-6" width="20" height="2" rx="0.5" fill="url(#ph-brass)" stroke="#080e11" stroke-width="0.7"/>
    <g fill="#bec4b6" stroke="#11191c" stroke-width=".3">
      <circle cx="-6" cy="-5" r="0.45"/>
      <path d="M-6.225 -5h0.45" fill="none" stroke="#263332" stroke-width="0.25"/>
      <circle cx="10" cy="-5" r="0.45"/>
      <path d="M9.775 -5h0.45" fill="none" stroke="#263332" stroke-width="0.25"/>
    </g>
  </g>
</g>`
  };
})();
