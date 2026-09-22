arena.registerShot("slam", (fx, ctx) => {
  const { x, y, kind, scale } = ctx;
  const r = 2 * (fx.amount || 0) * scale;
  arena.spawnFx("fx-slam-ring", x, y, kind, {
    "--r": `${Math.max(r, 12)}px`,
  });
});
