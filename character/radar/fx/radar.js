window.lookFX = window.lookFX || {};
window.radarBeam = window.radarBeam || {};

window.lookFX.radar = {
  unmount(el) {
    if (!el) return;
    el.querySelector(":scope > .radar-beam")?.remove();
  },
  tick(el, u, ctx) {
    if (!el || !el.classList.contains("look-radar")) return;
    const st = window.radarBeam[u.id] || { dx: 0, dy: 1, len: 560 };
    let beam = el.querySelector(":scope > .radar-beam");
    if (!beam) {
      beam = document.createElement("i");
      beam.className = "radar-beam";
      for (const name of ["radar-glow", "radar-core", "radar-flare"]) {
        const layer = document.createElement("i");
        layer.className = name;
        beam.appendChild(layer);
      }
      el.appendChild(beam);
    }
    const scale = (ctx && ctx.scale) || 1;
    const dx = st.dx || 0;
    const dy = st.dy || 0;
    // 世界 +Y 朝上，屏幕 Y 朝下，所以对 dy 取负。
    const ang = Math.atan2(-dy, dx);
    beam.style.setProperty("--ang", `${ang}rad`);
    beam.style.setProperty("--len", `${Math.max(48, (st.len || 560) * scale)}px`);
  },
};
