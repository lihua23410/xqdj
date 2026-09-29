(function () {
  window.charcoalMarks = window.charcoalMarks || [];

  function clearDots(owner) {
    document.querySelectorAll(".coal-dot").forEach((el) => {
      if (el.dataset.owner === String(owner)) el.remove();
    });
  }

  arena.registerShot("coals", (fx) => {
    if (!fx) return;
    const owner = fx.unitId;
    window.charcoalMarks = (window.charcoalMarks || []).filter((m) => m.owner !== owner);
    clearDots(owner);
  });

  arena.registerShot("hand", (fx, ctx) => {
    if (!fx || !ctx || !arena.screenPos || !arena.spawnFx) return;
    const [x1, y1] = arena.screenPos(fx.vx || 0, fx.vy || 0, ctx.scale, ctx.cx, ctx.cy);
    const el = arena.spawnFx("coal-hand", ctx.x, ctx.y, fx.kind);
    if (!el) return;
    const x0 = ctx.x;
    const y0 = ctx.y;
    const toId = fx.slot;
    const born = performance.now();
    const dur = 420;
    function frame(now) {
      if (!el.isConnected) return;
      const t = Math.min(1, (now - born) / dur);
      let tx = x1;
      let ty = y1;
      const host = document.getElementById(`u-${toId}`);
      if (host) {
        const hx = parseFloat(host.style.left);
        const hy = parseFloat(host.style.top);
        if (Number.isFinite(hx) && Number.isFinite(hy)) {
          tx = hx;
          ty = hy;
        }
      }
      const e = 1 - (1 - t) ** 3;
      el.style.left = `${x0 + (tx - x0) * e}px`;
      el.style.top = `${y0 + (ty - y0) * e}px`;
      const s = 0.9 + 0.28 * Math.sin(Math.PI * t);
      el.style.transform = `translate(-50%, -50%) scale(${s})`;
      el.style.opacity = t > 0.84 ? String(1 - (t - 0.84) / 0.16) : "1";
      if (t < 1) requestAnimationFrame(frame);
      else el.remove();
    }
    requestAnimationFrame(frame);
  });

  arena.registerShot("coal", (fx) => {
    if (!fx || fx.unitId == null) return;
    const holderId = fx.slot;
    const host = document.getElementById(`u-${holderId}`);
    window.charcoalMarks.push({
      owner: fx.unitId,
      holder: holderId,
      white: (fx.amount || 0) >= 0.5,
    });
    if (!host) return;
    const n = Math.max(1, fx.vy || 1);
    const i = fx.vx || 0;
    const ang = ((i + 0.5) / n) * Math.PI * 2;
    const d = parseFloat(host.style.width) || 0;
    const rr = Math.max(4, d * 0.31);
    const dot = document.createElement("div");
    dot.className = "coal-dot" + ((fx.amount || 0) >= 0.5 ? " white" : "");
    dot.dataset.owner = String(fx.unitId);
    const px = Math.max(4, 7 * ((d || 36) / 36));
    dot.style.width = `${px}px`;
    dot.style.height = `${px}px`;
    dot.style.left = `calc(50% + ${Math.cos(ang) * rr}px)`;
    dot.style.top = `calc(50% + ${Math.sin(ang) * rr}px)`;
    host.appendChild(dot);
  });
})();
