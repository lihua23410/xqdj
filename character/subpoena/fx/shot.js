// 传票员的一次性特效：票面数字、故障气泡。
//
// 注意：故障不能叫 "scream" 或 "shot"。这两个名字在前端属于「声音即时通道」
// （app.js 的 IMMEDIATE_AUDIO / shot 分支），派发时只给一个没有坐标的
// audioOnly 上下文，画面那一遍会直接跳过——拿来画气泡会画不出来。
window.subpoenaDmg = window.subpoenaDmg || {};

(function () {
  // 表里只该留下场上的票。超过这个量说明攒了不少作废的 id，整表清一次；
  // 票每帧都会重发票面，清掉最坏只会让数字空一帧。
  const CAP = 512;

  arena.registerShot("ticket", (fx) => {
    if (!fx || fx.unitId == null) return;
    const table = window.subpoenaDmg;
    if (!table) return;
    if (Object.keys(table).length > CAP) {
      for (const k in table) delete table[k];
    }
    table[fx.unitId] = fx.amount || 0;
  });

  arena.registerShot("fault", (fx, ctx) => {
    if (!ctx || ctx.audioOnly) return;
    // 贴着场边挨打时气泡会被 #fx 的 overflow:hidden 切掉：夹进球场宽度里再画。
    const host = document.getElementById("fx");
    const wide = host ? host.clientWidth : 0;
    const x = wide > 160 ? Math.min(Math.max(ctx.x, 64), wide - 64) : ctx.x;
    const el = arena.spawnFx("fx-subpoena-fault", x, Math.max(28, ctx.y - 40), fx.kind);
    if (el) el.textContent = "发生了一些故障";
  });
})();
