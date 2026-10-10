window.lookFX = window.lookFX || {};
window.radarVetBeam = window.radarVetBeam || {};
window.radarVetMark = window.radarVetMark || {};

window.lookFX["radar-vet"] = {
  unmount(el) {
    if (!el) return;
    el.querySelector(":scope > .radar-vet-beam")?.remove();
    el.querySelector(":scope > .radar-vet-reticle")?.remove();
    el.querySelector(":scope > .radar-vet-rings")?.remove();
  },
  tick(el, u, ctx) {
    if (!el || !el.classList.contains("look-radar-vet")) return;
    if (!el.querySelector(":scope > .radar-vet-reticle")) {
      const retic = document.createElement("i");
      retic.className = "radar-vet-reticle";
      el.appendChild(retic);
      const rings = document.createElement("i");
      rings.className = "radar-vet-rings";
      el.appendChild(rings);
    }
    const st = window.radarVetBeam[u.id] || { dx: 0, dy: 1, len: 280 };
    let beam = el.querySelector(":scope > .radar-vet-beam");
    if (!beam) {
      beam = document.createElement("i");
      beam.className = "radar-vet-beam";
      el.appendChild(beam);
    }
    const scale = (ctx && ctx.scale) || 1;
    const dx = st.dx || 0;
    const dy = st.dy || 0;
    const ang = Math.atan2(-dy, dx);
    beam.style.setProperty("--ang", `${ang}rad`);
    beam.style.setProperty("--len", `${Math.max(8, (st.len || 0) * scale)}px`);
  },
};

function tickShellMark(el, u, ctx, lookClass, markClass) {
  if (!el || !el.classList.contains(lookClass)) return;
  let mark = el.querySelector(`:scope > .${markClass}`);
  if (!mark) {
    mark = document.createElement("i");
    mark.className = markClass;
    el.appendChild(mark);
  }
  const st = window.radarVetMark[u.id] || { r: 4 };
  const scale = (ctx && ctx.scale) || 1;
  const r = Math.max(3, (st.r || 4) * scale * 2);
  mark.style.setProperty("--r", `${r}px`);
}

window.lookFX["radar-shell"] = {
  unmount(el) {
    if (!el) return;
    el.querySelector(":scope > .radar-shell-mark")?.remove();
  },
  tick(el, u, ctx) {
    tickShellMark(el, u, ctx, "look-radar-shell", "radar-shell-mark");
  },
};

window.lookFX["radar-shell-sm"] = {
  unmount(el) {
    if (!el) return;
    el.querySelector(":scope > .radar-shell-mark-sm")?.remove();
  },
  tick(el, u, ctx) {
    tickShellMark(el, u, ctx, "look-radar-shell-sm", "radar-shell-mark-sm");
  },
};
