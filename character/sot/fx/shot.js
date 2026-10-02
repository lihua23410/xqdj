arena.registerShot("sot-drop", (fx, ctx) => {
  arena.spawnFx("fx-sot-shatter", ctx.x, ctx.y, ctx.kind);
});

arena.registerShot("sot-shatter", (fx, ctx) => {
  arena.spawnFx("fx-sot-shatter", ctx.x, ctx.y, ctx.kind);
  arena.burst(ctx.x, ctx.y, ctx.kind, 6);
});
