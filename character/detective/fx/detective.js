(function () {
  const arena = window.arena;
  arena.registerShot("judge", (fx, ctx) => {
    if (ctx && ctx.audioOnly) return;
    const x = ctx ? ctx.x : 0;
    const y = ctx ? ctx.y : 0;
    arena.spawnFx("fx-judge", x, y, fx && fx.kind);
  });
  arena.registerShot("detective-shot", (fx, ctx) => {
    if (ctx && ctx.audioOnly) return;
    arena.spawnFx("fx-detective-shot", ctx.x, ctx.y, fx && fx.kind);
  });
  // 不叫 scream：那个名字在直播里只走声音，画面那一遍会被跳过。
  const verdicts = ["无罪", "有罪"];
  const verdictTone = ["#f4efe4", "#f3d2a2"];
  arena.registerShot("verdict", (fx, ctx) => {
    if (ctx && ctx.audioOnly) return;
    const n = Math.max(0, Math.min(verdicts.length - 1, Math.round((fx && fx.amount) || 0)));
    const x = ctx ? ctx.x : 0;
    const y = ctx ? ctx.y : 0;
    const el = arena.spawnFx("fx-verdict", x, y, fx && fx.kind);
    if (!el) return;
    el.textContent = verdicts[n];
    el.dataset.verdict = String(n);
    el.style.color = verdictTone[n];
  });
})();
