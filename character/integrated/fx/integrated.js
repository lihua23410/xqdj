window.lookFX = window.lookFX || {};
window.integratedTrack = window.integratedTrack || {};

const nearRadius = 185;
const fleeTrailAt = 148;

window.lookFX.integrated = {
  tick(el, u, ctx) {
    if (!el || !u || !ctx || !ctx.spawnGhost) return;
    if (Math.hypot(u.vx || 0, u.vy || 0) <= fleeTrailAt) return;
    const now = ctx.now || 0;
    if (now - (el._fleeAt || 0) < 16) return;
    el._fleeAt = now;
    const g = ctx.spawnGhost(el, u.kind, ctx.fxRoot, "trail");
    if (g) g.classList.add("integrated-ghost");
  },
  guide(u, ctx) {
    if (!u || u.role !== "fighter" || !ctx || !ctx.ensureGuide) return;
    const g = ctx.ensureGuide(`guide-near-${u.id}`, "integrated-ring");
    const d = nearRadius * 2 * ctx.scale;
    g.style.width = `${d}px`;
    g.style.height = `${d}px`;
    g.style.left = `${ctx.cx + u.x * ctx.scale}px`;
    g.style.top = `${ctx.cy - u.y * ctx.scale}px`;
    const engaged = (ctx.units || []).some((o) => {
      if (!o || o.id === u.id || o.slot === u.slot) return false;
      if (o.role !== "fighter" && !o.mortal) return false;
      return Math.hypot(o.x - u.x, o.y - u.y) <= nearRadius;
    });
    g.classList.toggle("alert", engaged);
    if (ctx.seenGuides) ctx.seenGuides.add(g.id);
    const tid = window.integratedTrack[u.id];
    if (!tid) return;
    const target = (ctx.units || []).find((o) => o.id === tid);
    if (!target) return;
    const m = ctx.ensureGuide(`guide-mark-${u.id}`, "integrated-mark");
    const mark = Math.max(target.radius || 18, 8) * 2.6 * ctx.scale;
    m.style.width = `${mark}px`;
    m.style.height = `${mark}px`;
    m.style.left = `${ctx.cx + target.x * ctx.scale}px`;
    m.style.top = `${ctx.cy - target.y * ctx.scale}px`;
    if (ctx.seenGuides) ctx.seenGuides.add(m.id);
  },
};

window.lookFX["integrated-shot"] = {
  tick(el, u, ctx) {
    if (!el || !u || !ctx || !ctx.spawnGhost) return;
    const ownerAlive = (ctx.units || []).some((o) => o.id === u.ownerId);
    if (!ownerAlive || !window.integratedTrack[u.ownerId]) return;
    const now = ctx.now || 0;
    if (now - (el._trailAt || 0) < 16) return;
    el._trailAt = now;
    const g = ctx.spawnGhost(el, u.kind, ctx.fxRoot, "trail");
    if (g) g.classList.add("integrated-ghost");
  },
};

arena.registerShot("track", (fx) => {
  if (!fx || !fx.unitId) return;
  if (fx.amount > 0) window.integratedTrack[fx.unitId] = fx.vx;
  else delete window.integratedTrack[fx.unitId];
});
