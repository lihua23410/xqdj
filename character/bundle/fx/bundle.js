// 捆前端：后端生成和判定的绳迹线段在这里原样绘制。
// 被绳子捕获的目标会沿绳路退回绳尾，并显示单独的束缚环。
window.bundleSt = window.bundleSt || {};
window.lookFX = window.lookFX || {};

(function () {
  const hooks = window.lookFX;

  function state(id) {
    let s = window.bundleSt[id];
    if (!s) {
      s = {
        time: -Infinity,
        segments: new Map(),
        riders: new Set(),
        riderCount: 0,
        hitUntil: -Infinity,
      };
      window.bundleSt[id] = s;
    }
    return s;
  }

  function sync(id, time) {
    const s = state(id);
    if (Number.isFinite(time) && time + 1e-6 < s.time) {
      s.segments.clear();
      s.riders.clear();
      s.riderCount = 0;
      s.hitUntil = -Infinity;
    }
    if (Number.isFinite(time)) s.time = time;
    return s;
  }

  function ensure(el, cls, tag = "div") {
    let node = el.querySelector(`:scope > .${cls}`);
    if (!node) {
      node = document.createElement(tag);
      node.className = cls;
      el.appendChild(node);
    }
    return node;
  }

  function timeOf(ctx) {
    return ctx && Number.isFinite(ctx.time) ? ctx.time : 0;
  }

  function spawnEffect(ctx, cls) {
    if (!ctx || ctx.audioOnly) return null;
    const root = document.getElementById("fx");
    if (!root) return null;
    const node = document.createElement("div");
    node.className = `fx ${cls}`;
    node.style.left = `${ctx.x}px`;
    node.style.top = `${ctx.y}px`;
    root.appendChild(node);
    node.addEventListener("animationend", () => node.remove(), { once: true });
    setTimeout(() => node.remove(), 900);
    return node;
  }

  const bodyArt = `<div class="bundle-body">
    <span class="bundle-coil c0"></span>
    <span class="bundle-coil c1"></span>
    <span class="bundle-coil c2"></span>
    <span class="bundle-fiber f0"></span>
    <span class="bundle-fiber f1"></span>
    <span class="bundle-knot"></span>
  </div>
  <div class="bundle-pips"><i></i><i></i></div>`;

  hooks.bundle = {
    unmount(el) {
      if (!el) return;
      el.querySelector(":scope > .bundle-art")?.remove();
      el.classList.remove("bundle-binding", "bundle-hit");
    },
    tick(el, u, ctx = {}) {
      if (!el || !el.classList.contains("look-bundle")) return;
      const s = sync(u.id, ctx.time);
      const art = ensure(el, "bundle-art");
      if (!art.firstElementChild) art.innerHTML = bodyArt;
      el.classList.toggle("bundle-binding", s.riderCount > 0);
      el.classList.toggle("bundle-hit", timeOf(ctx) < s.hitUntil);
      const pips = art.querySelectorAll(".bundle-pips i");
      pips.forEach((node, i) => {
        node.classList.toggle("on", i === 0 ? s.segments.size > 0 : s.riderCount > 0);
      });
    },
    guide(u, ctx) {
      if (!u || !ctx || !ctx.ensureGuide || !ctx.placeSeg || !window.arena) return;
      const s = sync(u.id, ctx.time);
      for (const seg of s.segments.values()) {
        const [x1, y1] = arena.screenPos(seg.ax, seg.ay, ctx.scale, ctx.cx, ctx.cy);
        const [x2, y2] = arena.screenPos(seg.bx, seg.by, ctx.scale, ctx.cx, ctx.cy);
        const node = ctx.ensureGuide(`bundle-rope-${u.id}-${seg.id}`, "bundle-rope");
        node.className = "guide bundle-rope";
        ctx.placeSeg(node, x1, y1, x2, y2, u.kind);
        if (ctx.seenGuides) ctx.seenGuides.add(node.id);
      }
      for (const targetId of s.riders) {
        const target = (ctx.units || []).find((o) => o && o.id === targetId);
        if (!target) continue;
        const [x, y] = arena.screenPos(target.x, target.y, ctx.scale, ctx.cx, ctx.cy);
        const node = ctx.ensureGuide(`bundle-rider-${u.id}-${targetId}`, "bundle-rider-ring");
        node.className = "guide bundle-rider-ring";
        node.style.left = `${x}px`;
        node.style.top = `${y}px`;
        node.style.width = `${(target.radius + 9) * 2}px`;
        node.style.height = `${(target.radius + 9) * 2}px`;
        node.style.setProperty("--kind", arena.kindColor(u.kind));
        if (ctx.seenGuides) ctx.seenGuides.add(node.id);
      }
    },
  };

  arena.registerShot("bundle-reset", (fx) => {
    const s = state(fx.unitId);
    s.segments.clear();
    s.riders.clear();
    s.riderCount = 0;
  });

  arena.registerShot("bundle-state", (fx) => {
    const s = state(fx.unitId);
    s.riderCount = Math.max(0, Math.round(Number(fx.amount) || 0));
  });

  arena.registerShot("bundle-trail-add", (fx) => {
    const s = state(fx.unitId);
    const id = Math.round(Number(fx.amount) || 0);
    if (!id) return;
    s.segments.set(id, {
      id,
      ax: Number(fx.x) || 0,
      ay: Number(fx.y) || 0,
      bx: Number(fx.vx) || 0,
      by: Number(fx.vy) || 0,
    });
  });

  arena.registerShot("bundle-trail-remove", (fx) => {
    state(fx.unitId).segments.delete(Math.round(Number(fx.amount) || 0));
  });

  arena.registerShot("bundle-rope-catch", (fx, ctx) => {
    const s = state(fx.unitId);
    const targetId = Math.round(Number(fx.vx) || 0);
    if (targetId) s.riders.add(targetId);
    spawnEffect(ctx, "fx-bundle-rope-catch");
    if (ctx && window.arena) arena.burst(ctx.x, ctx.y, fx.kind, 7);
  });

  arena.registerShot("bundle-rope-burn", (fx, ctx) => {
    const s = state(fx.unitId);
    s.hitUntil = timeOf(ctx) + 0.12;
    spawnEffect(ctx, "fx-bundle-rope-burn");
  });

  arena.registerShot("bundle-rope-release", (fx, ctx) => {
    const s = state(fx.unitId);
    const targetId = Math.round(Number(fx.vx) || 0);
    if (targetId) s.riders.delete(targetId);
    spawnEffect(ctx, "fx-bundle-rope-release");
  });
})();
