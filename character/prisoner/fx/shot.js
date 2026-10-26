(function () {
  window.prisonerChain = window.prisonerChain || {};
  window.prisonerDrops = window.prisonerDrops || {};
  window.prisonerShock = window.prisonerShock || {};

  const CHAIN_LEN = 350;
  const CHAIN_TAUT = 1.5;
  const SHOCK_MS = 5000;

  function screenOf(ctx, x, y) {
    return arena.screenPos(x, y, ctx.scale, ctx.cx, ctx.cy);
  }

  function clank(x, y) {
    arena.spawnFx("fx-chain-clank", x, y, "囚徒钩爪");
    arena.burst(x, y, "囚徒钩爪", 4);
  }

  function strain(ctx, fx) {
    const [x1, y1] = screenOf(ctx, fx.x, fx.y);
    const [x2, y2] = screenOf(ctx, fx.vx, fx.vy);
    const dx = x2 - x1;
    const dy = y2 - y1;
    arena.spawnFx("fx-chain-strain", (x1 + x2) / 2, (y1 + y2) / 2, "囚徒钩爪", {
      "--len": `${Math.max(20, Math.hypot(dx, dy))}px`,
      "--ang": `${Math.atan2(dy, dx)}rad`,
    });
  }

  arena.registerShot("chain", (fx, ctx) => {
    if (!fx || fx.unitId == null) return;
    const prev = window.prisonerChain[fx.unitId];
    const len = Number(fx.amount) || Math.hypot(fx.x - fx.vx, fx.y - fx.vy);
    const taut = len >= CHAIN_LEN - CHAIN_TAUT;
    const moved = !prev || Math.hypot(prev.hx - fx.vx, prev.hy - fx.vy) > 0.5;
    if (ctx && arena.spawnFx) {
      if (moved) {
        // 换钩爪：新锚点落钉。
        const [ax, ay] = screenOf(ctx, fx.vx, fx.vy);
        clank(ax, ay);
      } else if (!prev.taut && taut) {
        // 松弛 → 绷直：沿弦紧一下，锚点再响一声。
        strain(ctx, fx);
        const [ax, ay] = screenOf(ctx, fx.vx, fx.vy);
        clank(ax, ay);
      }
    }
    window.prisonerChain[fx.unitId] = {
      hx: fx.vx,
      hy: fx.vy,
      len,
      taut,
      seenAt: performance.now(),
    };
  });

  arena.registerShot("drop", (fx) => {
    if (!fx) return;
    const key = `${fx.unitId || 0}:${fx.kind}`;
    window.prisonerDrops[key] = { x: fx.x, y: fx.y, kind: fx.kind, until: performance.now() + 120 };
  });

  arena.registerShot("bolt", (fx, ctx) => {
    if (!fx || !ctx) return;
    const [x2, y2] = arena.screenPos(fx.vx, fx.vy, ctx.scale, ctx.cx, ctx.cy);
    const dx = x2 - ctx.x;
    const dy = y2 - ctx.y;
    const len = Math.hypot(dx, dy);
    arena.spawnFx("fx-bolt", (ctx.x + x2) / 2, (ctx.y + y2) / 2, fx.kind, {
      "--len": `${Math.max(8, len)}px`,
      "--ang": `${Math.atan2(dy, dx)}rad`,
    });
  });

  arena.registerShot("scream", (fx, ctx) => {
    // 触电持续 5 秒（Go 的 shockSec）；这里让锁链和球一起亮 5 秒，
    // 而不是只在挨电的那一帧闪一下。
    if (fx && fx.unitId != null) {
      window.prisonerShock[fx.unitId] = performance.now() + SHOCK_MS;
    }
    const el = arena.spawnFx("fx-scream", ctx.x, ctx.y - 18, fx.kind);
    if (el) el.textContent = "啊";
    arena.spawnFx("fx-flash", ctx.x, ctx.y, fx.kind);
  });
})();
