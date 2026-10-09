(function () {
  arena.registerShot("shot", (fx, ctx) => {
    const ang = Math.atan2(-(fx.vy || 0), fx.vx || 1);
    const beam = arena.spawnFx("fx-chroma-muzzle", ctx.x, ctx.y, ctx.kind, { "--ang": `${ang}rad` });
    if (beam) beam.style.transform = `translate(-50%, -50%) rotate(${ang}rad)`;
  });
})();
