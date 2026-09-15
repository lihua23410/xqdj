(function () {
  window.radarBeam = window.radarBeam || {};

  arena.registerShot("beam", (fx) => {
    if (!fx || fx.unitId == null) return;
    window.radarBeam[fx.unitId] = {
      dx: fx.vx || 0,
      dy: fx.vy || 0,
      len: fx.amount || 560,
    };
  });
})();
