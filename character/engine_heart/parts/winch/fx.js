(function () {
  const parts=window.pumpParts=window.pumpParts||{};
  const p=parts[10]={
    name:"回收绞盘",
    color:"#d4af69",
    site:"右下肋回收机构",
    mount:{"x": 26, "y": 20, "layer": "front"},
    motion:"drive",
    replaces:[],
    svg:`<g data-part="10" stroke-linejoin="round">
  <g class="ph-motion-drive">
    <path d="M-12-8H3L7-2V7L3 10H-6L-12 3Z" fill="url(#ph-metal)" stroke="#080e11" stroke-width="1"/>
    <path d="M-12,-5 -8,-5 -8,6 -5,6" fill="none" stroke="#120f09" stroke-width="4.1" stroke-linecap="square" stroke-linejoin="miter"/>
    <rect x="-13.25" y="-6.25" width="6.5" height="2.5" rx="0.25" fill="url(#ph-pipeH)" stroke="none" stroke-width="0"/>
    <rect x="-9.25" y="-6.25" width="2.5" height="13.5" rx="0.25" fill="url(#ph-pipeV)" stroke="none" stroke-width="0"/>
    <rect x="-9.25" y="4.75" width="5.5" height="2.5" rx="0.25" fill="url(#ph-pipeH)" stroke="none" stroke-width="0"/>
    <rect x="-9.575" y="-6.575" width="3.15" height="3.15" rx="0.65" fill="url(#ph-elbow)" stroke="#63481f" stroke-width="0.55"/>
    <path d="M-9.125 -5.8h1.925" fill="none" stroke="#ffdf9b" stroke-width="0.45"/>
    <rect x="-9.575" y="4.425" width="3.15" height="3.15" rx="0.65" fill="url(#ph-elbow)" stroke="#63481f" stroke-width="0.55"/>
    <path d="M-9.125 5.2h1.925" fill="none" stroke="#ffdf9b" stroke-width="0.45"/>
    <rect x="-4" y="-4" width="7" height="10" rx="0.5" fill="#071319" stroke="#080e11" stroke-width="0.7"/>
    <path d="M-3 -3H3" fill="none" stroke="#c8b787" stroke-width="0.85"/>
    <path d="M-3 -1H3" fill="none" stroke="#c8b787" stroke-width="0.85"/>
    <path d="M-3 1H3" fill="none" stroke="#c8b787" stroke-width="0.85"/>
    <path d="M-3 3H3" fill="none" stroke="#c8b787" stroke-width="0.85"/>
    <path d="M-3 5H3" fill="none" stroke="#c8b787" stroke-width="0.85"/>
    <rect x="-6" y="-5" width="2.6" height="12" rx="0.5" fill="url(#ph-metal)" stroke="#080e11" stroke-width="0.7"/>
    <rect x="3" y="-5" width="2.6" height="12" rx="0.5" fill="url(#ph-metal)" stroke="#080e11" stroke-width="0.7"/>
    <path d="M0-4V-8H6V-5Q6-2 3-2" fill="none" stroke="#a7b5a8" stroke-width="1.4"/>
    <g transform="translate(-5 2)">
      <g class="ph-valve">
        <circle cx="0" cy="0" r="2" fill="#101a1d" stroke="url(#ph-brass)" stroke-width="1.1"/>
        <path d="M-2 0H2M0 -2V2" fill="none" stroke="#e8bd6e" stroke-width="0.65"/>
        <circle cx="0" cy="0" r="0.8" fill="#c49a50" stroke="#0b1216" stroke-width="0.7"/>
      </g>
    </g>
    <g fill="#bec4b6" stroke="#11191c" stroke-width=".3">
      <circle cx="-10" cy="-6" r="0.45"/>
      <path d="M-10.225 -6h0.45" fill="none" stroke="#263332" stroke-width="0.25"/>
      <circle cx="-4.8" cy="-3.5" r="0.45"/>
      <path d="M-5.025 -3.5h0.45" fill="none" stroke="#263332" stroke-width="0.25"/>
      <circle cx="4.5" cy="5.5" r="0.45"/>
      <path d="M4.275 5.5h0.45" fill="none" stroke="#263332" stroke-width="0.25"/>
    </g>
  </g>
</g>`
  };
})();
