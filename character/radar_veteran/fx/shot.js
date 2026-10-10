(function () {
  window.radarVetBeam = window.radarVetBeam || {};
  window.radarVetMark = window.radarVetMark || {};

  arena.registerShot("beam", (fx) => {
    if (!fx || fx.unitId == null) return;
    if (fx.kind !== "雷达(百战)") return;
    window.radarVetBeam[fx.unitId] = {
      dx: fx.vx || 0,
      dy: fx.vy || 0,
      len: fx.amount || 0,
    };
  });

  arena.registerShot("mark", (fx) => {
    if (!fx || fx.unitId == null) return;
    window.radarVetMark[fx.unitId] = { r: fx.amount || 4 };
  });

  arena.registerShot("blast", (fx, ctx) => {
    if (!fx) return;
    if (fx.kind === "大炮击") {
      const r = Math.max(48, (fx.amount || 80) * (ctx.scale || 1) * 2);
      const size = { "--r": `${r}px` };
      arena.spawnFx("fx-radar-vet-blast", ctx.x, ctx.y, fx.kind, size);
      arena.spawnFx("fx-radar-vet-blast-ring", ctx.x, ctx.y, fx.kind, size);
      arena.burst(ctx.x, ctx.y, fx.kind, 28);
      return;
    }
    if (fx.kind === "小炮击") {
      const r = Math.max(28, (fx.amount || 55) * (ctx.scale || 1) * 2);
      arena.spawnFx("fx-radar-vet-blast-sm", ctx.x, ctx.y, fx.kind, {
        "--r": `${r}px`,
      });
      arena.burst(ctx.x, ctx.y, fx.kind, 12);
    }
  });
})();
