// 电弧与预警共用后端给出的世界坐标。命中火花落在实际接触的位置。
(function () {
  const lastStrike = new Map();

  function impact(x, y, kind) {
    arena.spawnFx("fx-static-flash", x, y, kind);
    for (let i = 0; i < 6; i++) {
      const angle = (Math.PI * 2 * i) / 6;
      const distance = 14 + 3 * (i % 3);
      arena.spawnFx("fx-static-spark", x, y, kind, {
        "--dx": `${Math.cos(angle) * distance}px`,
        "--dy": `${Math.sin(angle) * distance}px`,
      });
    }
  }

  function drawBolt(fx, ctx) {
    const key = `${fx.name}:${fx.unitId}`;
    if (lastStrike.get(key) === ctx.time) return;
    lastStrike.set(key, ctx.time);

    const [x1, y1] = arena.screenPos(fx.x, fx.y, ctx.scale, ctx.cx, ctx.cy);
    const [x2, y2] = arena.screenPos(fx.vx, fx.vy, ctx.scale, ctx.cx, ctx.cy);
    const length = Math.hypot(x2 - x1, y2 - y1);
    if (length > 0.1) {
      const el = arena.spawnFx("fx-static-bolt", x1, y1, fx.kind, {
        "--static-angle": `${Math.atan2(y2 - y1, x2 - x1)}rad`,
        "--static-length": `${length}px`,
        "--static-height": `${Math.max(4, 10 * ctx.scale)}px`,
      });
      if (el) {
        el.classList.toggle("static-relay", fx.name === "static-relay");
        el.innerHTML = `<svg viewBox="0 0 100 12" preserveAspectRatio="none" aria-hidden="true">
          <polyline class="static-bolt-glow" points="0,6 14,2 28,9 43,4 61,10 77,3 91,8 100,6"/>
          <polyline class="static-bolt-core" points="0,6 14,2 28,9 43,4 61,10 77,3 91,8 100,6"/>
        </svg>`;
      }
    }
    if (fx.amount > 0) {
      const [hitX, hitY] = arena.screenPos(fx.data?.x ?? fx.vx, fx.data?.y ?? fx.vy, ctx.scale, ctx.cx, ctx.cy);
      impact(hitX, hitY, fx.kind);
    }
  }

  // 预警由常驻皮肤随对局时钟绘制。
  arena.registerShot("static-warn", () => {});
  arena.registerShot("static-cancel", () => {});
  arena.registerShot("static-bolt", drawBolt);
  arena.registerShot("static-relay", drawBolt);
})();
