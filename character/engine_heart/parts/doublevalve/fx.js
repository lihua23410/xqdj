(function () {
  const parts=window.pumpParts=window.pumpParts||{};
  const p=parts[6]={
    name:"双联阀组",
    color:"#d4af69",
    site:"炉膛上方双联进气",
    mount:{"x": 3, "y": -11, "layer": "front"},
    motion:"chest",
    replaces:[],
    svg:`<g data-part="6" stroke-linejoin="round">
  <g class="ph-motion-chest">
    <path d="M-9-4H8L10 0 7 6H-8L-10 1Z" fill="url(#ph-metal)" stroke="#080e11" stroke-width="1"/>
    <path d="M-7,5 -7,-3 6,-3 6,5" fill="none" stroke="#120f09" stroke-width="4.6" stroke-linecap="square" stroke-linejoin="miter"/>
    <rect x="-8.5" y="-4.5" width="3" height="11" rx="0.25" fill="url(#ph-pipeV)" stroke="none" stroke-width="0"/>
    <rect x="-8.5" y="-4.5" width="16" height="3" rx="0.25" fill="url(#ph-pipeH)" stroke="none" stroke-width="0"/>
    <rect x="4.5" y="-4.5" width="3" height="11" rx="0.25" fill="url(#ph-pipeV)" stroke="none" stroke-width="0"/>
    <rect x="-8.89" y="-4.890000000000001" width="3.7800000000000002" height="3.7800000000000002" rx="0.65" fill="url(#ph-elbow)" stroke="#63481f" stroke-width="0.55"/>
    <path d="M-8.35 -3.96h2.31" fill="none" stroke="#ffdf9b" stroke-width="0.45"/>
    <rect x="4.109999999999999" y="-4.890000000000001" width="3.7800000000000002" height="3.7800000000000002" rx="0.65" fill="url(#ph-elbow)" stroke="#63481f" stroke-width="0.55"/>
    <path d="M4.65 -3.96h2.31" fill="none" stroke="#ffdf9b" stroke-width="0.45"/>
    <path d="M-2,5 -2,1 3,1 3,5" fill="none" stroke="#120f09" stroke-width="3.6" stroke-linecap="square" stroke-linejoin="miter"/>
    <rect x="-3.0" y="0.0" width="2" height="6" rx="0.25" fill="url(#ph-pipeV)" stroke="none" stroke-width="0"/>
    <rect x="-3.0" y="0.0" width="7" height="2" rx="0.25" fill="url(#ph-pipeH)" stroke="none" stroke-width="0"/>
    <rect x="2.0" y="0.0" width="2" height="6" rx="0.25" fill="url(#ph-pipeV)" stroke="none" stroke-width="0"/>
    <rect x="-3.26" y="-0.26" width="2.52" height="2.52" rx="0.65" fill="url(#ph-elbow)" stroke="#63481f" stroke-width="0.55"/>
    <path d="M-2.9 0.36h1.54" fill="none" stroke="#ffdf9b" stroke-width="0.45"/>
    <rect x="1.74" y="-0.26" width="2.52" height="2.52" rx="0.65" fill="url(#ph-elbow)" stroke="#63481f" stroke-width="0.55"/>
    <path d="M2.1 0.36h1.54" fill="none" stroke="#ffdf9b" stroke-width="0.45"/>
    <g transform="translate(-6 -2)">
      <g class="ph-valve">
        <circle cx="0" cy="0" r="2.8" fill="#101a1d" stroke="url(#ph-brass)" stroke-width="1.1"/>
        <path d="M-2.8 0H2.8M0 -2.8V2.8" fill="none" stroke="#e8bd6e" stroke-width="0.65"/>
        <circle cx="0" cy="0" r="0.8" fill="#c49a50" stroke="#0b1216" stroke-width="0.7"/>
      </g>
    </g>
    <g transform="translate(5 -2)">
      <g class="ph-valve">
        <circle cx="0" cy="0" r="2.8" fill="#101a1d" stroke="url(#ph-brass)" stroke-width="1.1"/>
        <path d="M-2.8 0H2.8M0 -2.8V2.8" fill="none" stroke="#e8bd6e" stroke-width="0.65"/>
        <circle cx="0" cy="0" r="0.8" fill="#c49a50" stroke="#0b1216" stroke-width="0.7"/>
      </g>
    </g>
    <g fill="#bec4b6" stroke="#11191c" stroke-width=".3">
      <circle cx="-8" cy="3" r="0.45"/>
      <path d="M-8.225 3h0.45" fill="none" stroke="#263332" stroke-width="0.25"/>
      <circle cx="8" cy="3" r="0.45"/>
      <path d="M7.775 3h0.45" fill="none" stroke="#263332" stroke-width="0.25"/>
    </g>
  </g>
</g>`
  };
})();
