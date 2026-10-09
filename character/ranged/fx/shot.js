arena.registerShot("shot", (fx, ctx) => {
  const { x, y, kind } = ctx;
  const ang = Math.atan2(-(fx.vy || 0), fx.vx || 1);
  arena.spawnFx("fx-tracer-flare", x, y, kind);
  const beam = arena.spawnFx("fx-tracer-muzzle", x, y, kind, { "--ang": `${ang}rad` });
  if (beam) beam.style.transform = `translate(-50%, -50%) rotate(${ang}rad)`;
});
