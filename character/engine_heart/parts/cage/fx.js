(function () {
  const parts=window.pumpParts=window.pumpParts||{};
  const p=parts[8]={
    name:"捕弹阀笼",
    color:"#d4af69",
    site:"左下腹压力腔",
    mount:{"x": -12, "y": 22, "layer": "front"},
    motion:"drive",
    replaces:["lower-blank"],
    svg:`<g data-part="8" stroke-linejoin="round">
  <g class="ph-motion-drive">
    <path d="M-6-8H5L7-4V7L3 10H-5L-7 5Z" fill="url(#ph-metal)" stroke="#080e11" stroke-width="1"/>
    <rect x="-4.5" y="-6" width="9" height="14" rx="2" fill="#081316" stroke="#080e11" stroke-width="0.7"/>
    <g class="ph-ammo">
      <circle cx="0" cy="1" r="3.5" fill="#e6b25e" stroke="#f9db93" stroke-width="0.5"/>
    </g>
    <rect x="-3.7" y="-5" width="1.4" height="12" rx="0.4" fill="url(#ph-pipeV)" stroke="#5f4923" stroke-width="0.25"/>
    <rect x="-0.7" y="-5" width="1.4" height="12" rx="0.4" fill="url(#ph-pipeV)" stroke="#5f4923" stroke-width="0.25"/>
    <rect x="2.3" y="-5" width="1.4" height="12" rx="0.4" fill="url(#ph-pipeV)" stroke="#5f4923" stroke-width="0.25"/>
    <rect x="-6" y="-8" width="12" height="3" rx="1" fill="url(#ph-brass)" stroke="#080e11" stroke-width="0.7"/>
    <rect x="-6" y="6" width="12" height="3" rx="1" fill="url(#ph-brass)" stroke="#080e11" stroke-width="0.7"/>
    <path d="M-5,-6 -8,-6 -8,-2" fill="none" stroke="#120f09" stroke-width="3.6" stroke-linecap="square" stroke-linejoin="miter"/>
    <rect x="-9.0" y="-7.0" width="5" height="2" rx="0.25" fill="url(#ph-pipeH)" stroke="none" stroke-width="0"/>
    <rect x="-9.0" y="-7.0" width="2" height="6" rx="0.25" fill="url(#ph-pipeV)" stroke="none" stroke-width="0"/>
    <rect x="-9.26" y="-7.26" width="2.52" height="2.52" rx="0.65" fill="url(#ph-elbow)" stroke="#63481f" stroke-width="0.55"/>
    <path d="M-8.9 -6.64h1.54" fill="none" stroke="#ffdf9b" stroke-width="0.45"/>
    <path d="M5,-6 8,-6 8,-2" fill="none" stroke="#120f09" stroke-width="3.6" stroke-linecap="square" stroke-linejoin="miter"/>
    <rect x="4.0" y="-7.0" width="5" height="2" rx="0.25" fill="url(#ph-pipeH)" stroke="none" stroke-width="0"/>
    <rect x="7.0" y="-7.0" width="2" height="6" rx="0.25" fill="url(#ph-pipeV)" stroke="none" stroke-width="0"/>
    <rect x="6.74" y="-7.26" width="2.52" height="2.52" rx="0.65" fill="url(#ph-elbow)" stroke="#63481f" stroke-width="0.55"/>
    <path d="M7.1 -6.64h1.54" fill="none" stroke="#ffdf9b" stroke-width="0.45"/>
    <g fill="#bec4b6" stroke="#11191c" stroke-width=".3">
      <circle cx="-4" cy="-6.5" r="0.45"/>
      <path d="M-4.225 -6.5h0.45" fill="none" stroke="#263332" stroke-width="0.25"/>
      <circle cx="4" cy="-6.5" r="0.45"/>
      <path d="M3.775 -6.5h0.45" fill="none" stroke="#263332" stroke-width="0.25"/>
      <circle cx="-4" cy="7.5" r="0.45"/>
      <path d="M-4.225 7.5h0.45" fill="none" stroke="#263332" stroke-width="0.25"/>
      <circle cx="4" cy="7.5" r="0.45"/>
      <path d="M3.775 7.5h0.45" fill="none" stroke="#263332" stroke-width="0.25"/>
    </g>
  </g>
</g>`
  };
})();
