(function () {
  window.ningyushiHUD = window.ningyushiHUD || {};

  arena.registerShot("energy", (fx) => {
    if (!fx || fx.unitId == null) return;
    const prev = window.ningyushiHUD[fx.unitId] || {};
    window.ningyushiHUD[fx.unitId] = {
      ...prev,
      energy: fx.amount || 0,
      cap: fx.vy || 5,
    };
  });

  arena.registerShot("cards", (fx) => {
    if (!fx || fx.unitId == null) return;
    const packed = Math.round(fx.vx || 0) >>> 0;
    const n = Math.max(0, Math.min(5, Math.round(fx.vy || 0)));
    const cards = [];
    for (let i = 0; i < n; i++) {
      cards.push((packed >> (i * 4)) & 0xf);
    }
    const prev = window.ningyushiHUD[fx.unitId] || {};
    window.ningyushiHUD[fx.unitId] = {
      ...prev,
      cards,
      fill: Math.max(0, Math.min(1, fx.amount || 0)),
    };
  });

  arena.registerShot("shot", (fx, ctx) => {
    arena.spawnFx("fx-flash", ctx.x, ctx.y, fx.kind);
    const ang = Math.atan2(-(fx.vy || 0), fx.vx || 1) * (180 / Math.PI);
    const beam = arena.spawnFx("fx-beam", ctx.x, ctx.y, fx.kind);
    if (beam) beam.style.transform = `translate(0, -50%) rotate(${ang}deg)`;
  });

  arena.registerShot("break", (fx, ctx) => {
    const r = Math.max(48, (fx.amount || 108) * (ctx.scale || 1) * 2);
    arena.spawnFx("fx-ningyushi-ring", ctx.x, ctx.y, fx.kind, { "--r": `${r}px` });
    arena.spawnFx("fx-flash", ctx.x, ctx.y, fx.kind);
    arena.burst(ctx.x, ctx.y, fx.kind, 16);
  });

  arena.registerShot("range", (fx) => {
    if (!fx || fx.unitId == null) return;
    const prev = window.ningyushiHUD[fx.unitId] || {};
    window.ningyushiHUD[fx.unitId] = {
      ...prev,
      range: (fx.amount || 0) > 0,
      rangeAng: Math.atan2(-(fx.vy || 0), fx.vx || 1),
    };
  });

  arena.registerShot("fan", (fx) => {
    if (!fx || fx.unitId == null) return;
    const prev = window.ningyushiHUD[fx.unitId] || {};
    if (!(fx.amount > 0)) {
      window.ningyushiHUD[fx.unitId] = { ...prev, fan: null };
      return;
    }
    window.ningyushiHUD[fx.unitId] = {
      ...prev,
      fan: {
        x: fx.x,
        y: fx.y,
        vx: fx.vx || 1,
        vy: fx.vy || 0,
        span: fx.amount,
        reach: 36,
      },
    };
  });

  arena.registerShot("pose", (fx) => {
    if (!fx || fx.unitId == null) return;
    const prev = window.ningyushiHUD[fx.unitId] || {};
    window.ningyushiHUD[fx.unitId] = {
      ...prev,
      pose: Math.max(0, Math.round(fx.amount || 0)),
    };
  });

  arena.registerShot("laser", (fx) => {
    if (!fx || fx.unitId == null) return;
    const prev = window.ningyushiHUD[fx.unitId] || {};
    if (!(fx.amount > 0)) {
      window.ningyushiHUD[fx.unitId] = { ...prev, laser: [] };
      return;
    }
    const laser = [...(prev.laser || []), { x1: fx.x, y1: fx.y, x2: fx.vx, y2: fx.vy }];
    window.ningyushiHUD[fx.unitId] = { ...prev, laser };
  });

  arena.registerShot("blast", (fx, ctx) => {
    const r = Math.max(48, (fx.amount || 54) * (ctx.scale || 1) * 2);
    const size = { "--r": `${r}px` };
    arena.spawnFx("fx-ningyushi-blast-core", ctx.x, ctx.y, fx.kind);
    arena.spawnFx("fx-ningyushi-blast-wave", ctx.x, ctx.y, fx.kind, size);
    arena.spawnFx("fx-flash", ctx.x, ctx.y, fx.kind);
    arena.burst(ctx.x, ctx.y, fx.kind, 18);
  });

  const CALLS = {
    5: ["「人偶振起」", "「导引线」"],
    6: ["「人偶操创」", "「人偶归巢」"],
    7: ["「人偶千枪」", "「人偶千枪」"],
    8: ["「大江户炸药机关人偶」", "「大江户炸药机关人偶」"],
    9: ["魔符「Artful Sacrifice」", "魔符「Artful Sacrifice」"],
    10: ["诅咒「蓬莱人偶」", "诅咒「蓬莱人偶」"],
    11: ["战符「Little Legion」", "战符「Little Legion」"],
  };

  arena.registerShot("call", (fx, ctx) => {
    const sk = Math.round(fx.amount || 0);
    const pair = CALLS[sk];
    if (!pair) return;
    const spell = (fx.vy || 0) > 0;
    const el = arena.spawnFx(spell ? "fx-ningyushi-call is-spell" : "fx-ningyushi-call", ctx.x, ctx.y - 22, fx.kind);
    if (el) el.textContent = spell ? pair[1] : pair[0];
  });
})();
