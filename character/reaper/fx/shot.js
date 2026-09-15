(function () {
  window.sickleBlood = window.sickleBlood || {};

  arena.registerShot("blood", (fx) => {
    if (!fx || fx.unitId == null) return;
    window.sickleBlood[fx.unitId] = (fx.amount || 0) > 0;
  });
})();
