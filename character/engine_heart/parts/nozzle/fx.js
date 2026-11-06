(function () {
  const parts=window.pumpParts=window.pumpParts||{};
  const p=parts[4]={
    name:"燃油喷嘴",
    color:"#d4af69",
    site:"炉窗下方燃烧室",
    mount:{"x": 2, "y": 8, "layer": "front"},
    motion:"chest",
    replaces:["fuel-valve"],
    svg:`<g data-part="4" stroke-linejoin="round">
  <g class="ph-motion-chest">
    <path d="M-8 1H8L10 6 7 10H-7L-10 6Z" fill="url(#ph-metal)" stroke="#080e11" stroke-width="1"/>
    <path d="M-7,7 -3,7 -3,1" fill="none" stroke="#120f09" stroke-width="4.6" stroke-linecap="square" stroke-linejoin="miter"/>
    <rect x="-8.5" y="5.5" width="7" height="3" rx="0.25" fill="url(#ph-pipeH)" stroke="none" stroke-width="0"/>
    <rect x="-4.5" y="-0.5" width="3" height="9" rx="0.25" fill="url(#ph-pipeV)" stroke="none" stroke-width="0"/>
    <rect x="-4.890000000000001" y="5.109999999999999" width="3.7800000000000002" height="3.7800000000000002" rx="0.65" fill="url(#ph-elbow)" stroke="#63481f" stroke-width="0.55"/>
    <path d="M-4.35 6.04h2.31" fill="none" stroke="#ffdf9b" stroke-width="0.45"/>
    <path d="M7,7 3,7 3,1" fill="none" stroke="#120f09" stroke-width="4.6" stroke-linecap="square" stroke-linejoin="miter"/>
    <rect x="1.5" y="5.5" width="7" height="3" rx="0.25" fill="url(#ph-pipeH)" stroke="none" stroke-width="0"/>
    <rect x="1.5" y="-0.5" width="3" height="9" rx="0.25" fill="url(#ph-pipeV)" stroke="none" stroke-width="0"/>
    <rect x="1.1099999999999999" y="5.109999999999999" width="3.7800000000000002" height="3.7800000000000002" rx="0.65" fill="url(#ph-elbow)" stroke="#63481f" stroke-width="0.55"/>
    <path d="M1.65 6.04h2.31" fill="none" stroke="#ffdf9b" stroke-width="0.45"/>
    <rect x="-5" y="-1" width="4" height="4" rx="0.5" fill="url(#ph-brass)" stroke="#080e11" stroke-width="0.7"/>
    <rect x="1" y="-1" width="4" height="4" rx="0.5" fill="url(#ph-brass)" stroke="#080e11" stroke-width="0.7"/>
    <path d="M-4 0h2M2 0h2" fill="none" stroke="#18130c" stroke-width="0.8"/>
    <g class="ph-injection-flame">
      <path d="M-3-1-4-6-2-4-1-7-1-1M3-1 2-6 4-4 5-7 5-1" fill="#f5c16e" stroke="none" stroke-width="0"/>
    </g>
    <g fill="#bec4b6" stroke="#11191c" stroke-width=".3">
      <circle cx="-7" cy="4" r="0.55"/>
      <path d="M-7.275 4h0.55" fill="none" stroke="#263332" stroke-width="0.25"/>
      <circle cx="7" cy="4" r="0.55"/>
      <path d="M6.725 4h0.55" fill="none" stroke="#263332" stroke-width="0.25"/>
    </g>
  </g>
</g>`
  };
})();
