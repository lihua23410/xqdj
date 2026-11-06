(function () {
  const parts=window.pumpParts=window.pumpParts||{};
  const p=parts[2]={
    name:"惯性飞轮",
    color:"#d4af69",
    site:"右下传动轴",
    mount:{"x": 13, "y": 24, "layer": "front"},
    motion:"drive",
    replaces:["drive-cover"],
    svg:`<g data-part="2" stroke-linejoin="round">
  <g class="ph-motion-drive">
    <path d="M-10-8H7L10-4V6L5 11H-7L-11 6Z" fill="url(#ph-metal)" stroke="#080e11" stroke-width="1"/>
    <path d="M-8,-8 -8,-11 0,-11" fill="none" stroke="#120f09" stroke-width="4.0" stroke-linecap="square" stroke-linejoin="miter"/>
    <rect x="-9.2" y="-12.2" width="2.4" height="5.4" rx="0.25" fill="url(#ph-pipeV)" stroke="none" stroke-width="0"/>
    <rect x="-9.2" y="-12.2" width="10.4" height="2.4" rx="0.25" fill="url(#ph-pipeH)" stroke="none" stroke-width="0"/>
    <rect x="-9.512" y="-12.512" width="3.024" height="3.024" rx="0.65" fill="url(#ph-elbow)" stroke="#63481f" stroke-width="0.55"/>
    <path d="M-9.08 -11.768h1.848" fill="none" stroke="#ffdf9b" stroke-width="0.45"/>
    <circle cx="0" cy="0" r="9" fill="url(#ph-metal)" stroke="#050c11" stroke-width="1.1"/>
    <circle cx="0" cy="0" r="7.4" fill="#0b151a" stroke="url(#ph-brass)" stroke-width="1.4"/>
    <g class="ph-wheel">
      <circle cx="0" cy="0" r="5.9" fill="none" stroke="#8c713e" stroke-width="1.6"/>
      <path d="M0-6.2V6.2" fill="none" stroke="url(#ph-brass)" stroke-width="1.4" transform="rotate(0)"/>
      <path d="M0-6.2V6.2" fill="none" stroke="url(#ph-brass)" stroke-width="1.4" transform="rotate(45)"/>
      <path d="M0-6.2V6.2" fill="none" stroke="url(#ph-brass)" stroke-width="1.4" transform="rotate(90)"/>
      <path d="M0-6.2V6.2" fill="none" stroke="url(#ph-brass)" stroke-width="1.4" transform="rotate(135)"/>
      <circle cx="0" cy="0" r="2.3" fill="url(#ph-metal)" stroke="#d8b568" stroke-width="0.6"/>
      <circle cx="0" cy="0" r="0.7" fill="#b9b798" stroke="#0b1216" stroke-width="0.7"/>
    </g>
    <g fill="#bec4b6" stroke="#11191c" stroke-width=".3">
      <circle cx="-7.8" cy="-7.8" r="0.5"/>
      <path d="M-8.05 -7.8h0.5" fill="none" stroke="#263332" stroke-width="0.25"/>
      <circle cx="7.8" cy="-6.5" r="0.5"/>
      <path d="M7.55 -6.5h0.5" fill="none" stroke="#263332" stroke-width="0.25"/>
      <circle cx="7" cy="7.3" r="0.5"/>
      <path d="M6.75 7.3h0.5" fill="none" stroke="#263332" stroke-width="0.25"/>
      <circle cx="-7.5" cy="7" r="0.5"/>
      <path d="M-7.75 7h0.5" fill="none" stroke="#263332" stroke-width="0.25"/>
    </g>
  </g>
</g>`
  };
})();
