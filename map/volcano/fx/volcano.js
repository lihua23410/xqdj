(function () {
  window.lookFX = window.lookFX || {};
  window.shotFX = window.shotFX || {};
  window.volcanoWarn = window.volcanoWarn || {};
  window.volcanoEruptUntil = 0;
  window.volcanoLobStarted = window.volcanoLobStarted || {};
  window.volcanoLobDone = window.volcanoLobDone || {};
  window.volcanoLavaBorn = window.volcanoLavaBorn || {};

  // 硬挂本包 shot：不依赖 currentScript / dataset.pack（中文路径下曾出现钩子没挂上）。
  // 整文件 IIFE：避免与 app.js 顶层 const fxRoot 等标识符冲突导致脚本静默失败。
  const VOLCANO_BASE = "/ball/火山";
  const LOB_MS = 550;
  window.shotFX[VOLCANO_BASE] = window.shotFX[VOLCANO_BASE] || {};

  function volcanoFxLayer() {
    return document.getElementById("fx");
  }

  function regShot(name, fn) {
    window.shotFX[VOLCANO_BASE][name] = fn;
  }

  function ensureRim(el) {
    if (!el || el.querySelector(":scope > .volcano-rim")) return;
    const rim = document.createElement("i");
    rim.className = "volcano-rim";
    el.appendChild(rim);
  }

  function emitSprayBits(spray, n) {
    for (let i = 0; i < n; i++) {
      const bit = document.createElement("i");
      // 上向锥形喷溅，略带左右散开
      const ang = -Math.PI / 2 + (Math.random() - 0.5) * 1.35;
      const dist = 36 + Math.random() * 78;
      const size = 8 + Math.random() * 10;
      bit.style.setProperty("--dx", `${Math.cos(ang) * dist}px`);
      bit.style.setProperty("--dy", `${Math.sin(ang) * dist - 12 - Math.random() * 28}px`);
      bit.style.setProperty("--spin", `${(Math.random() - 0.5) * 180}deg`);
      bit.style.setProperty("--sz", `${size}px`);
      bit.style.animationDuration = `${0.5 + Math.random() * 0.4}s`;
      if (Math.random() < 0.4) bit.classList.add("is-ember");
      bit.addEventListener("animationend", () => bit.remove());
      spray.appendChild(bit);
    }
  }

  function syncEruptDecor(el) {
    const effects = (typeof state !== "undefined" && state && state.effects) || [];
    // 双通道：shot 钩子续命 + 直接读本帧 effects（防钩子未挂时完全无特效）
    if (effects.some((e) => e && (e.name === "erupt" || e.name === "warn"))) {
      window.volcanoEruptUntil = performance.now() + 220;
    }
    const now = performance.now();
    const on = now < (window.volcanoEruptUntil || 0);
    el.classList.toggle("is-erupt", on);
    if (!on) {
      el.querySelector(":scope > .volcano-spray")?.remove();
      return;
    }
    let spray = el.querySelector(":scope > .volcano-spray");
    if (!spray) {
      spray = document.createElement("span");
      spray.className = "volcano-spray";
      spray.dataset.next = "0";
      el.appendChild(spray);
    }
    // 喷发期间持续吐粒子，替代原先伸缩长方柱
    if (now >= Number(spray.dataset.next || 0)) {
      spray.dataset.next = String(now + 45);
      emitSprayBits(spray, 5 + (Math.random() < 0.5 ? 2 : 0));
    }
  }

  function placeWarn(fx, ctx) {
    const root = volcanoFxLayer();
    if (!root || !ctx) return;
    const key = `${Math.round(fx.x || 0)}:${Math.round(fx.y || 0)}`;
    const r = Math.max(10, (fx.amount || 22) * (ctx.scale || 1));
    let el = window.volcanoWarn[key];
    if (!el || !el.isConnected) {
      el = document.createElement("div");
      el.className = "fx fx-volcano-warn";
      root.appendChild(el);
      window.volcanoWarn[key] = el;
    }
    el.style.left = `${ctx.x}px`;
    el.style.top = `${ctx.y}px`;
    el.style.width = `${r * 2}px`;
    el.style.height = `${r * 2}px`;
    el.dataset.until = String(performance.now() + 200);
  }

  function startLob(fx, ctx) {
    const root = volcanoFxLayer();
    if (!root || !ctx) return;
    const key = `${Math.round(fx.x || 0)}:${Math.round(fx.y || 0)}`;
    if (window.volcanoLobStarted[key] && performance.now() < window.volcanoLobStarted[key]) {
      return;
    }
    window.volcanoLobStarted[key] = performance.now() + LOB_MS + 80;
    window.volcanoLobDone[key] = performance.now() + LOB_MS + 400;

    const warn = window.volcanoWarn[key];
    if (warn) {
      warn.remove();
      delete window.volcanoWarn[key];
    }

    const scale = ctx.scale || 1;
    const r = Math.max(8, (fx.amount || 22) * scale * 0.85);
    const x0 = ctx.cx;
    const y0 = ctx.cy;
    const x1 = ctx.x;
    const y1 = ctx.y;
    const dist = Math.hypot(x1 - x0, y1 - y0);
    const xm = (x0 + x1) / 2;
    const ym = (y0 + y1) / 2 - Math.max(40, dist * 0.45);

    const el = document.createElement("div");
    el.className = "fx fx-volcano-lob";
    el.style.width = `${r * 2}px`;
    el.style.height = `${r * 2}px`;
    root.appendChild(el);

    const t0 = performance.now();
    function frame(now) {
      const u = Math.min(1, (now - t0) / LOB_MS);
      const omu = 1 - u;
      const x = omu * omu * x0 + 2 * omu * u * xm + u * u * x1;
      const y = omu * omu * y0 + 2 * omu * u * ym + u * u * y1;
      const squash = 0.85 + 0.25 * Math.sin(u * Math.PI);
      el.style.left = `${x}px`;
      el.style.top = `${y}px`;
      el.style.transform = `translate(-50%, -50%) scale(${squash}) rotate(${u * 220}deg)`;
      if (u < 1) {
        requestAnimationFrame(frame);
        return;
      }
      el.remove();
    }
    requestAnimationFrame(frame);
  }

  function placeLand(fx, ctx) {
    const root = volcanoFxLayer();
    if (!root || !ctx) return;
    const key = `${Math.round(fx.x || 0)}:${Math.round(fx.y || 0)}`;
    const warn = window.volcanoWarn[key];
    if (warn) {
      warn.remove();
      delete window.volcanoWarn[key];
    }
    const r = Math.max(10, (fx.amount || 22) * (ctx.scale || 1));
    const el = document.createElement("div");
    el.className = "fx fx-volcano-land";
    el.style.left = `${ctx.x}px`;
    el.style.top = `${ctx.y}px`;
    for (let i = 0; i < 8; i++) {
      const bit = document.createElement("i");
      const ang = (Math.PI * 2 * i) / 8;
      const dist = r * (0.55 + (i % 2) * 0.4);
      bit.style.setProperty("--dx", `${Math.cos(ang) * dist}px`);
      bit.style.setProperty("--dy", `${Math.sin(ang) * dist}px`);
      el.appendChild(bit);
    }
    root.appendChild(el);
    setTimeout(() => el.remove(), 420);
  }

  regShot("erupt", () => {
    window.volcanoEruptUntil = performance.now() + 220;
  });
  regShot("warn", placeWarn);
  regShot("lob", startLob);
  regShot("land", placeLand);

  window.lookFX["volcano-crater"] = {
    tick(el, u, ctx) {
      if (!el || !el.classList.contains("look-volcano-crater")) return;
      ensureRim(el);
      syncEruptDecor(el);
      // 钩子未挂时，从 effects 补播 warn / lob
      const effects = (typeof state !== "undefined" && state && state.effects) || [];
      for (const fx of effects) {
        if (!fx || !ctx) continue;
        if (fx.name === "warn") {
          const [x, y] = arena.screenPos(fx.x, fx.y, ctx.scale, ctx.cx, ctx.cy);
          placeWarn(fx, { ...ctx, x, y });
        } else if (fx.name === "lob") {
          const [x, y] = arena.screenPos(fx.x, fx.y, ctx.scale, ctx.cx, ctx.cy);
          startLob(fx, { ...ctx, x, y });
        }
      }
    },
    unmount(el) {
      el?.classList.remove("is-erupt");
      el?.querySelector(":scope > .volcano-rim")?.remove();
      el?.querySelector(":scope > .volcano-spray")?.remove();
    },
  };

  window.lookFX["volcano-lava"] = {
    tick(el, u, ctx) {
      if (!el || !el.classList.contains("look-volcano-lava")) return;
      if (u && u.id != null && !window.volcanoLavaBorn[u.id]) {
        window.volcanoLavaBorn[u.id] = true;
        const key = `${Math.round(u.x || 0)}:${Math.round(u.y || 0)}`;
        // lob 已播过则直接显示；整段 lob 丢失才补抛
        if (window.volcanoLobDone[key] && performance.now() < window.volcanoLobDone[key]) {
          delete window.volcanoLobDone[key];
          return;
        }
        el.classList.add("is-lobbing");
        const [x, y] = arena.screenPos(u.x, u.y, ctx.scale, ctx.cx, ctx.cy);
        startLob({ x: u.x, y: u.y, amount: u.radius || 22 }, { ...ctx, x, y });
        setTimeout(() => el.classList.remove("is-lobbing"), LOB_MS);
      }
    },
    unmount(el, u) {
      if (u && u.id != null) delete window.volcanoLavaBorn[u.id];
    },
  };

  (function volcanoWarnGC() {
    const now = performance.now();
    for (const [key, el] of Object.entries(window.volcanoWarn)) {
      if (!el || !el.isConnected || now > Number(el.dataset.until || 0)) {
        el?.remove();
        delete window.volcanoWarn[key];
      }
    }
    requestAnimationFrame(volcanoWarnGC);
  })();
})();
