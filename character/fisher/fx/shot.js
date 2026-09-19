(function () {
  window.fisherHUD = window.fisherHUD || {};
  window.fisherWeight = window.fisherWeight || {};
  window.fisherCarry = window.fisherCarry || {};

  arena.registerShot("hud", (fx) => {
    if (!fx || fx.unitId == null) return;
    window.fisherHUD[fx.unitId] = {
      misses: fx.amount || 0,
      prog: fx.vx || 0,
      flags: fx.vy || 0,
    };
  });

  arena.registerShot("weight", (fx) => {
    if (!fx || fx.unitId == null) return;
    window.fisherWeight[fx.unitId] = fx.amount || 0;
  });

  arena.registerShot("cast", (fx, ctx) => {
    arena.spawnFx("fx-fisher-splash", ctx.x, ctx.y, fx.kind, { "--r": "70px" });
    for (let i = 0; i < 3; i++) {
      const a = Math.random() * Math.PI * 2;
      const d = 10 + Math.random() * 14;
      arena.spawnFx("fx-fisher-drop", ctx.x, ctx.y, fx.kind, {
        "--dx": `${Math.cos(a) * d}px`,
        "--dy": `${Math.sin(a) * d}px`,
      });
    }
  });

  arena.registerShot("reel", (fx, ctx) => {
    const r = Math.max(80, 48 + (fx.amount || 10) * 1.2);
    arena.spawnFx("fx-fisher-splash", ctx.x, ctx.y, fx.kind, { "--r": `${r}px` });
    arena.burst(ctx.x, ctx.y, fx.kind, 5);
  });

  arena.registerShot("miss", (fx, ctx) => {
    arena.spawnFx("fx-fisher-miss", ctx.x, ctx.y, fx.kind);
    arena.spawnFx("fx-flash", ctx.x, ctx.y, fx.kind);
  });

  arena.registerShot("ram", (fx, ctx) => {
    const r = Math.max(96, (fx.amount || 50) * 1.6);
    arena.spawnFx("fx-fisher-ram", ctx.x, ctx.y, fx.kind, { "--r": `${r}px` });
    arena.burst(ctx.x, ctx.y, fx.kind, 6);
  });

  arena.registerShot("carry", (fx) => {
    if (!fx || fx.unitId == null) return;
    window.fisherCarry = window.fisherCarry || {};
    window.fisherCarry[fx.unitId] = fx.amount || 0;
  });
})();
