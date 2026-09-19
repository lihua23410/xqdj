window.lookFX = window.lookFX || {};

window.lookFX["melee-arc"] = {
  unmount(el) {
    el?.querySelector(":scope > .blade-glow")?.remove();
    el?.querySelector(":scope > .blade-edge")?.remove();
    el?.querySelector(":scope > .blade-core")?.remove();
  },
  tick(el) {
    if (!el || !el.classList.contains("look-melee-arc")) return;
    for (const name of ["blade-glow", "blade-edge", "blade-core"]) {
      if (!el.querySelector(`:scope > .${name}`)) {
        const layer = document.createElement("i");
        layer.className = name;
        el.appendChild(layer);
      }
    }
  },
};

arena.registerShot("dash", (fx, ctx) => {
  const { x, y, kind } = ctx;
  arena.spawnFx("fx-melee-flare", x, y, kind);
  arena.spawnFx("fx-melee-ring", x, y, kind);
  arena.spawnFx("fx-melee-ring", x, y, kind, { "--delay": "0.07s" });
});
