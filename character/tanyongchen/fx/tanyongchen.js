// 谭泳辰老先生前端：青色小球 + 狐狸头饰，场上的铃铛/苦无，灵狐场域与净化特效。
window.tanSt = window.tanSt || {};

(function () {
  function baseOf(kind) {
    return (window.arena && arena.lookOf && arena.lookOf(kind).base) || "/ball/谭泳辰老先生";
  }

  function ensureImg(el, cls, src) {
    let img = el.querySelector(`:scope > .${cls}`);
    if (!img) {
      img = document.createElement("img");
      img.className = cls;
      img.alt = "";
      el.appendChild(img);
    }
    if (src && img.getAttribute("src") !== src) img.src = src;
    return img;
  }

  window.lookFX["tanyongchen"] = {
    unmount(el) {
      el?.querySelector(":scope > .tan-fox")?.remove();
    },
    tick(el, u) {
      if (!el || !el.classList.contains("look-tanyongchen")) return;
      ensureImg(el, "tan-fox", `${baseOf(u.kind)}/fx/狐狸头饰.png`);
    },
  };

  window.lookFX["tan-bell"] = {
    unmount(el) {
      el?.querySelector(":scope > .tan-item")?.remove();
    },
    tick(el, u) {
      ensureImg(el, "tan-item", `${baseOf(u.kind)}/fx/铃铛.png`);
    },
  };

  window.lookFX["tan-kunai"] = {
    unmount(el) {
      el?.querySelector(":scope > .tan-item")?.remove();
    },
    tick(el, u) {
      ensureImg(el, "tan-item", `${baseOf(u.kind)}/fx/苦无.png`);
    },
  };

  window.lookFX["tan-domain"] = {
    unmount(el) {
      el?.querySelector(":scope > .tan-domain-img")?.remove();
    },
    tick(el, u) {
      ensureImg(el, "tan-domain-img", `${baseOf(u.kind)}/fx/灵狐场域.png`);
    },
  };

  window.lookFX["tan-milk"] = {
    unmount(el) {
      el?.querySelector(":scope > .tan-milk-img")?.remove();
    },
    tick(el, u) {
      ensureImg(el, "tan-milk-img", `${baseOf(u.kind)}/fx/奶符.png`);
    },
  };

  function fxRoot() {
    return document.getElementById("fx");
  }

  function spawnFxImg(ctx, fx, imgName, worldSize, spin) {
    const root = fxRoot();
    if (!root) return;
    const scale = ctx.scale || 1;
    const size = Math.max(40, worldSize * scale);
    const el = document.createElement("div");
    el.className = "fx tan-fx";
    el.style.left = `${ctx.x}px`;
    el.style.top = `${ctx.y}px`;
    el.style.setProperty("--size", `${size}px`);
    const img = document.createElement("img");
    img.className = `tan-fx-img${spin ? " spin" : ""}`;
    img.alt = "";
    img.src = `${baseOf(fx.kind)}/fx/${imgName}`;
    el.appendChild(img);
    root.appendChild(el);
    img.addEventListener("animationend", () => el.remove());
  }

  function dirAng(fx) {
    const vx = Number(fx && fx.vx) || 0;
    const vy = Number(fx && fx.vy) || 0;
    if (Math.hypot(vx, vy) < 1e-6) return 0;
    return Math.atan2(-vy, vx);
  }

  // 苦无：沿方向飞出一枚，暴击时额外闪光。
  arena.registerShot("kunai", (fx, ctx) => {
    const root = fxRoot();
    if (!root) return;
    const scale = ctx.scale || 1;
    const el = document.createElement("div");
    el.className = "fx tan-shot";
    el.style.left = `${ctx.x}px`;
    el.style.top = `${ctx.y}px`;
    el.style.setProperty("--ang", `${dirAng(fx)}rad`);
    // Amount 是苦无要飞的实数距离（世界单位）。
    el.style.setProperty("--dist", `${Math.max(40, Number(fx.amount) || 400) * scale}px`);
    const img = document.createElement("img");
    img.className = "tan-shot-img";
    img.alt = "";
    img.src = `${baseOf(fx.kind)}/fx/苦无.png`;
    el.appendChild(img);
    root.appendChild(el);
    img.addEventListener("animationend", () => el.remove());
  });

  // 苦无飞到落点命中：伤害在这一拍才结算，暴击有更大的爆发。
  arena.registerShot("kunai-hit", (fx, ctx) => {
    arena.spawnFx("fx-flash", ctx.x, ctx.y, fx.kind);
    arena.burst(ctx.x, ctx.y, fx.kind, fx.amount > 0 ? 16 : 8);
    if (fx.amount > 0) {
      arena.spawnFx("fx-ring", ctx.x, ctx.y, fx.kind);
      spawnFxImg(ctx, fx, "苦无.png", 110, true);
    }
  });

  arena.registerShot("purify", (fx, ctx) => {
    spawnFxImg(ctx, fx, "净化.png", 280, true);
    arena.spawnFx("fx-flash", ctx.x, ctx.y, fx.kind);
    arena.spawnFx("fx-ring", ctx.x, ctx.y, fx.kind);
    arena.burst(ctx.x, ctx.y, fx.kind, 24);
  });

  arena.registerShot("pickup", (fx, ctx) => {
    spawnFxImg(ctx, fx, fx.amount === 2 ? "苦无.png" : "铃铛.png", 96, false);
    arena.spawnFx("fx-flash", ctx.x, ctx.y, fx.kind);
    arena.burst(ctx.x, ctx.y, fx.kind, 10);
  });

  arena.registerShot("domain", (fx, ctx) => {
    spawnFxImg(ctx, fx, "灵狐场域.png", 360, true);
    arena.spawnFx("fx-shock", ctx.x, ctx.y, fx.kind);
    arena.burst(ctx.x, ctx.y, fx.kind, 20);
  });

  arena.registerShot("milk", (fx, ctx) => {
    spawnFxImg(ctx, fx, "奶符.png", 90, false);
    arena.spawnFx("fx-dmg heal", ctx.x, ctx.y - 12, fx.kind);
    arena.burst(ctx.x, ctx.y, fx.kind, 10);
  });
})();
