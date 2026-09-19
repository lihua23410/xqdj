window.lookFX = window.lookFX || {};
window.wolfRaging = window.wolfRaging || {};

window.lookFX["wolf-arc"] = {
  unmount(el) {
    el?.querySelector(":scope > .blade-glow")?.remove();
    el?.querySelector(":scope > .blade-edge")?.remove();
    el?.querySelector(":scope > .blade-core")?.remove();
  },
  tick(el) {
    if (!el || !el.classList.contains("look-wolf-arc")) return;
    for (const name of ["blade-glow", "blade-edge", "blade-core"]) {
      if (!el.querySelector(`:scope > .${name}`)) {
        const layer = document.createElement("i");
        layer.className = name;
        el.appendChild(layer);
      }
    }
  },
};

window.lookFX.wolf = {
  unmount(el) {
    el?.classList.remove("wolf-rage");
  },
  tick(el, u) {
    if (!el || !el.classList.contains("look-wolf")) return;
    el.classList.toggle("wolf-rage", !!window.wolfRaging[u.id]);
  },
};
