// 非因果 FX 字段（与后端常量一致，给联调看）：
// spectrum      amount=射速秒  vx=柱数  vy=脉冲数
// spectrum-bin  slot=binIndex  amount=柱伤害  vx=权重  vy=射速秒
// doom-at       amount=T†秒  vx=GhostHP上限  vy=射速秒
// probe-hit     amount=伤害  vx=时间t
// lock-shot     slot=binIndex  amount=drawnAmount  vx/vy=初速
// lock-armed    slot=binIndex  amount=drawnAmount  unitId=弹
// lock-block    slot=binIndex  vx=挡住者id(墙为0)  vy=0墙/1单位
// ghost-hit     slot=binIndex  amount=drawnAmount  vx=剩余GhostHP
// ghost-down    amount=最后一发  vx=敌人id  vy=GhostHP上限
// doom          amount=T†

arena.registerShot("spectrum", (fx, ctx) => {
  arena.spawnFx("fx-ring", ctx.x, ctx.y, ctx.kind);
});

arena.registerShot("spectrum-bin", () => {});

arena.registerShot("doom-at", (fx, ctx) => {
  arena.spawnFx("fx-ring", ctx.x, ctx.y, ctx.kind);
});

arena.registerShot("probe-hit", (fx, ctx) => {
  arena.spawnFx("fx-flash", ctx.x, ctx.y, ctx.kind);
});

arena.registerShot("lock-shot", (fx, ctx) => {
  arena.spawnFx("fx-shock", ctx.x, ctx.y, ctx.kind);
});

arena.registerShot("lock-armed", () => {});

arena.registerShot("lock-block", (fx, ctx) => {
  arena.spawnFx("fx-flash", ctx.x, ctx.y, ctx.kind);
});

arena.registerShot("ghost-hit", (fx, ctx) => {
  arena.burst(ctx.x, ctx.y, ctx.kind, 8);
});

arena.registerShot("ghost-down", (fx, ctx) => {
  arena.spawnFx("fx-ring", ctx.x, ctx.y, ctx.kind);
  arena.burst(ctx.x, ctx.y, ctx.kind, 14);
});

arena.registerShot("doom", (fx, ctx) => {
  arena.spawnFx("fx-shock", ctx.x, ctx.y, ctx.kind);
});
