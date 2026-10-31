(function () {
  const hooks = window.lookFX = window.lookFX || {};
  // 正交投影的明暗分界：上弦/下弦为直线，盈亏凸月为椭圆。
  function litPath(phase) {
    if (phase === 0) return "";
    if (phase === 4) return "M50 2a48 48 0 1 1 0 96 48 48 0 1 1 0-96Z";
    const right = phase < 4;
    const fraction = right ? phase : 8 - phase;
    const x = 48 * Math.abs(Math.cos(fraction * Math.PI / 4));
    const start = right ? "M50 2A48 48 0 0 1 50 98" : "M50 2A48 48 0 0 0 50 98";
    if (fraction === 2) return `${start}L50 2Z`;
    const sweep = (fraction === 1) === right ? 0 : 1;
    return `${start}A${x.toFixed(3)} 48 0 0 ${sweep} 50 2Z`;
  }
  hooks.moon = {
    tick(el, u, ctx) {
      let art = el.querySelector(":scope > .moon-art");
      if (!art) {
        art = document.createElementNS("http://www.w3.org/2000/svg", "svg");
        art.setAttribute("viewBox", "0 0 100 100");
        art.setAttribute("aria-hidden", "true");
        art.classList.add("moon-art");
        const id = `wolf-moon-${u.id}`;
        art.innerHTML = `<defs><radialGradient id="${id}-light" cx="32%" cy="27%" r="80%"><stop stop-color="#fff6da"/><stop offset=".57" stop-color="#e8d9a0"/><stop offset="1" stop-color="#a99768"/></radialGradient><clipPath id="${id}-phase"><path class="moon-lit-mask"/></clipPath></defs>
          <circle class="moon-dark" cx="50" cy="50" r="48"/>
          <g class="moon-surface" clip-path="url(#${id}-phase)"><circle cx="50" cy="50" r="48" fill="url(#${id}-light)"/>
          <path class="moon-maria" d="M22 24q16-12 28 3t22 12q8 14-6 22T36 64Q17 56 21 43Z"/>
          ${[[29,31,8],[39,62,11],[71,34,6],[66,71,9],[21,58,4],[52,20,3],[80,53,3],[49,82,4]].map(([x,y,r])=>`<circle class="moon-crater" cx="${x}" cy="${y}" r="${r}"/><path class="moon-crater-rim" d="M${x-r*.7} ${y-r*.45}a${r*.85} ${r*.85} 0 0 1 ${r*1.4} 0"/>`).join("")}</g>
          <path class="moon-terminator"/><circle class="moon-rim" cx="50" cy="50" r="48"/>
          <circle class="moon-phase-flash" cx="50" cy="50" r="47"/>`;
        el.appendChild(art);
      }
      const data = window.wolfVisual?.state(u.ownerId);
      const phase = data?.phase || 0;
      const path = litPath(phase);
      if (el.dataset.moon !== String(phase)) {
        art.querySelector(".moon-lit-mask").setAttribute("d", path);
        art.querySelector(".moon-terminator").setAttribute("d", phase === 0 || phase === 4 ? "" : path);
        el.dataset.moon = String(phase);
      }
      const time = ctx.time || 0;
      const age = time - (data?.phaseAt ?? -Infinity);
      el.style.setProperty("--moon-flash", (age >= 0 && age < 0.35 ? (1 - age / 0.35) * 0.7 : 0).toFixed(4));
      el.style.setProperty("--moon-bloom", (phase === 4 ? 0.5 + 0.12 * Math.sin(time * 3) : 0.17).toFixed(4));
    },
    unmount(el) {
      el.querySelector(":scope > .moon-art")?.remove();
      delete el.dataset.moon;
      for (const prop of ["--moon-flash", "--moon-bloom"]) el.style.removeProperty(prop);
    },
  };
})();
