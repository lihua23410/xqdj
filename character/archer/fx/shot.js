// 弓箭的 FX：
//   face   瞄准方向（站定蓄力时速度是 0，靠它把弓转对方向）
//   draw   开始蓄力（Amount = 站定秒数）→ 弓弦往后拉
//   shoot  出膛一支箭 → 弦回弹 + 枪口闪
//   volley 四支贴排同射（Amount = 支数）→ 一排四道轨迹
//   cancel 收弓（蓄力被打断，这一箭哑火）
//   dash   后撤步滑步（VX/VY = 安全向）→ 落地尘
//   roll   翻滚（VX/VY = 安全向）→ 落地尘
arena.registerShot("face", (fx) => {
  window.archerSt = window.archerSt || {};
  const st = (window.archerSt[fx.unitId] = window.archerSt[fx.unitId] || { face: 0, draw: 0 });
  // 存 CSS 角（世界 y 朝上 → 屏幕 y 朝下，取负），供站定蓄力时沿用
  const deg = window.archerFaceDeg(fx.vx, fx.vy);
  if (deg !== null) st.face = deg;
});

arena.registerShot("draw", (fx) => {
  window.archerSt = window.archerSt || {};
  const st = (window.archerSt[fx.unitId] = window.archerSt[fx.unitId] || { face: 0, draw: 0 });
  st.draw = Number(fx.amount) || 1;
});

function archerRelease(fx) {
  window.archerSt = window.archerSt || {};
  const st = window.archerSt[fx.unitId];
  if (st) st.draw = 0;
}

arena.registerShot("shoot", (fx, ctx) => {
  archerRelease(fx);
  arena.spawnFx("fx-flash", ctx.x, ctx.y, ctx.kind);
});

arena.registerShot("volley", (fx, ctx) => {
  archerRelease(fx);
  arena.spawnFx("fx-flash", ctx.x, ctx.y, ctx.kind);
  arena.spawnFx("fx-archer-volley", ctx.x, ctx.y, ctx.kind);
});

arena.registerShot("cancel", (fx, ctx) => {
  archerRelease(fx);
  arena.spawnFx("fx-ring", ctx.x, ctx.y, ctx.kind);
});

arena.registerShot("dash", (fx, ctx) => {
  arena.spawnFx("fx-archer-dash", ctx.x, ctx.y, ctx.kind);
});

arena.registerShot("roll", (fx, ctx) => {
  arena.spawnFx("fx-archer-roll", ctx.x, ctx.y, ctx.kind);
});
