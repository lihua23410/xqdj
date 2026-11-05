// 老虎机前端：像素机身覆盖球体，三个转轮按服务端结果播放 5 秒抽奖动画。
window.slotSt = window.slotSt || {};
window.lookFX = window.lookFX || {};

(function () {
  const SYMBOL_FILES = ["皇冠.png", "幸运7.png", "西瓜.png", "金币.png", "炸弹.png", "樱桃.png"];
  const SYMBOL_NAMES = ["皇冠", "幸运7", "西瓜", "金币", "炸弹", "樱桃"];
  const SYMBOL_COLORS = ["#ffd33d", "#ff3d4d", "#3fca68", "#ffd84a", "#6d6576", "#e62255"];
  const REEL_H = 22;
  const REEL_COUNT = 18;

  function st(id) {
    let s = window.slotSt[id];
    if (!s) {
      s = {
        results: [0, 1, 3],
        duration: 5,
        serverRemaining: 0,
        serverAt: 0,
        drawing: false,
        revealAt: 0,
        triple: false,
      };
      window.slotSt[id] = s;
    }
    return s;
  }

  function baseOf(kind) {
    return (window.arena && arena.lookOf && arena.lookOf(kind).base) || "/ball/老虎机";
  }

  function ensure(el, cls, tag = "div") {
    let n = el.querySelector(`:scope > .${cls}`);
    if (!n) {
      n = document.createElement(tag);
      n.className = cls;
      el.appendChild(n);
    }
    return n;
  }

  function setImage(el, cls, src) {
    let img = el.querySelector(`:scope > .${cls}`);
    if (!img) {
      img = document.createElement("img");
      img.className = cls;
      img.alt = "";
      el.appendChild(img);
    }
    if (img.getAttribute("src") !== src) img.src = src;
    return img;
  }

  function decodeReels(amount) {
    const n = Math.max(0, Math.round(Number(amount) || 0));
    return [Math.floor(n / 100) % 10, Math.floor(n / 10) % 10, n % 10].map((v) => Math.max(0, Math.min(5, v)));
  }

  function remainingOf(s, now) {
    return Math.max(0, s.serverRemaining - (now - s.serverAt) / 1000);
  }

  function fileForSymbol(symbol) {
    return SYMBOL_FILES[Math.max(0, Math.min(5, symbol | 0))];
  }

  function fileForBullet(kind) {
    if (kind === "老虎机樱桃弹") return "樱桃.png";
    if (kind === "老虎机西瓜弹") return "西瓜.png";
    if (kind === "老虎机皇冠弹") return "皇冠.png";
    if (kind === "老虎机幸运7弹" || kind === "老虎机超大幸运7") return "幸运7.png";
    if (kind === "老虎机炸弹弹") return "炸弹.png";
    if (kind === "老虎机金币弹") return "金币.png";
    return "幸运7.png";
  }

  function overRoot() {
    return document.getElementById("fx") || document.body;
  }

  function spawnImage(ctx, fx, file, worldSize, angle = 0, cls = "slot-fx-img") {
    const root = overRoot();
    const scale = ctx.scale || 1;
    const size = Math.max(20, worldSize * scale);
    const node = document.createElement("div");
    node.className = `fx ${cls}`;
    node.style.left = `${ctx.x}px`;
    node.style.top = `${ctx.y}px`;
    node.style.setProperty("--size", `${size}px`);
    node.style.setProperty("--ang", `${angle}rad`);
    const img = document.createElement("img");
    img.alt = "";
    img.src = `${baseOf(fx.kind)}/fx/${file}`;
    node.appendChild(img);
    root.appendChild(node);
    img.addEventListener("animationend", () => node.remove(), { once: true });
    return node;
  }

  function spawnBombTimer(ctx, fx) {
    const root = overRoot();
    const node = document.createElement("div");
    node.className = "fx fx-slot-bomb-timer";
    node.style.left = `${ctx.x}px`;
    node.style.top = `${ctx.y - 20 * (ctx.scale || 1)}px`;
    const img = document.createElement("img");
    img.alt = "";
    img.src = `${baseOf(fx.kind)}/fx/炸弹.png`;
    const span = document.createElement("span");
    span.textContent = `${Math.max(0, Number(fx.amount) || 0).toFixed(1)}s`;
    node.append(img, span);
    root.appendChild(node);
    node.addEventListener("animationend", () => node.remove(), { once: true });
  }

  window.lookFX.slot = {
    unmount(el) {
      el?.querySelector(":scope > .slot-machine")?.remove();
    },
    tick(el, u) {
      if (!el || !el.classList.contains("look-slot")) return;
      const s = st(u.id);
      const base = baseOf(u.kind);
      const now = performance.now();
      const remaining = remainingOf(s, now);
      const drawing = s.drawing && remaining > 0;
      el.classList.toggle("slot-drawing", drawing);

      const art = ensure(el, "slot-machine");
      setImage(art, "slot-frame", `${base}/fx/老虎机机身.png`);
      const reels = ensure(art, "slot-reels");
      for (let i = 0; i < 3; i++) {
        let win = reels.querySelector(`:scope > .slot-window.r${i}`);
        if (!win) {
          win = document.createElement("div");
          win.className = `slot-window r${i}`;
          const strip = document.createElement("div");
          strip.className = "slot-strip";
          for (let n = 0; n < REEL_COUNT; n++) {
            const img = document.createElement("img");
            img.alt = "";
            img.src = `${base}/fx/${SYMBOL_FILES[n % 6]}`;
            strip.appendChild(img);
          }
          win.appendChild(strip);
          reels.appendChild(win);
        }
        let strip = win.querySelector(":scope > .slot-strip");
        const symbol = s.results[i] | 0;
        const elapsed = Math.max(0, s.duration - remaining);
        const stopAt = s.duration - [0.85, 0.45, 0.08][i];
        let offset = symbol * REEL_H;
        const active = drawing && elapsed < stopAt;
        win.classList.toggle("rolling", active);
        win.classList.toggle("stopped", !active);
        if (active) {
          const cycle = REEL_H * 6;
          offset = (elapsed * (520 + i * 95) + i * 9) % cycle;
          strip.style.filter = "blur(1.2px)";
        } else {
          strip.style.filter = "none";
        }
        strip.style.transform = `translateY(${-offset}px)`;
      }

      let badge = ensure(art, "slot-badge");
      if (drawing) {
        const reelNames = s.results.map((v) => SYMBOL_NAMES[v] || "").join(" · ");
        badge.textContent = `抽奖 ${remaining.toFixed(1)}s`;
        badge.title = reelNames;
      } else if (s.triple && now < s.revealAt + 900) {
        badge.textContent = `${SYMBOL_NAMES[s.results[0]]}大奖`;
        badge.style.color = SYMBOL_COLORS[s.results[0]] || "#fff";
      } else {
        badge.textContent = "老虎机";
        badge.style.color = "#ffe9a3";
      }
      art.classList.toggle("triple", s.triple && now < s.revealAt + 900);
    },
  };

  window.lookFX["slot-cherry-ball"] = {
    unmount(el) {
      el?.querySelector(":scope > .slot-cherry-art")?.remove();
    },
    tick(el, u) {
      if (!el) return;
      setImage(el, "slot-cherry-art", `${baseOf(u.kind)}/fx/樱桃球.png`);
    },
  };

  window.lookFX["slot-bullet"] = {
    unmount(el) {
      el?.querySelector(":scope > .slot-bullet-img")?.remove();
    },
    tick(el, u) {
      if (!el) return;
      const img = setImage(el, "slot-bullet-img", `${baseOf(u.kind)}/fx/${fileForBullet(u.kind)}`);
      const angle = Math.atan2(-(Number(u.vy) || 0), Number(u.vx) || 1) * 180 / Math.PI;
      img.style.transform = `translate(-50%, -50%) rotate(${angle}deg)`;
      el.classList.toggle("jackpot-shot", u.kind === "老虎机超大幸运7");
    },
  };

  function updateDraw(s, fx, initial) {
    const results = decodeReels(fx.amount);
    const duration = Math.max(0.2, Number(fx.vx) || 5);
    const remaining = Math.max(0, Math.min(duration, Number(fx.vy) || duration));
    const changed = results.some((v, i) => v !== s.results[i]);
    s.results = results;
    s.duration = duration;
    s.serverRemaining = remaining;
    s.serverAt = performance.now();
    s.drawing = remaining > 0.02;
    s.triple = results[0] === results[1] && results[1] === results[2];
    if (initial || changed || !s.drawing) s.revealAt = 0;
  }

  arena.registerShot("slot-draw", (fx, ctx) => {
    const s = st(fx.unitId);
    updateDraw(s, fx, true);
    s.revealAt = 0;
    arena.spawnFx("fx-slot-shield", ctx.x, ctx.y, fx.kind);
  });

  arena.registerShot("slot-state", (fx) => updateDraw(st(fx.unitId), fx, false));

  arena.registerShot("slot-result", (fx, ctx) => {
    const s = st(fx.unitId);
    s.results = decodeReels(fx.amount);
    s.triple = Number(fx.vx) > 0;
    s.drawing = false;
    s.serverRemaining = 0;
    s.revealAt = performance.now();
    arena.spawnFx(fx.vx > 0 ? "fx-slot-jackpot-burst" : "fx-slot-reveal", ctx.x, ctx.y, fx.kind);
    if (Number(fx.vx) > 0) {
      const symbol = Math.max(0, Math.min(5, Math.round(Number(fx.vy) || 0)));
      for (let i = 0; i < 6; i++) {
        spawnImage(ctx, fx, fileForSymbol(symbol), 42 + i * 5, Math.random() * Math.PI * 2, "slot-fx-symbol");
      }
    }
  });

  arena.registerShot("slot-coin-queue", (fx, ctx) => {
    for (let i = 0; i < 10; i++) {
      spawnImage({ ...ctx, x: ctx.x + (Math.random() - 0.5) * 100, y: ctx.y + (Math.random() - 0.5) * 80 }, fx, "金币.png", 24 + Math.random() * 16, Math.random() * 6, "slot-fx-symbol");
    }
  });

  arena.registerShot("slot-jackpot", (fx, ctx) => {
    arena.spawnFx("fx-slot-jackpot-burst", ctx.x, ctx.y, fx.kind);
    for (let i = 0; i < 14; i++) {
      const ang = Math.PI * 2 * i / 14;
      spawnImage(ctx, fx, "幸运7.png", 52, ang, "slot-fx-symbol");
    }
  });

  arena.registerShot("slot-bullet-hit", (fx, ctx) => {
    arena.spawnFx("fx-flash", ctx.x, ctx.y, fx.kind);
    arena.burst(ctx.x, ctx.y, fx.kind, 8);
  });

  arena.registerShot("slot-bomb-bullet", (fx, ctx) => {
    spawnImage(ctx, fx, "炸弹.png", 62, 0, "slot-fx-bomb");
    arena.spawnFx("fx-blast-wave", ctx.x, ctx.y, fx.kind, { "--r": `${Number(fx.amount) * 2 * (ctx.scale || 1)}px` }, overRoot());
  });

  arena.registerShot("slot-bomb-boom", (fx, ctx) => {
    spawnImage(ctx, fx, "炸弹.png", 100, 0, "slot-fx-bomb");
    arena.spawnFx("fx-blast-core", ctx.x, ctx.y, fx.kind);
    arena.spawnFx("fx-blast-wave", ctx.x, ctx.y, fx.kind, { "--r": `${Number(fx.amount) * 2.4 * (ctx.scale || 1)}px` }, overRoot());
    arena.burst(ctx.x, ctx.y, fx.kind, 26);
  });

  arena.registerShot("slot-bomb-hold", (fx, ctx) => spawnBombTimer(ctx, fx));
  arena.registerShot("slot-bomb-transfer", (fx, ctx) => {
    spawnImage(ctx, fx, "炸弹.png", 56, 0, "slot-fx-bomb");
    arena.spawnFx("fx-ring", ctx.x, ctx.y, fx.kind);
  });

  arena.registerShot("slot-cherry-fire", (fx, ctx) => {
    spawnImage(ctx, fx, "樱桃.png", 30, 0, "slot-fx-symbol");
  });
})();
