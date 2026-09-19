arena.registerShot("blink", (fx, ctx) => {
  const { x, y, kind } = ctx;
  arena.spawnFx("fx-knight-blink", x, y, kind);
});

arena.registerShot("dash", (fx, ctx) => {
  const { x, y, kind } = ctx;
  arena.spawnFx("fx-knight-shock", x, y, kind);
  arena.burst(x, y, kind, 5);
});
