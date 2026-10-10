(function () {
  window.lookFX = window.lookFX || {};
  window.shotFX = window.shotFX || {};

  const CHAMBER_BASE = "/ball/机关房";
  window.shotFX[CHAMBER_BASE] = window.shotFX[CHAMBER_BASE] || {};

  function layer() {
    return document.getElementById("fx");
  }

  function regShot(name, fn) {
    window.shotFX[CHAMBER_BASE][name] = fn;
  }

  function placeRect(cls, fx, ctx, untilPad) {
    const root = layer();
    if (!root || !ctx || typeof arena === "undefined") return;
    const [x0, y0] = arena.screenPos(fx.x, fx.y, ctx.scale, ctx.cx, ctx.cy);
    const [x1, y1] = arena.screenPos(fx.vx, fx.vy, ctx.scale, ctx.cx, ctx.cy);
    const left = Math.min(x0, x1);
    const top = Math.min(y0, y1);
    const w = Math.abs(x1 - x0);
    const h = Math.abs(y1 - y0);
    const key = `${cls}:${Math.round(fx.x)}:${Math.round(fx.y)}:${Math.round(fx.vx)}:${Math.round(fx.vy)}`;
    let el = root.querySelector(`[data-chamber-key="${key}"]`);
    if (!el) {
      el = document.createElement("div");
      el.className = `fx ${cls}`;
      el.dataset.chamberKey = key;
      root.appendChild(el);
    }
    el.style.left = `${left}px`;
    el.style.top = `${top}px`;
    el.style.width = `${w}px`;
    el.style.height = `${h}px`;
    el.dataset.until = String(performance.now() + (untilPad || 200));
  }

  function placeGrid(fx, ctx) {
    const root = layer();
    if (!root || !ctx || typeof arena === "undefined") return;
    const half = fx.amount || 242.487;
    let el = root.querySelector(":scope > .fx-chamber-grid");
    if (!el) {
      el = document.createElement("div");
      el.className = "fx fx-chamber-grid";
      root.appendChild(el);
      for (let i = 0; i < 2; i++) {
        el.appendChild(Object.assign(document.createElement("i"), { className: "v" }));
        el.appendChild(Object.assign(document.createElement("i"), { className: "h" }));
      }
    }
    const lines = el.querySelectorAll("i");
    [-half / 3, half / 3].forEach((d, i) => {
      const [sx] = arena.screenPos(d, 0, ctx.scale, ctx.cx, ctx.cy);
      const [, sy] = arena.screenPos(0, d, ctx.scale, ctx.cx, ctx.cy);
      if (lines[i * 2]) lines[i * 2].style.left = `${sx}px`;
      if (lines[i * 2 + 1]) lines[i * 2 + 1].style.top = `${sy}px`;
    });
    el.dataset.until = String(performance.now() + 220);
  }

  regShot("grid", placeGrid);
  regShot("warn", (fx, ctx) => placeRect("fx-chamber-warn", fx, ctx, 200));
  regShot("gas", (fx, ctx) => placeRect("fx-chamber-gas", fx, ctx, 200));

  function syncFromEffects(ctx) {
    const effects = (typeof state !== "undefined" && state && state.effects) || [];
    for (const fx of effects) {
      if (!fx || fx.kind !== "机关房") continue;
      if (fx.name === "grid") placeGrid(fx, ctx);
      else if (fx.name === "warn") placeRect("fx-chamber-warn", fx, ctx, 200);
      else if (fx.name === "gas") placeRect("fx-chamber-gas", fx, ctx, 200);
    }
  }

  window.lookFX["chamber-lock"] = {
    tick(el, u, ctx) {
      if (ctx) syncFromEffects(ctx);
    },
    unmount() {},
  };

  (function chamberFXGC() {
    const root = layer();
    const now = performance.now();
    if (root) {
      for (const el of root.querySelectorAll(".fx-chamber-grid, .fx-chamber-warn, .fx-chamber-gas")) {
        if (now > Number(el.dataset.until || 0)) el.remove();
      }
    }
    requestAnimationFrame(chamberFXGC);
  })();
})();
