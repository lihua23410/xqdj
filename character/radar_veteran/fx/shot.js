(function () {
  window.radarVetBeam = window.radarVetBeam || {};
  window.radarVetMark = window.radarVetMark || {};
  window.radarVetLock = window.radarVetLock || {};
  window.radarVetGun = window.radarVetGun || {};

  arena.registerShot("beam", (fx) => {
    if (!fx || fx.unitId == null) return;
    if (fx.kind !== "雷达(百战)") return;
    if ((fx.amount || 0) <= 0) {
      window.radarVetBeam[fx.unitId] = { off: true };
      return;
    }
    window.radarVetBeam[fx.unitId] = {
      dx: fx.vx || 0,
      dy: fx.vy || 0,
      len: fx.amount || 0,
    };
  });

  arena.registerShot("name", (fx) => {
    if (!fx || fx.unitId == null) return;
    if (fx.kind !== "雷达(百战)") return;
    if ((fx.amount || 0) <= 0) {
      window.radarVetLock[fx.unitId] = { off: true };
      return;
    }
    window.radarVetLock[fx.unitId] = {
      x: fx.x || 0,
      y: fx.y || 0,
      r: fx.amount || 18,
    };
  });

  arena.registerShot("gun", (fx) => {
    if (!fx || fx.unitId == null) return;
    if (fx.kind !== "雷达(百战)") return;
    if ((fx.amount || 0) < 0) {
      window.radarVetGun[fx.unitId] = { off: true };
      return;
    }
    window.radarVetGun[fx.unitId] = { grow: fx.amount || 0 };
  });

  arena.registerShot("mark", (fx) => {
    if (!fx || fx.unitId == null) return;
    const prev = window.radarVetMark[fx.unitId];
    if (prev && prev.born) return;
    window.radarVetMark[fx.unitId] = {
      born: performance.now(),
      dur: Math.max(120, (fx.vy || 0.28) * 1000),
      r: fx.amount || 4,
    };
  });

  arena.registerShot("launch", (fx, ctx) => {
    if (!fx || !ctx) return;
    if (fx.kind !== "雷达(百战)") return;
    const big = (fx.amount || 0) > 0;
    const tone = big ? "" : " sm";
    const vars = {
      "--delay": `${Math.max(0, fx.vy || 0)}s`,
      "--dx": `${fx.vx || 0}px`,
    };
    arena.spawnFx(`fx-radar-vet-muzzle${tone}`, ctx.x, ctx.y, fx.kind, vars);
    arena.spawnFx(`fx-radar-vet-column${tone}`, ctx.x, ctx.y, fx.kind, vars);
    arena.spawnFx(`fx-radar-vet-launch${tone}`, ctx.x, ctx.y, fx.kind, vars);
    window.radarVetGunKick = window.radarVetGunKick || {};
    if (fx.unitId != null) window.radarVetGunKick[fx.unitId] = performance.now();
  });

  arena.registerShot("blast", (fx, ctx) => {
    if (!fx) return;
    const play = (cls, r, extra) => {
      const el = arena.spawnFx(cls, ctx.x, ctx.y, fx.kind, { "--r": `${r}px`, ...(extra || {}) });
      if (!el) return;
      el.style.animation = "none";
      void el.offsetWidth;
      el.style.animation = "";
    };
    const spin = `${Math.floor(Math.random() * 360)}deg`;
    const shards = (n, reach) => {
      for (let i = 0; i < n; i++) {
        const ang = Math.random() * Math.PI * 2;
        const dist = reach * (0.28 + Math.random() * 0.62);
        const el = arena.spawnFx("fx-radar-vet-shard", ctx.x, ctx.y, fx.kind, {
          "--dx": `${Math.cos(ang) * dist}px`,
          "--dy": `${Math.sin(ang) * dist}px`,
          "--rot": `${Math.floor(Math.random() * 220 - 40)}deg`,
        });
        if (!el) continue;
        const w = 4 + Math.random() * 8;
        const h = 3 + Math.random() * 4;
        el.style.width = `${w}px`;
        el.style.height = `${h}px`;
        el.style.margin = `${-h / 2}px 0 0 ${-w / 2}px`;
        if (Math.random() < 0.45) el.style.background = "#9fe7e7";
      }
    };
    if (fx.kind === "大炮击") {
      const r = Math.max(48, (fx.amount || 80) * (ctx.scale || 1) * 2);
      play("fx-radar-vet-blast", r, { "--spin": spin });
      play("fx-radar-vet-blast-ring", r);
      shards(9, r * 0.55);
      arena.burst(ctx.x, ctx.y, fx.kind, 28);
      return;
    }
    if (fx.kind === "小炮击") {
      const r = Math.max(28, (fx.amount || 55) * (ctx.scale || 1) * 2);
      play("fx-radar-vet-blast-sm", r, { "--spin": spin });
      shards(5, r * 0.5);
      arena.burst(ctx.x, ctx.y, fx.kind, 12);
    }
  });
})();
