(function () {
  window.engineHeat = window.engineHeat || {};

  arena.registerShot("heat", (fx) => {
    if (!fx || fx.unitId == null) return;
    window.engineHeat[fx.unitId] = {
      gear: fx.amount || 1,
      weak: fx.vx || 0,
      frail: (fx.vy || 0) > 0,
    };
  });

  arena.registerShot("blast", (fx, ctx) => {
    const r = Math.max(48, (fx.amount || 96) * (ctx.scale || 1) * 2);
    const size = { "--r": `${r}px` };
    arena.spawnFx("fx-engine-blast-core", ctx.x, ctx.y, fx.kind);
    arena.spawnFx("fx-engine-blast-wave", ctx.x, ctx.y, fx.kind, size);
    arena.burst(ctx.x, ctx.y, fx.kind, 6);
  });
})();
