(function () {
  const parts=window.pumpParts=window.pumpParts||{};
  const p=parts[7]={
    name:"偏流叶轮",
    color:"#d4af69",
    site:"左上肩增压进气",
    mount:{"x": -21, "y": -21, "layer": "front"},
    motion:"crown",
    replaces:[],
    svg:`<g data-part="7" stroke-linejoin="round">
  <g class="ph-motion-crown">
    <path d="M-8-7H5L13-5V7L4 10H-6L-10 4Z" fill="url(#ph-metal)" stroke="#080e11" stroke-width="1"/>
    <path d="M6,0 13,0 13,-3" fill="none" stroke="#120f09" stroke-width="5.300000000000001" stroke-linecap="square" stroke-linejoin="miter"/>
    <rect x="4.15" y="-1.85" width="10.7" height="3.7" rx="0.25" fill="url(#ph-pipeH)" stroke="none" stroke-width="0"/>
    <rect x="11.15" y="-4.85" width="3.7" height="6.7" rx="0.25" fill="url(#ph-pipeV)" stroke="none" stroke-width="0"/>
    <rect x="10.669" y="-2.331" width="4.662" height="4.662" rx="0.65" fill="url(#ph-elbow)" stroke="#63481f" stroke-width="0.55"/>
    <path d="M11.335 -1.184h2.849" fill="none" stroke="#ffdf9b" stroke-width="0.45"/>
    <circle cx="0" cy="0" r="8.5" fill="url(#ph-metal)" stroke="#070f14" stroke-width="1"/>
    <circle cx="0" cy="0" r="6.8" fill="#081418" stroke="url(#ph-brass)" stroke-width="1.2"/>
    <g class="ph-impeller">
      <path d="M1 0Q-3-4 1-6L4-5Q1-3 3 1Z" fill="url(#ph-brass)" stroke="#755724" stroke-width="0.25" transform="rotate(0)"/>
      <path d="M1 0Q-3-4 1-6L4-5Q1-3 3 1Z" fill="url(#ph-brass)" stroke="#755724" stroke-width="0.25" transform="rotate(60)"/>
      <path d="M1 0Q-3-4 1-6L4-5Q1-3 3 1Z" fill="url(#ph-brass)" stroke="#755724" stroke-width="0.25" transform="rotate(120)"/>
      <path d="M1 0Q-3-4 1-6L4-5Q1-3 3 1Z" fill="url(#ph-brass)" stroke="#755724" stroke-width="0.25" transform="rotate(180)"/>
      <path d="M1 0Q-3-4 1-6L4-5Q1-3 3 1Z" fill="url(#ph-brass)" stroke="#755724" stroke-width="0.25" transform="rotate(240)"/>
      <path d="M1 0Q-3-4 1-6L4-5Q1-3 3 1Z" fill="url(#ph-brass)" stroke="#755724" stroke-width="0.25" transform="rotate(300)"/>
      <circle cx="0" cy="0" r="2" fill="url(#ph-metal)" stroke="#c6bb8b" stroke-width="0.5"/>
    </g>
    <g fill="#bec4b6" stroke="#11191c" stroke-width=".3">
      <circle cx="-6" cy="-6" r="0.45"/>
      <path d="M-6.225 -6h0.45" fill="none" stroke="#263332" stroke-width="0.25"/>
      <circle cx="6" cy="-6" r="0.45"/>
      <path d="M5.775 -6h0.45" fill="none" stroke="#263332" stroke-width="0.25"/>
      <circle cx="-6" cy="6" r="0.45"/>
      <path d="M-6.225 6h0.45" fill="none" stroke="#263332" stroke-width="0.25"/>
      <circle cx="6" cy="6" r="0.45"/>
      <path d="M5.775 6h0.45" fill="none" stroke="#263332" stroke-width="0.25"/>
    </g>
  </g>
</g>`
  };
})();
