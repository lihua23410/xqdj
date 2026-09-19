window.lookFX = window.lookFX || {};
window.lookFX.pull = { tick: fieldTick("pull") };
window.lookFX.push = { tick: fieldTick("push") };
window.lookFX["twin-arc"] = {
  unmount(el) {
    el?.querySelector(":scope > .blade-glow")?.remove();
    el?.querySelector(":scope > .blade-edge")?.remove();
    el?.querySelector(":scope > .blade-core")?.remove();
  },
  tick(el) {
    if (!el || !el.classList.contains("look-twin-arc")) return;
    for (const name of ["blade-glow", "blade-edge", "blade-core"]) {
      if (!el.querySelector(`:scope > .${name}`)) {
        const layer = document.createElement("i");
        layer.className = name;
        el.appendChild(layer);
      }
    }
  },
};
window.lookFX["void-orb"] = {
  unmount(el) {
    el?.querySelector(":scope > .void-rim")?.remove();
    el?.querySelector(":scope > .void-cross")?.remove();
  },
  tick(el) {
    if (!el || !el.classList.contains("look-void-orb")) return;
    for (const name of ["void-rim", "void-cross"]) {
      if (!el.querySelector(`:scope > .${name}`)) {
        const layer = document.createElement("i");
        layer.className = name;
        el.appendChild(layer);
      }
    }
  },
};

function fieldTick(mode) {
  return function (el, u, ctx) {
    if (!el) return;
    let ring = el.querySelector(":scope > .field-ring");
    if (!ring) {
      ring = document.createElement("div");
      ring.className = `field-ring ${mode}`;
      ring.appendChild(document.createElement("i"));
      el.appendChild(ring);
    }
    const scale = ctx.scale || 1;
    const fr = 72 * scale;
    ring.style.width = `${fr * 2}px`;
    ring.style.height = `${fr * 2}px`;
    ring.style.color = arena.kindColor(u.kind);
  };
}

window.lookFX.bond = {
  guide(u, ctx) {
    if (!u || u.role !== "fighter" || !ctx.ensureGuide || !ctx.placeSeg) return;
    const other = (ctx.units || []).find(
      (o) => o.slot === u.slot && o.id !== u.id && (arena.lookOf(o.kind).fx || []).includes("bond")
    );
    if (!other) return;
    const [x1, y1] = arena.screenPos(u.x, u.y, ctx.scale, ctx.cx, ctx.cy);
    const [x2, y2] = arena.screenPos(other.x, other.y, ctx.scale, ctx.cx, ctx.cy);
    const g = ctx.ensureGuide(`guide-bond-${u.slot}`, "bond");
    ctx.placeSeg(g, x1, y1, x2, y2, u.kind);
    g.style.setProperty("--kind", "#b44cff");
    const dist = Math.hypot(u.x - other.x, u.y - other.y);
    g.style.opacity = dist < 56 ? "0.9" : "0.42";
    if (ctx.seenGuides) ctx.seenGuides.add(g.id);
  },
};
