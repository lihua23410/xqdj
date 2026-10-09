window.lookFX = window.lookFX || {};

window.lookFX.shard = {
  unmount(el) {
    el?.querySelector(":scope > .shard-art")?.remove();
  },
  tick(el, u) {
    if (!el || !el.classList.contains("look-shard")) return;
    let art = el.querySelector(":scope > .shard-art");
    if (!art) {
      art = document.createElement("span");
      art.className = "shard-art";
      for (const name of ["shard-glow", "shard-body", "shard-core"]) {
        const layer = document.createElement("i");
        layer.className = name;
        art.appendChild(layer);
      }
      el.appendChild(art);
    }
    const ang = Math.atan2(-(u.vy || 0), u.vx || 1);
    const r = el.clientWidth || 12;
    art.style.setProperty("--shard-ang", `${ang}rad`);
    art.style.setProperty("--shard-len", `${Math.max(22, r * 2.4)}px`);
  },
};

arena.registerShot("shatter", (fx, ctx) => {
  const { x, y, kind } = ctx;
  arena.spawnFx("fx-warden-flare", x, y, kind);
  arena.spawnFx("fx-warden-ring", x, y, kind);
  arena.spawnFx("fx-warden-ring", x, y, kind, { "--delay": "0.08s" });
  arena.burst(x, y, kind, 14);
});
