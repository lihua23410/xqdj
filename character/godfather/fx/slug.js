window.lookFX = window.lookFX || {};

window.lookFX.slug = {
  unmount(el) {
    el?.querySelector(":scope > .slug-art")?.remove();
  },
  tick(el, u) {
    if (!el || !el.classList.contains("look-slug")) return;
    let art = el.querySelector(":scope > .slug-art");
    if (!art) {
      art = document.createElement("span");
      art.className = "slug-art";
      for (const name of ["slug-glow", "slug-core", "slug-hot", "slug-ticks", "slug-tip"]) {
        const layer = document.createElement("i");
        layer.className = name;
        art.appendChild(layer);
      }
      el.appendChild(art);
    }
    const ang = Math.atan2(-(u.vy || 0), u.vx || 1);
    const r = el.clientWidth || 10;
    art.style.setProperty("--slug-ang", `${ang}rad`);
    art.style.setProperty("--slug-len", `${Math.max(132, r * 16)}px`);
  },
};
