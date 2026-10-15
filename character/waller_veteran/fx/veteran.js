window.lookFX = window.lookFX || {};

window.lookFX.veteran = {
  guide(u, ctx) {
    if (!u || !ctx) return;
    const pose = window.masonPose && window.masonPose[u.id];
    if (!pose || performance.now() - pose.at > 400) {
      if (pose) delete window.masonPose[u.id];
      return;
    }
    if (pose.phase === 1) paintTicks(u, pose, ctx);
    spray(u, pose, ctx);
  },
};

function paintTicks(u, pose, ctx) {
  const walls = pose.walls || [];
  if (walls.length < 2) return;
  walls.forEach((w, i) => {
    const other = walls[i === 0 ? 1 : 0];
    const [x1, y1, x2, y2] = tickEnds(w, other);
    const [sx1, sy1] = arena.screenPos(x1, y1, ctx.scale, ctx.cx, ctx.cy);
    const [sx2, sy2] = arena.screenPos(x2, y2, ctx.scale, ctx.cx, ctx.cy);
    const g = ctx.ensureGuide(`mason-tick-${u.id}-${i}`, "mason-tick");
    ctx.placeSeg(g, sx1, sy1, sx2, sy2, u.kind);
    g.classList.toggle("held", w.amount >= 1);
    if (ctx.seenGuides) ctx.seenGuides.add(g.id);
  });
}

function tickEnds(w, other) {
  const dx = w.x2 - w.x1;
  const dy = w.y2 - w.y1;
  const len = Math.hypot(dx, dy) || 1;
  let nx = -dy / len;
  let ny = dx / len;
  const cx = (w.x1 + w.x2) / 2;
  const cy = (w.y1 + w.y2) / 2;
  const ox = (other.x1 + other.x2) / 2 - cx;
  const oy = (other.y1 + other.y2) / 2 - cy;
  if (nx * ox + ny * oy < 0) {
    nx = -nx;
    ny = -ny;
  }
  const reach = 16;
  return [cx, cy, cx + nx * reach, cy + ny * reach];
}

function spray(u, pose, ctx) {
  if (pose.phase !== 0 && pose.phase !== 2) return;
  if (pose.phase === 0 && pose.speed < 1) return;
  const now = performance.now();
  const gap = pose.phase === 0 ? 100 : Math.max(40, 130 - pose.speed * 0.12);
  if (now - (pose.dustAt || 0) < gap) return;
  pose.dustAt = now;
  const n = pose.phase === 0 ? 2 : Math.min(4, 2 + Math.floor(pose.speed / 200));
  const travel = pose.phase === 0 ? 16 : Math.min(72, 26 + pose.speed * 0.03);
  for (const wall of pose.walls || []) {
    const motion = motionOf(pose, wall);
    if (!motion) continue;
    for (let i = 0; i < n; i++) puff(u, wall, motion, travel, ctx);
  }
}

function motionOf(pose, wall) {
  const dx = pose.dx || 0;
  const dy = pose.dy || 0;
  if (Math.hypot(dx, dy) < 1e-6) return null;
  if (pose.phase === 0) return { x: dx, y: dy };
  const s = wall.amount < 0 ? -1 : 1;
  return { x: s * dx, y: s * dy };
}

function puff(u, wall, motion, travel, ctx) {
  const t = Math.random();
  const wx = wall.x1 + (wall.x2 - wall.x1) * t - motion.x * 4;
  const wy = wall.y1 + (wall.y2 - wall.y1) * t - motion.y * 4;
  const [sx, sy] = arena.screenPos(wx, wy, ctx.scale, ctx.cx, ctx.cy);
  const dist = travel * (ctx.scale || 1) * (0.65 + Math.random() * 0.5);
  const j = (Math.random() - 0.5) * 10;
  const cls = poseClass(u);
  arena.spawnFx(cls, sx, sy, u.kind, {
    "--dx": `${(-motion.x * dist - motion.y * j).toFixed(1)}px`,
    "--dy": `${(motion.y * dist - motion.x * j).toFixed(1)}px`,
  });
}

function poseClass(u) {
  const pose = window.masonPose && window.masonPose[u.id];
  return pose && pose.phase === 2 ? "fx-mason-dust ram" : "fx-mason-dust";
}
