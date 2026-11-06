(function () {
  const parts=window.pumpParts=window.pumpParts||{};
  const p=parts[0]={
    name:"增压活塞",
    color:"#d4af69",
    site:"左肋主泵缸",
    mount:{"x": -24, "y": 1, "layer": "front"},
    motion:"chest",
    replaces:[],
    svg:`<g data-part="0" stroke-linejoin="round">
  <g class="ph-motion-chest">
    <path d="M-10-13H1L5-9V12L0 17H-10Z" fill="url(#ph-metal)" stroke="#080e11" stroke-width="1"/>
    <rect x="-8" y="-11" width="7" height="24" rx="2" fill="#0a1519" stroke="#080e11" stroke-width="0.7"/>
    <rect x="-7" y="-9" width="5" height="19" rx="1" fill="url(#ph-pipeV)" stroke="#080e11" stroke-width="0.7"/>
    <g class="ph-piston">
      <rect x="-8" y="-8" width="7" height="6" rx="1" fill="url(#ph-metal)" stroke="#84928a" stroke-width="0.4"/>
      <rect x="-5" y="-3" width="1.8" height="16" rx="0.3" fill="url(#ph-brass)" stroke="#080e11" stroke-width="0.7"/>
    </g>
    <rect x="-11" y="-13" width="13" height="3" rx="1" fill="url(#ph-brass)" stroke="#080e11" stroke-width="0.7"/>
    <rect x="-11" y="12" width="13" height="3" rx="1" fill="url(#ph-brass)" stroke="#080e11" stroke-width="0.7"/>
    <path d="M0,-10 6,-10 6,-6" fill="none" stroke="#120f09" stroke-width="4.6" stroke-linecap="square" stroke-linejoin="miter"/>
    <rect x="-1.5" y="-11.5" width="9" height="3" rx="0.25" fill="url(#ph-pipeH)" stroke="none" stroke-width="0"/>
    <rect x="4.5" y="-11.5" width="3" height="7" rx="0.25" fill="url(#ph-pipeV)" stroke="none" stroke-width="0"/>
    <rect x="4.109999999999999" y="-11.89" width="3.7800000000000002" height="3.7800000000000002" rx="0.65" fill="url(#ph-elbow)" stroke="#63481f" stroke-width="0.55"/>
    <path d="M4.65 -10.96h2.31" fill="none" stroke="#ffdf9b" stroke-width="0.45"/>
    <path d="M0,12 6,12" fill="none" stroke="#120f09" stroke-width="4.6" stroke-linecap="square" stroke-linejoin="miter"/>
    <rect x="-1.5" y="10.5" width="9" height="3" rx="0.25" fill="url(#ph-pipeH)" stroke="none" stroke-width="0"/>
    <g fill="#bec4b6" stroke="#11191c" stroke-width=".3">
      <circle cx="-9" cy="-11.5" r="0.5"/>
      <path d="M-9.25 -11.5h0.5" fill="none" stroke="#263332" stroke-width="0.25"/>
      <circle cx="0" cy="-11.5" r="0.5"/>
      <path d="M-0.25 -11.5h0.5" fill="none" stroke="#263332" stroke-width="0.25"/>
      <circle cx="-9" cy="13.5" r="0.5"/>
      <path d="M-9.25 13.5h0.5" fill="none" stroke="#263332" stroke-width="0.25"/>
      <circle cx="0" cy="13.5" r="0.5"/>
      <path d="M-0.25 13.5h0.5" fill="none" stroke="#263332" stroke-width="0.25"/>
    </g>
  </g>
</g>`
  };
})();
