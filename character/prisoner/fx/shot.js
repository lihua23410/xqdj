(function () {
  window.prisonerChain = window.prisonerChain || {};
  window.prisonerDrops = window.prisonerDrops || {};

  arena.registerShot("chain", (fx) => {
    if (!fx || fx.unitId == null) return;
    window.prisonerChain[fx.unitId] = {
      hx: fx.vx,
      hy: fx.vy,
      len: fx.amount || 0,
    };
  });

  arena.registerShot("drop", (fx) => {
    if (!fx) return;
    const key = `${fx.unitId || 0}:${fx.kind}`;
    window.prisonerDrops[key] = { x: fx.x, y: fx.y, kind: fx.kind, until: performance.now() + 120 };
  });

  arena.registerShot("bolt", (fx, ctx) => {
    if (!fx || !ctx) return;
    const [x2, y2] = arena.screenPos(fx.vx, fx.vy, ctx.scale, ctx.cx, ctx.cy);
    const dx = x2 - ctx.x;
    const dy = y2 - ctx.y;
    const len = Math.hypot(dx, dy);
    arena.spawnFx("fx-bolt", (ctx.x + x2) / 2, (ctx.y + y2) / 2, fx.kind, {
      "--len": `${Math.max(8, len)}px`,
      "--ang": `${Math.atan2(dy, dx)}rad`,
    });
  });

  arena.registerShot("scream", (fx, ctx) => {
    const el = arena.spawnFx("fx-scream", ctx.x, ctx.y - 18, fx.kind);
    if (el) el.textContent = "啊";
  });
})();
