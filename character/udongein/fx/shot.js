(function () {
  window.udongeinHUD = window.udongeinHUD || {};

  arena.registerShot("energy", (fx) => {
    if (!fx || fx.unitId == null) return;
    const prev = window.udongeinHUD[fx.unitId] || {};
    window.udongeinHUD[fx.unitId] = {
      ...prev,
      energy: fx.amount || 0,
      dose: fx.vx || 0,
      cap: fx.vy || 5,
    };
  });

  arena.registerShot("cards", (fx) => {
    if (!fx || fx.unitId == null) return;
    const packed = Math.round(fx.vx || 0) >>> 0;
    const n = Math.max(0, Math.min(5, Math.round(fx.vy || 0)));
    const cards = [];
    for (let i = 0; i < n; i++) {
      cards.push((packed >> (i * 4)) & 0xf);
    }
    const prev = window.udongeinHUD[fx.unitId] || {};
    window.udongeinHUD[fx.unitId] = {
      ...prev,
      cards,
      fill: Math.max(0, Math.min(1, fx.amount || 0)),
    };
  });

  arena.registerShot("shard-fade", (fx) => {
    if (!fx || fx.unitId == null) return;
    const prev = window.udongeinHUD[fx.unitId] || {};
    window.udongeinHUD[fx.unitId] = {
      ...prev,
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

  arena.registerShot("laser-warn", (fx, ctx) => {
    const ang = Math.atan2(-(fx.vy || 0), fx.vx || 1) * (180 / Math.PI);
    const len = 560 * ((ctx && ctx.scale) || 1);
    const ray = arena.spawnFx("fx-udongein-warn", ctx.x, ctx.y, fx.kind, {
      "--len": `${len}px`,
      "--t": `${fx.amount || 0.4}s`,
    });
    if (ray) ray.style.transform = `translate(0, -50%) rotate(${ang}deg)`;
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

  arena.registerShot("blast", (fx, ctx) => {
    const r = Math.max(48, (fx.amount || 168) * (ctx.scale || 1) * 2);
    const size = { "--r": `${r}px` };
    arena.spawnFx("fx-udongein-blast-core", ctx.x, ctx.y, fx.kind);
    arena.spawnFx("fx-udongein-blast-wave", ctx.x, ctx.y, fx.kind, size);
    arena.spawnFx("fx-udongein-blast-wave", ctx.x, ctx.y, fx.kind, {
      "--r": `${r}px`,
      "--delay": "0.1s",
    });
    arena.spawnFx("fx-udongein-blast-ring", ctx.x, ctx.y, fx.kind, size);
    arena.spawnFx("fx-flash", ctx.x, ctx.y, fx.kind);
    arena.spawnFx("fx-shock", ctx.x, ctx.y, fx.kind);
    arena.burst(ctx.x, ctx.y, fx.kind, 28);
  });
})();
