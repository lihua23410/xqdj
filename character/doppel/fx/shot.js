(function () {
  arena.registerShot('doppel-touch', (fx, ctx) => {
    if (ctx.audioOnly) return;
    arena.spawnFx('fx-doppel-touch', ctx.x, ctx.y, ctx.kind);
  });
})();
