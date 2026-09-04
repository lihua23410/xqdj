arena.registerShot("shatter", (fx, ctx) => {
  const { x, y, kind } = ctx;
  arena.spawnFx("fx-flash", x, y, kind);
  arena.spawnFx("fx-ring", x, y, kind);
  arena.spawnFx("fx-shock", x, y, kind);
  arena.burst(x, y, kind, 14);
});
