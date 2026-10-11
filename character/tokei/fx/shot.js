// 时计的四个 FX：
//   clock  本体表针状态：Amount = 已提交小时读数、VX = 停针/满仓、X/Y = 本拍窗口的起止模拟时刻
//   moment 每个时刻残影自己的针位（Amount = 落下钟点，UnitID = 残影实体）——残影的针停在它刻下时的时间
//   rewind 回拨：旧位置 → 档位拉一条影子，落点闪一下
//   trip   踩影拉撞：时刻位置爆一圈
arena.registerShot("clock", (fx) => {
  window.tokeiSt = window.tokeiSt || {};
  window.tokeiSt[fx.unitId] = {
    hours: Number(fx.amount) || 0,
    full: (fx.vx || 0) > 0,
    from: Number(fx.x) || 0,
    to: Number(fx.y) || 0,
  };
});

arena.registerShot("moment", (fx) => {
  window.tokeiSt = window.tokeiSt || {};
  // 残影没有窗口（from/to 都是 0），针就停在它刻下那一刻的读数上
  window.tokeiSt[fx.unitId] = { hours: Number(fx.amount) || 0, full: false, ghost: true, from: 0, to: 0 };
});

arena.registerShot("rewind", (fx, ctx) => {
  const from = { x: ctx.x, y: ctx.y };
  const to = { x: fx.vx || 0, y: fx.vy || 0 };
  arena.spawnFx("fx-tokei-rewind", from.x, from.y, ctx.kind, {
    "--dx": `${to.x - from.x}px`,
    "--dy": `${to.y - from.y}px`,
  });
  arena.spawnFx("fx-flash", to.x, to.y, ctx.kind);
  arena.spawnFx("fx-ring", to.x, to.y, ctx.kind);
});

arena.registerShot("trip", (fx, ctx) => {
  arena.spawnFx("fx-shock", ctx.x, ctx.y, ctx.kind);
  arena.spawnFx("fx-ring", ctx.x, ctx.y, ctx.kind);
  arena.burst(ctx.x, ctx.y, ctx.kind, 10);
});
