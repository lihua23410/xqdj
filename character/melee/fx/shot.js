arena.registerShot("dash", (fx, ctx) => {
  const { x, y, kind } = ctx;
  arena.spawnFx("fx-melee-shock", x, y, kind);
  arena.burst(x, y, kind, 5);
});
