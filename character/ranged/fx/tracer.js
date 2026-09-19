window.lookFX = window.lookFX || {};

const TRACER_SVG = `<svg viewBox="0 0 56 16" aria-hidden="true">
  <polygon points="4,8 11,3.6 22,5.2 34,2.2 54,8 34,13.8 22,10.8 11,12.4" fill="#161b24"/>
  <polygon points="12,8 22,5.6 34,3.4 50,8 34,12.6 22,10.4" fill="#2a3342"/>
  <polygon points="22,8 34,4.6 46,8 34,11.4" fill="#3a4556"/>
  <polyline points="11,3.6 22,5.2 34,2.2" fill="none" stroke="#ff6a18" stroke-width="0.9" stroke-linejoin="miter"/>
  <polyline points="11,12.4 22,10.8 34,13.8" fill="none" stroke="#ff6a18" stroke-width="0.9" stroke-linejoin="miter"/>
  <line x1="2" y1="8" x2="50" y2="8" stroke="#6af0ff" stroke-width="1.15" stroke-linecap="round"/>
</svg>`;

window.lookFX.tracer = {
  unmount(el) {
    el?.querySelector(":scope > .tracer-art")?.remove();
  },
  tick(el, u) {
    if (!el || !el.classList.contains("look-tracer")) return;
    let art = el.querySelector(":scope > .tracer-art");
    if (!art) {
      art = document.createElement("span");
      art.className = "tracer-art";
      for (const name of ["tracer-wake-glow", "tracer-wake", "tracer-heat", "tracer-core"]) {
        const layer = document.createElement("i");
        layer.className = name;
        art.appendChild(layer);
      }
      const body = document.createElement("span");
      body.className = "tracer-body";
      body.innerHTML = TRACER_SVG;
      art.appendChild(body);
      const tip = document.createElement("i");
      tip.className = "tracer-tip";
      art.appendChild(tip);
      el.appendChild(art);
    }
    const ang = Math.atan2(-(u.vy || 0), u.vx || 1);
    const r = el.clientWidth || 12;
    art.style.setProperty("--tracer-ang", `${ang}rad`);
    art.style.setProperty("--tracer-len", `${Math.max(72, r * 7.8)}px`);
  },
};
