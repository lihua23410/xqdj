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

arena.registerShot("command", (fx, ctx) => {
  arena.spawnFx("fx-command", ctx.x, ctx.y, ctx.kind, {
    "--bonus": String(fx.amount || 0),
  });
});

arena.registerShot("bite", (fx, ctx) => {
  arena.spawnFx("fx-bite", ctx.x, ctx.y, ctx.kind);
});
