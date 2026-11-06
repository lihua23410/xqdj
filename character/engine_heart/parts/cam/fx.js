(function () {
  const parts=window.pumpParts=window.pumpParts||{};
  const p=parts[3]={
    name:"配气凸轮",
    color:"#d4af69",
    site:"右肋正时脊",
    mount:{"x": 23, "y": 3, "layer": "front"},
    motion:"chest",
    replaces:["service-bypass"],
    svg:`<g data-part="3" stroke-linejoin="round">
  <g class="ph-motion-chest">
    <path d="M-5-12H3L7-8V9L3 13H-5Z" fill="url(#ph-metal)" stroke="#080e11" stroke-width="1"/>
    <rect x="-3" y="-10" width="7" height="21" rx="1" fill="#091317" stroke="#080e11" stroke-width="0.7"/>
    <path d="M-6,-11 0,-11 0,-8" fill="none" stroke="#120f09" stroke-width="4.0" stroke-linecap="square" stroke-linejoin="miter"/>
    <rect x="-7.2" y="-12.2" width="8.4" height="2.4" rx="0.25" fill="url(#ph-pipeH)" stroke="none" stroke-width="0"/>
    <rect x="-1.2" y="-12.2" width="2.4" height="5.4" rx="0.25" fill="url(#ph-pipeV)" stroke="none" stroke-width="0"/>
    <rect x="-1.512" y="-12.512" width="3.024" height="3.024" rx="0.65" fill="url(#ph-elbow)" stroke="#63481f" stroke-width="0.55"/>
    <path d="M-1.08 -11.768h1.848" fill="none" stroke="#ffdf9b" stroke-width="0.45"/>
    <path d="M0,8 0,11 -6,11" fill="none" stroke="#120f09" stroke-width="4.0" stroke-linecap="square" stroke-linejoin="miter"/>
    <rect x="-1.2" y="6.8" width="2.4" height="5.4" rx="0.25" fill="url(#ph-pipeV)" stroke="none" stroke-width="0"/>
    <rect x="-7.2" y="9.8" width="8.4" height="2.4" rx="0.25" fill="url(#ph-pipeH)" stroke="none" stroke-width="0"/>
    <rect x="-1.512" y="9.488" width="3.024" height="3.024" rx="0.65" fill="url(#ph-elbow)" stroke="#63481f" stroke-width="0.55"/>
    <path d="M-1.08 10.232h1.848" fill="none" stroke="#ffdf9b" stroke-width="0.45"/>
    <path d="M0-7V7" fill="none" stroke="#d8ba79" stroke-width="1.5"/>
    <g transform="translate(0 -4)">
      <g class="ph-cam-lobe">
        <path d="M-2 1Q-4-4 0-5Q5-3 3 1Z" fill="url(#ph-brass)" stroke="#5e4725" stroke-width="0.4"/>
        <circle cx="0" cy="0" r="0.8" fill="#263337" stroke="#0b1216" stroke-width="0.7"/>
      </g>
    </g>
    <g transform="translate(0 5)">
      <g class="ph-cam-lobe">
        <path d="M-2 1Q-4-4 0-5Q5-3 3 1Z" fill="url(#ph-brass)" stroke="#5e4725" stroke-width="0.4"/>
        <circle cx="0" cy="0" r="0.8" fill="#263337" stroke="#0b1216" stroke-width="0.7"/>
      </g>
    </g>
    <g fill="#bec4b6" stroke="#11191c" stroke-width=".3">
      <circle cx="-3" cy="-9" r="0.5"/>
      <path d="M-3.25 -9h0.5" fill="none" stroke="#263332" stroke-width="0.25"/>
      <circle cx="4" cy="-7" r="0.5"/>
      <path d="M3.75 -7h0.5" fill="none" stroke="#263332" stroke-width="0.25"/>
      <circle cx="-3" cy="9" r="0.5"/>
      <path d="M-3.25 9h0.5" fill="none" stroke="#263332" stroke-width="0.25"/>
      <circle cx="4" cy="7" r="0.5"/>
      <path d="M3.75 7h0.5" fill="none" stroke="#263332" stroke-width="0.25"/>
    </g>
  </g>
</g>`
  };
})();
