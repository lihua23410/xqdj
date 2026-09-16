(function () {
  const colors = ["#3ec8e0", "#ff3b3b", "#b44cff", "#8dffb0"];

  arena.registerShot("shot", (fx, ctx) => {
    const { x, y, kind } = ctx;
    arena.spawnFx("fx-flash", x, y, kind);
    const ang = Math.atan2(-(fx.vy || 0), fx.vx || 1) * (180 / Math.PI);
    const beam = arena.spawnFx("fx-beam", x, y, kind);
    if (beam) beam.style.transform = `translate(0, -50%) rotate(${ang}deg)`;
  });

  arena.registerShot("blast", (fx, ctx) => {
    const r = Math.max(160, (fx.amount || 54) * (ctx.scale || 1) * 2);
    const size = { "--r": `${r}px` };
    arena.spawnFx("fx-menreiki-blast-fill", ctx.x, ctx.y, fx.kind, size);
    arena.spawnFx("fx-menreiki-blast-core", ctx.x, ctx.y, fx.kind, size);
    colors.forEach((c, i) => {
      arena.spawnFx("fx-menreiki-blast-wave", ctx.x, ctx.y, fx.kind, {
        "--r": `${r}px`,
        "--c": c,
        "--delay": `${i * 0.07}s`,
      });
    });
    for (let i = 0; i < 8; i++) {
      const a = (Math.PI / 4) * i - Math.PI / 8;
      const c = colors[i % colors.length];
      arena.spawnFx("fx-menreiki-blast-shard", ctx.x, ctx.y, fx.kind, {
        "--c": c,
        "--dx": `${Math.cos(a) * r * 0.46}px`,
        "--dy": `${Math.sin(a) * r * 0.46}px`,
        "--ang": `${a}rad`,
      });
    }
    arena.spawnFx("fx-menreiki-blast-ring", ctx.x, ctx.y, fx.kind, size);
    arena.spawnFx("fx-flash", ctx.x, ctx.y, fx.kind);
    arena.spawnFx("fx-shock", ctx.x, ctx.y, fx.kind);
    arena.burst(ctx.x, ctx.y, fx.kind, 28);
  });
})();
