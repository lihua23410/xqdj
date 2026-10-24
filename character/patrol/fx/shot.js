// 巡回的一次性特效。持 Pass 的标识物撞人时引擎不发通用 hit 火花，
// patrol-hit 是命中反馈的唯一来源：核心炸开 → 冲击环 → 沿撞击锥的碎屑火花，
// 三段 animation-delay 错峰，碎屑逐颗递增 delay，伤害越大 (--hit-power) 撞得越大。
(function () {
  arena.registerShot("patrol-throw", (fx, ctx) => {
    if (!fx || !ctx) return;
    arena.spawnFx("fx-patrol-throw-flash", ctx.x, ctx.y, fx.kind);
    const ang = Math.atan2(-(fx.vy || 0), fx.vx || 0);
    arena.spawnFx("fx-patrol-throw-streak", ctx.x, ctx.y, fx.kind, { "--ang": `${ang.toFixed(3)}rad` });
  });

  arena.registerShot("patrol-hit", (fx, ctx) => {
    if (!fx || !ctx) return;
    const power = Math.max(0, Math.min(1, ((fx.amount || 0) - 8) / 18));
    const extra = { "--hit-power": power.toFixed(3) };
    arena.spawnFx("fx-patrol-hit-core", ctx.x, ctx.y, fx.kind, extra);
    arena.spawnFx("fx-patrol-hit-ring", ctx.x, ctx.y, fx.kind, extra);
    const ang = Math.atan2(-(fx.vy || 0), fx.vx || 0);
    const n = 6 + Math.round(power * 4);
    for (let i = 0; i < n; i++) {
      const a = ang + (Math.random() - 0.5) * 1.9;
      const d = 16 + Math.random() * 26 + power * 14;
      const spark = arena.spawnFx("fx-patrol-spark", ctx.x, ctx.y, fx.kind, {
        "--dx": `${Math.cos(a) * d}px`,
        "--dy": `${Math.sin(a) * d}px`,
      });
      if (spark) spark.style.animationDelay = `${(0.09 + i * 0.022).toFixed(3)}s`;
    }
  });

  arena.registerShot("patrol-catch", (fx, ctx) => {
    if (!fx || !ctx) return;
    arena.spawnFx("fx-patrol-catch-ring", ctx.x, ctx.y, fx.kind);
    arena.spawnFx("fx-patrol-catch-flash", ctx.x, ctx.y, fx.kind);
  });

  arena.registerShot("patrol-bounce", (fx, ctx) => {
    if (!fx || !ctx) return;
    arena.spawnFx("fx-patrol-bounce-flash", ctx.x, ctx.y, fx.kind);
    const ang = Math.atan2(-(fx.vy || 0), fx.vx || 0);
    for (let i = 0; i < 3; i++) {
      const a = ang + (Math.random() - 0.5) * 1.4;
      const d = 8 + Math.random() * 12;
      arena.spawnFx("fx-patrol-spark", ctx.x, ctx.y, fx.kind, {
        "--dx": `${Math.cos(a) * d}px`,
        "--dy": `${Math.sin(a) * d}px`,
      });
    }
  });
})();
