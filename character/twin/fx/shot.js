arena.registerShot("split", (fx, ctx) => {
  const { x, y, kind } = ctx;
  const ang = Math.atan2(-(fx.vy || 0), fx.vx || 1);
  arena.spawnFx("fx-flash", x, y, kind);
  arena.spawnFx("fx-ring", x, y, kind);
  arena.spawnFx("fx-split-slash", x, y, kind, { "--ang": `${ang}rad` });
  arena.burst(x, y, kind, 8);
});

arena.registerShot("void-shot", (fx, ctx) => {
  const { x, y, kind } = ctx;
  const ang = Math.atan2(-(fx.vy || 0), fx.vx || 1);
  arena.spawnFx("fx-flash", x, y, kind);
  arena.spawnFx("fx-void-core", x, y, kind);
  arena.spawnFx("fx-ring", x, y, kind);
  const beam = arena.spawnFx("fx-beam void", x, y, kind);
  if (beam) beam.style.transform = `translate(0, -50%) rotate(${ang * (180 / Math.PI)}deg)`;
  arena.burst(x, y, kind, 8);
});

arena.registerShot("void-hit", (fx, ctx) => {
  const { x, y, kind } = ctx;
  arena.spawnFx("fx-flash", x, y, kind);
  arena.spawnFx("fx-void-core", x, y, kind);
  arena.spawnFx("fx-shock", x, y, kind);
  arena.burst(x, y, kind, 10);
});
