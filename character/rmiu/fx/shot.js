(function () {
  arena.registerShot("scream", (fx, ctx) => {
    const el = arena.spawnFx("fx-miu-scream", ctx.x, ctx.y - 22, fx.kind);
    if (!el) return;
    el.textContent = "我才是缪……你们都不是……";
    el.style.color = "#ff1a1a";
  });
})();
