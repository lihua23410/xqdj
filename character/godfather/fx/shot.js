(function () {
  window.godfatherAim = window.godfatherAim || {};

  arena.registerShot("aim", (fx, ctx) => {
    if (!fx || fx.unitId == null) return;
    if ((fx.amount || 0) < 0) {
      window.godfatherAim[fx.unitId] = {
        ...window.godfatherAim[fx.unitId], p: -1, leaving: true, time: ctx?.time || 0,
      };
      return;
    }
    window.godfatherAim[fx.unitId] = {
      tx: fx.vx, ty: fx.vy, p: fx.amount || 0, leaving: false, time: ctx?.time || 0,
    };
  });

  arena.registerShot("scream", (fx, ctx) => {
    const el = arena.spawnFx("fx-dealer-scream", ctx.x, ctx.y - 18, fx.kind);
    if (el) el.textContent = "这东西我只卖不碰的";
  });

  arena.registerShot("leave", (fx) => {
    if (!fx || fx.unitId == null) return;
    delete window.godfatherAim[fx.unitId];
    const el = document.getElementById(`u-${fx.unitId}`);
    if (el) el.classList.add("minion-leave");
  });
})();
