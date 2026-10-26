window.knightRing = window.knightRing || {};

arena.registerShot("fire-ring", (fx) => {
  if (!fx || fx.unitId == null) return;
  window.knightRing[fx.unitId] = {
    on: (fx.amount || 0) > 0,
    readyAt: fx.vx || 0,
    start: fx.vy || 0,
  };
});

arena.registerShot("blink", (fx, ctx) => {
  const { x, y, kind, scale } = ctx;
  const d = 36 * (scale || 1);
  const ghostAt = (px, py) => {
    knightPix(px, py, kind, d, d, knightDoomDisc(12));
  };
  ghostAt(x, y);
  const [lx, ly] = arena.screenPos(Number(fx.vx) || 0, Number(fx.vy) || 0, scale, ctx.cx, ctx.cy);
  ghostAt(lx, ly);
  const dx = lx - x;
  const dy = ly - y;
  const len = Math.max(8, Math.hypot(dx, dy));
  const ang = Math.atan2(dy, dx);
  const cols = Math.max(6, Math.round(len / 6));
  const cut = knightPix(x, y, kind, len, Math.max(6, d * 0.18), knightDoomStrip(cols), {
    "--ang": `${ang}rad`,
  });
  if (cut) cut.classList.add("knight-cut-pix");
});

arena.registerShot("dash", (fx, ctx) => {
  const { x, y, kind, scale } = ctx;
  const d = 40 * (scale || 1);
  knightPix(x, y, kind, d, d * 1.3, knightDoomBurst(14));
});

arena.registerShot("lock", (fx, ctx) => {
  const { x, y, kind, scale } = ctx;
  const d = 44 * (scale || 1);
  knightPix(x, y, kind, d, d * 1.35, knightDoomBurst(16));
});

arena.registerShot("lock-hit", (fx, ctx) => {
  const { x, y, kind } = ctx;
  for (let i = 0; i < 4; i++) {
    const a = (Math.PI * 2 * i) / 4 + 0.4;
    knightPix(x + Math.cos(a) * 10, y + Math.sin(a) * 10, kind, 16, 20, knightDoomBurst(8));
  }
});

arena.registerShot("lock-end", (fx, ctx) => {
  const { x, y, kind, scale } = ctx;
  const d = 36 * (scale || 1);
  knightPix(x, y, kind, d, d, knightDoomDisc(10));
});
