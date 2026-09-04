arena.registerShot("shot", (fx, ctx) => {
  const { x, y, kind } = ctx;
  arena.spawnFx("fx-flash", x, y, kind);
  const ang = Math.atan2(-(fx.vy || 0), fx.vx || 1) * (180 / Math.PI);
  const beam = arena.spawnFx("fx-beam", x, y, kind);
  if (beam) beam.style.transform = `translate(0, -50%) rotate(${ang}deg)`;
});
