arena.registerShot("shatter", (fx, ctx) => {
  const { x, y, kind } = ctx;
  arena.spawnFx("fx-warden-shatter", x, y, kind);
  arena.burst(x, y, kind, 6);
});
