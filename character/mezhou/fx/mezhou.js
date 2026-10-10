// 我与周旋久前端：头顶「我」字 + 状态文字，每个形态用不同图片装饰，
// 一次性特效使用 特效-斩/星/爆炸/圣光/虚空/缠绕/血/问.png 等图片。
window.mezhouSt = window.mezhouSt || {};

(function () {
  const STATE_NAMES = ["无事发生", "误认", "精神病", "虚空", "神圣", "自杀", "怪物"];
  const STATE_COLORS = ["#c9d4e6", "#7cf9ff", "#ff5b6b", "#b98bff", "#ffe08a", "#ff4b5c", "#ff2a2a"];

  function st(id) {
    let s = window.mezhouSt[id];
    if (!s) {
      s = { mode: 0 };
      window.mezhouSt[id] = s;
    }
    return s;
  }

  function modeOf(id) {
    return st(id).mode | 0;
  }

  function baseOf(kind) {
    return (window.arena && arena.lookOf && arena.lookOf(kind).base) || "/ball/我与周旋久";
  }

  function ensure(el, cls) {
    let n = el.querySelector(`:scope > .${cls}`);
    if (!n) {
      n = document.createElement("div");
      n.className = cls;
      el.appendChild(n);
    }
    return n;
  }

  function setStateText(el, mode) {
    const txt = ensure(el, "mezhou-state");
    txt.textContent = STATE_NAMES[mode] || "";
    const c = STATE_COLORS[mode] || "#ffffff44";
    txt.style.color = c;
    txt.style.borderColor = c;
    txt.style.boxShadow = `0 0 9px ${c}`;
  }

  function setImg(el, cls, src, active) {
    let img = el.querySelector(`:scope > .${cls}`);
    if (active) {
      if (!img) {
        img = document.createElement("img");
        img.className = `mezhou-orn ${cls}`;
        img.alt = "";
        img.src = src;
        el.appendChild(img);
      }
    } else if (img) {
      img.remove();
    }
  }

  function dirAng(fx) {
    const vx = Number(fx && fx.vx) || 0;
    const vy = Number(fx && fx.vy) || 0;
    if (Math.hypot(vx, vy) < 1e-6) return 0;
    return Math.atan2(-vy, vx);
  }

  // 一次性图片特效：放在 #fx，动画结束自动移除。
  function spawnShot(ctx, fx, imgName, worldSize, ang, spin) {
    const root = document.getElementById("fx");
    if (!root) return;
    const scale = ctx.scale || 1;
    const size = Math.max(26, worldSize * scale);
    const el = document.createElement("div");
    el.className = "fx mezhou-shot";
    el.style.left = `${ctx.x}px`;
    el.style.top = `${ctx.y}px`;
    el.style.setProperty("--size", `${size}px`);
    el.style.setProperty("--ang", `${ang || 0}rad`);
    const img = document.createElement("img");
    img.className = `mezhou-shot-img${spin ? " spin" : ""}`;
    img.alt = "";
    img.src = `${baseOf(fx.kind)}/fx/${imgName}`;
    el.appendChild(img);
    root.appendChild(el);
    img.addEventListener("animationend", () => el.remove());
  }

  window.lookFX["mezhou-me"] = {
    unmount(el) {
      if (!el) return;
      for (const c of [".mezhou-name", ".mezhou-state", ".mezhou-knife", ".mezhou-orn"]) {
        el.querySelectorAll(`:scope > ${c}`).forEach((n) => n.remove());
      }
    },
    tick(el, u) {
      if (!el || !el.classList.contains("look-mezhou-me")) return;
      const s = st(u.id);
      const base = baseOf(u.kind);
      ensure(el, "mezhou-name").textContent = "我";
      setStateText(el, s.mode);

      el.classList.toggle("mode-mistake", s.mode === 1);
      el.classList.toggle("mode-psycho", s.mode === 2);
      el.classList.toggle("mode-void", s.mode === 3);
      el.classList.toggle("mode-holy", s.mode === 4);
      el.classList.toggle("mode-suicide", s.mode === 5);
      el.classList.toggle("mode-monster", s.mode === 6);

      let knife = el.querySelector(":scope > .mezhou-knife");
      if (s.mode === 2) {
        if (!knife) {
          knife = document.createElement("img");
          knife.className = "mezhou-knife";
          knife.alt = "";
          knife.src = `${base}/fx/小刀.png`;
          el.appendChild(knife);
        }
      } else if (knife) {
        knife.remove();
      }

      setImg(el, "orn-mistake", `${base}/fx/特效-缠绕.png`, s.mode === 1);
      setImg(el, "orn-mistake-q", `${base}/fx/特效-问.png`, s.mode === 1);
      setImg(el, "orn-psycho", `${base}/fx/特效-血.png`, s.mode === 2);
      setImg(el, "orn-void", `${base}/fx/特效-虚空.png`, s.mode === 3);
      setImg(el, "orn-holy", `${base}/fx/特效-圣光.png`, s.mode === 4);
      setImg(el, "orn-suicide", `${base}/fx/特效-爆炸.png`, s.mode === 5);
      setImg(el, "orn-monster", `${base}/fx/特效-嚎叫.png`, s.mode === 6);
      setImg(el, "orn-beast-shadow", `${base}/fx/特效-兽影.png`, s.mode === 6);
      setImg(el, "orn-beast-claw", `${base}/fx/特效-爪.png`, s.mode === 6);
      setImg(el, "orn-beast-claw2", `${base}/fx/特效-爪.png`, s.mode === 6);
      setImg(el, "orn-beast-eye", `${base}/fx/特效-兽眼.png`, s.mode === 6);
      setImg(el, "orn-beast-fang", `${base}/fx/特效-牙.png`, s.mode === 6);
    },
  };

  // 周旋久跟着主人显示同一个状态文字。
  window.lookFX["mezhou-zhou"] = {
    unmount(el) {
      el?.querySelector(":scope > .mezhou-state")?.remove();
    },
    tick(el, u) {
      if (!el) return;
      setStateText(el, modeOf(u.ownerId));
    },
  };

  arena.registerShot("mode", (fx) => {
    st(fx.unitId).mode = Math.max(0, Math.round(Number(fx.amount) || 0));
  });

  arena.registerShot("bump", (fx, ctx) => {
    spawnShot(ctx, fx, "特效-星.png", 46, Math.random() * Math.PI * 2, false);
  });

  arena.registerShot("knife", (fx, ctx) => {
    spawnShot(ctx, fx, "特效-斩.png", 130, dirAng(fx), false);
    spawnShot(ctx, fx, "特效-血.png", 92, Math.random() * Math.PI * 2, true);
  });

  arena.registerShot("void", (fx, ctx) => {
    spawnShot(ctx, fx, "特效-虚空.png", 58, 0, true);
  });

  arena.registerShot("holy", (fx, ctx) => {
    spawnShot(ctx, fx, "特效-圣光.png", 300, 0, true);
    spawnShot(ctx, fx, "特效-圣光.png", 210, Math.PI, true);
  });

  arena.registerShot("suicide-warn", (fx, ctx) => {
    spawnShot(ctx, fx, "特效-问.png", 40, 0, false);
  });

  arena.registerShot("suicide", (fx, ctx) => {
    spawnShot(ctx, fx, "特效-爆炸.png", 360, 0, true);
    spawnShot(ctx, fx, "特效-星.png", 90, 0, true);
  });

  arena.registerShot("howl", (fx, ctx) => {
    spawnShot(ctx, fx, "特效-嚎叫.png", 520, 0, true);
    spawnShot(ctx, fx, "特效-牙.png", 300, 0, false);
    spawnShot(ctx, fx, "特效-兽眼.png", 200, 0, false);
    for (let i = 0; i < 5; i++) {
      spawnShot(ctx, fx, "特效-爪.png", 250, Math.random() * Math.PI * 2, false);
    }
    arena.spawnFx("fx-shock", ctx.x, ctx.y, fx.kind);
    arena.spawnFx("fx-ring", ctx.x, ctx.y, fx.kind);
    arena.burst(ctx.x, ctx.y, fx.kind, 20);
  });
})();
