(function () {
  arena.registerShot("shot", (fx, ctx) => {
    arena.spawnFx("fx-flash", ctx.x, ctx.y, ctx.kind);
    const ang = Math.atan2(-(fx.vy || 0), fx.vx || 1) * (180 / Math.PI);
    const beam = arena.spawnFx("fx-beam", ctx.x, ctx.y, ctx.kind);
    if (beam) beam.style.transform = `translate(0, -50%) rotate(${ang}deg)`;
  });
})();
