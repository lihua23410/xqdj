(function () {
  window.udongeinHUD = window.udongeinHUD || {};

  arena.registerShot("energy", (fx) => {
    if (!fx || fx.unitId == null) return;
    window.udongeinHUD[fx.unitId] = {
      energy: fx.amount || 0,
      dose: fx.vx || 0,
      fade: (window.udongeinHUD[fx.unitId] || {}).fade,
    };
  });

  arena.registerShot("shard-fade", (fx) => {
    if (!fx || fx.unitId == null) return;
    const prev = window.udongeinHUD[fx.unitId] || {};
    window.udongeinHUD[fx.unitId] = {
      energy: prev.energy,
      dose: prev.dose,
      fade: Math.max(0, Math.min(1, fx.amount || 0)),
    };
  });

  arena.registerShot("shot", (fx, ctx) => {
    arena.spawnFx("fx-flash", ctx.x, ctx.y, fx.kind);
    const ang = Math.atan2(-(fx.vy || 0), fx.vx || 1) * (180 / Math.PI);
    const beam = arena.spawnFx("fx-beam", ctx.x, ctx.y, fx.kind);
    if (beam) beam.style.transform = `translate(0, -50%) rotate(${ang}deg)`;
  });

  arena.registerShot("break", (fx, ctx) => {
    const r = Math.max(48, (fx.amount || 72) * (ctx.scale || 1) * 2);
    arena.spawnFx("fx-udongein-ring", ctx.x, ctx.y, fx.kind, { "--r": `${r}px` });
    arena.spawnFx("fx-flash", ctx.x, ctx.y, fx.kind);
    arena.burst(ctx.x, ctx.y, fx.kind, 16);
  });

  arena.registerShot("laser", (fx, ctx) => {
    arena.spawnFx("fx-flash", ctx.x, ctx.y, fx.kind);
    arena.burst(ctx.x, ctx.y, fx.kind, 10);
  });

  arena.registerShot("clone", (fx, ctx) => {
    arena.spawnFx("fx-ring", ctx.x, ctx.y, fx.kind);
    arena.spawnFx("fx-flash", ctx.x, ctx.y, fx.kind);
  });

  arena.registerShot("gas", (fx, ctx) => {
    const r = Math.max(80, (fx.amount || 54) * (ctx.scale || 1) * 2);
    arena.spawnFx("fx-udongein-ring", ctx.x, ctx.y, fx.kind, { "--r": `${r}px` });
    arena.spawnFx("fx-flash", ctx.x, ctx.y, fx.kind);
  });

  arena.registerShot("crown", (fx, ctx) => {
    arena.spawnFx("fx-flash", ctx.x, ctx.y, fx.kind);
    arena.spawnFx("fx-ring", ctx.x, ctx.y, fx.kind);
  });

  arena.registerShot("dose", (fx, ctx) => {
    arena.spawnFx("fx-udongein-dose", ctx.x, ctx.y, fx.kind);
    arena.spawnFx("fx-udongein-dose-ring", ctx.x, ctx.y, fx.kind);
    for (let i = 0; i < 10; i++) {
      const a = -Math.PI / 2 + (Math.random() - 0.5) * 1.4;
      const d = 18 + Math.random() * 30;
      arena.spawnFx("fx-udongein-dose-spark", ctx.x, ctx.y, fx.kind, {
        "--dx": `${Math.cos(a) * d * 0.4}px`,
        "--dy": `${-Math.abs(Math.sin(a) * d) - 10}px`,
      });
    }
    if ((fx.amount || 0) >= 4) {
      arena.spawnFx("fx-udongein-boom", ctx.x, ctx.y, fx.kind);
      arena.burst(ctx.x, ctx.y, fx.kind, 22);
    }
  });
})();
