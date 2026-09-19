window.lookFX = window.lookFX || {};

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
      for (const name of ["tracer-glow", "tracer-body", "tracer-core", "tracer-cap", "tracer-flare"]) {
        const layer = document.createElement("i");
        layer.className = name;
        art.appendChild(layer);
      }
      el.appendChild(art);
    }
    const ang = Math.atan2(-(u.vy || 0), u.vx || 1);
    const r = el.clientWidth || 12;
    art.style.setProperty("--tracer-ang", `${ang}rad`);
    art.style.setProperty("--tracer-len", `${Math.max(56, r * 6.4)}px`);
  },
};
