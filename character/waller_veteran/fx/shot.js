window.masonPose = window.masonPose || {};

arena.registerShot("slam", (fx, ctx) => {
  const { x, y, kind, scale } = ctx;
  const r = 2 * (fx.amount || 0) * scale;
  arena.spawnFx("fx-slam-ring", x, y, kind, {
    "--r": `${Math.max(r, 12)}px`,
  });
});

arena.registerShot("mason", (fx) => {
  if (!fx || fx.unitId == null || fx.kind !== "筑墙者(百战)") return;
  if ((fx.amount ?? 0) < 0) {
    delete window.masonPose[fx.unitId];
    return;
  }
  const prev = window.masonPose[fx.unitId] || {};
  window.masonPose[fx.unitId] = {
    phase: fx.amount || 0,
    speed: fx.x || 0,
    dx: fx.vx || 0,
    dy: fx.vy || 0,
    walls: [],
    at: performance.now(),
    dustAt: prev.dustAt || 0,
  };
});

arena.registerShot("course", (fx) => {
  if (!fx || fx.unitId == null || fx.kind !== "筑墙者(百战)") return;
  const pose = window.masonPose[fx.unitId];
  if (!pose) return;
  pose.walls.push({
    x1: fx.x || 0,
    y1: fx.y || 0,
    x2: fx.vx || 0,
    y2: fx.vy || 0,
    amount: fx.amount || 0,
  });
  pose.at = performance.now();
});

arena.registerShot("shard", (fx, ctx) => {
  if (!fx || !ctx || fx.kind !== "筑墙者(百战)") return;
  const scale = ctx.scale || 1;
  const ang = Math.atan2(-(fx.vy || 0), fx.vx || 0);
  for (let i = 0; i < 3; i++) {
    const spread = (i - 1) * 0.22;
    const a = ang + spread;
    const d = (40 + i * 26) * scale;
    const el = arena.spawnFx("fx-mason-shard", ctx.x, ctx.y, fx.kind, {
      "--dx": `${Math.cos(a) * d}px`,
      "--dy": `${Math.sin(a) * d}px`,
      "--spin": `${spread * 70}deg`,
    });
    if (el) {
      el.style.width = `${Math.max(12, 18 * scale)}px`;
      el.style.height = `${Math.max(7, 9 * scale)}px`;
    }
  }
});
