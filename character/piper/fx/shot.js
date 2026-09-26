arena.registerShot("pipe", (fx, ctx) => {
  arena.spawnFx("fx-pipe", ctx.x, ctx.y, ctx.kind);
});

arena.registerShot("snatch", (fx, ctx) => {
  arena.spawnFx("fx-snatch", ctx.x, ctx.y, ctx.kind);
  arena.burst(ctx.x, ctx.y, ctx.kind, 5);
});

arena.registerShot("stash", (fx, ctx) => {
  arena.spawnFx("fx-stash", ctx.x, ctx.y, ctx.kind);
});

arena.registerShot("spill", (fx, ctx) => {
  arena.spawnFx("fx-spill", ctx.x, ctx.y, ctx.kind);
});

arena.registerShot("plague", (fx, ctx) => {
  arena.spawnFx("fx-plague", ctx.x, ctx.y, ctx.kind);
});

arena.registerShot("plague-tick", (fx, ctx) => {
  arena.spawnFx("fx-plague-tick", ctx.x, ctx.y, ctx.kind);
});
