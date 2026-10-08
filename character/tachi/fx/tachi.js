// 太刀前端皮肤：手里握着的刀、刃色/纳刀徽标，以及居合/登龙/见切的一次性特效。
// 居合刀光按地慧星那套「大号横扫刀光 + 全屏闪光」处理，颜色按刃色白/黄/红。
window.tachiSt = window.tachiSt || {};

(function () {
  const LEVELS = [
    { name: "无", color: "#d9dfe8", glow: "rgba(180, 190, 210, 0.5)" },
    { name: "白", color: "#f4f7ff", glow: "rgba(240, 245, 255, 0.95)" },
    { name: "黄", color: "#ffd75e", glow: "rgba(255, 205, 70, 0.95)" },
    { name: "红", color: "#ff4b5c", glow: "rgba(255, 70, 90, 0.95)" },
  ];

  // 世界单位长度，乘 scale 变成屏幕像素。场地半径约 280。
  const BLADE = { swing: 110, iaido: 800, helm: 500, spin: 260, burst: 450 };

  function st(id) {
    let s = window.tachiSt[id];
    if (!s) {
      s = { level: 0, sheathe: false, face: 0, pose: 90, shakeUntil: 0 };
      window.tachiSt[id] = s;
    }
    return s;
  }

  function levelOf(id) {
    return LEVELS[st(id).level] || LEVELS[0];
  }

  function baseOf(kind) {
    return (window.arena && window.arena.lookOf && window.arena.lookOf(kind).base) || "/ball/太刀";
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

  function weapon(id) {
    const ball = document.getElementById(`u-${id}`);
    return ball && ball.querySelector(":scope > .tachi-art > .tachi-weapon");
  }

  function facingDeg(u, s) {
    const vx = Number(u && u.vx) || 0;
    const vy = Number(u && u.vy) || 0;
    if (Math.hypot(vx, vy) < 1e-6) {
      return Number.isFinite(s.face) ? s.face : 0;
    }
    const deg = Math.atan2(vx, vy) * (180 / Math.PI);
    s.face = deg;
    return deg;
  }

  // 素材朝上，CSS rotate 顺时针，显示角 = 90 - pose（和盾斧同一口径）。
  function poseCss(pose) {
    return 90 - Number(pose);
  }

  function swing(id, from, to, dur) {
    const img = weapon(id);
    const s = st(id);
    if (!img || typeof gsap === "undefined") {
      if (s) s.pose = to;
      return;
    }
    img._tachiTween = true;
    gsap.killTweensOf(img);
    gsap.fromTo(img, { rotation: poseCss(from) }, {
      rotation: poseCss(to),
      duration: dur,
      ease: "power2.inOut",
      onComplete: () => {
        img._tachiTween = false;
        s.pose = to;
      },
    });
  }

  function dirAng(fx) {
    const vx = Number(fx && fx.vx) || 0;
    const vy = Number(fx && fx.vy) || 0;
    if (Math.hypot(vx, vy) < 1e-6) return Math.random() * Math.PI * 2;
    return Math.atan2(-vy, vx);
  }

  function overRoot() {
    // 一次性特效必须放 #fx：#over 每帧会清掉非单位子节点，放那里会被立刻移除。
    return document.getElementById("fx");
  }

  // 线状挥砍：常态挥刀 / 大回旋 / 突刺。
  function line(ctx, fx, cls, color, worldLen, ang) {
    const scale = ctx.scale || 1;
    const len = Math.max(100, worldLen * scale);
    arena.spawnFx(cls, ctx.x, ctx.y, fx.kind, {
      "--color": color,
      "--len": `${len}px`,
      "--thick": "6px",
      "--ang": `${ang}rad`,
    }, overRoot());
  }

  // 图片刀光：居合 / 登龙剑气。
  function sprite(ctx, fx, cls, color, worldLen, ang) {
    const scale = ctx.scale || 1;
    const len = Math.max(160, worldLen * scale);
    const base = baseOf(fx.kind);
    const imgName = cls === "fx-tachi-qi" ? "剑气.png" : "居合.png";
    const root = overRoot();
    const el = document.createElement("div");
    el.className = `fx ${cls}`;
    el.style.left = `${ctx.x}px`;
    el.style.top = `${ctx.y}px`;
    el.style.setProperty("--len", `${len}px`);
    el.style.setProperty("--ang", `${ang}rad`);
    el.style.setProperty("--color", color);
    const img = document.createElement("img");
    img.className = "tachi-sprite";
    img.alt = "";
    img.src = `${base}/fx/${imgName}`;
    el.appendChild(img);
    root.appendChild(el);
    img.addEventListener("animationend", () => el.remove());
  }

  function flashStage(color) {
    const stage = document.querySelector(".stage");
    if (!stage) return;
    const el = document.createElement("div");
    el.className = "tachi-flash";
    el.style.setProperty("--color", color);
    stage.appendChild(el);
    el.addEventListener("animationend", () => el.remove());
  }

  function shakeStage() {
    const stage = document.querySelector(".stage");
    if (!stage || typeof gsap === "undefined") return;
    gsap.killTweensOf(stage);
    gsap.fromTo(stage, { x: -4, y: -2 }, {
      x: 4,
      y: 2,
      duration: 0.05,
      repeat: 3,
      yoyo: true,
      ease: "power1.inOut",
      onComplete: () => gsap.set(stage, { x: 0, y: 0 }),
    });
  }

  window.lookFX.tachi = {
    unmount(el) {
      el?.querySelector(":scope > .tachi-art")?.remove();
      el?.querySelector(":scope > .tachi-badge")?.remove();
    },
    tick(el, u) {
      if (!el || !el.classList.contains("look-tachi")) return;
      const s = st(u.id);
      const L = levelOf(u.id);
      const base = baseOf(u.kind);

      const art = ensure(el, "tachi-art");
      let img = art.querySelector(":scope > .tachi-weapon");
      if (!img) {
        img = document.createElement("img");
        img.className = "tachi-weapon";
        img.alt = "";
        img.src = `${base}/fx/太刀-1.png`;
        art.appendChild(img);
        if (typeof gsap !== "undefined") gsap.set(img, { transformOrigin: "50% 78%" });
      }
      art.style.transform = `rotate(${facingDeg(u, s)}deg)`;
      img.style.filter = `drop-shadow(0 0 5px ${L.glow})`;
      // 纳刀：刀收回（消失）；拔刀后瞬间出现。
      img.style.opacity = s.sheathe ? "0" : "1";

      el.classList.toggle("sheathed", !!s.sheathe);
      if (!img._tachiTween) {
        const target = s.sheathe ? 170 : 90;
        if (Math.abs(s.pose - target) > 0.5) {
          if (typeof gsap !== "undefined") {
            img._tachiTween = true;
            gsap.to(img, {
              rotation: poseCss(target),
              duration: 0.14,
              ease: "power2.out",
              onComplete: () => {
                img._tachiTween = false;
                s.pose = target;
              },
            });
          } else {
            s.pose = target;
            img.style.transform = `rotate(${poseCss(target)}deg)`;
          }
        }
      }

      // 见切后撤：小球抖动一会儿。
      el.classList.toggle("shaking", performance.now() < s.shakeUntil);

      let b = el.querySelector(":scope > .tachi-badge");
      if (!b) {
        b = document.createElement("div");
        b.className = "tachi-badge";
        el.appendChild(b);
      }
      if (s.sheathe) {
        b.textContent = "纳刀";
        b.style.color = "#dff6ff";
        b.style.boxShadow = "0 0 6px rgba(150, 220, 255, 0.9)";
        b.style.borderColor = "rgba(150, 220, 255, 0.55)";
      } else {
        b.textContent = L.name;
        b.style.color = L.color;
        b.style.boxShadow = `0 0 6px ${L.glow}`;
        b.style.borderColor = L.glow;
      }
    },
  };

  arena.registerShot("blade", (fx) => {
    const s = st(fx.unitId);
    s.level = Math.max(0, Math.min(3, Math.round(Number(fx.amount) || 0)));
  });

  arena.registerShot("sheathe", (fx) => {
    st(fx.unitId).sheathe = (Number(fx.amount) || 0) > 0;
  });

  arena.registerShot("swing", (fx, ctx) => {
    const L = levelOf(fx.unitId);
    swing(fx.unitId, 150, 30, 0.24);
    line(ctx, fx, "fx-tachi-swing", L.color, BLADE.swing, dirAng(fx));
  });

  arena.registerShot("iaido", (fx, ctx) => {
    const L = levelOf(fx.unitId);
    const s = st(fx.unitId);
    s.sheathe = false;
    const img = weapon(fx.unitId);
    if (img) img.style.opacity = "1";
    swing(fx.unitId, 170, -10, 0.3);
    sprite(ctx, fx, "fx-tachi-iaido", L.color, BLADE.iaido, dirAng(fx));
    flashStage(L.color);
    shakeStage();
    arena.burst(ctx.x, ctx.y, fx.kind, 16);
  });

  arena.registerShot("thrust", (fx, ctx) => {
    const L = levelOf(fx.unitId);
    swing(fx.unitId, 90, 20, 0.16);
    line(ctx, fx, "fx-tachi-thrust", L.color, BLADE.swing, dirAng(fx));
  });

  arena.registerShot("helm", (fx, ctx) => {
    const L = levelOf(fx.unitId);
    swing(fx.unitId, -60, 140, 0.3);
    sprite(ctx, fx, "fx-tachi-qi", L.color, BLADE.helm, dirAng(fx));
    flashStage(L.color);
    shakeStage();
  });

  arena.registerShot("burst", (fx, ctx) => {
    const L = levelOf(fx.unitId);
    arena.spawnFx("fx-tachi-burst", ctx.x, ctx.y, fx.kind, {}, overRoot());
    for (let i = 0; i < 6; i++) {
      const ang = (Math.PI * 2 * i) / 6 + Math.random() * 0.5;
      sprite(ctx, fx, "fx-tachi-qi", L.color, BLADE.burst, ang);
    }
    flashStage(L.color);
    arena.burst(ctx.x, ctx.y, fx.kind, 14);
  });

  arena.registerShot("foresight", (fx, ctx) => {
    const L = levelOf(fx.unitId);
    const s = st(fx.unitId);
    s.shakeUntil = performance.now() + 420;
    swing(fx.unitId, 90, 150, 0.2);
    arena.spawnFx("fx-tachi-foresight", ctx.x, ctx.y, fx.kind, { "--color": L.color }, overRoot());
    arena.burst(ctx.x, ctx.y, fx.kind, 8);
  });

  arena.registerShot("spin", (fx, ctx) => {
    const L = levelOf(fx.unitId);
    swing(fx.unitId, 0, 180, 0.3);
    line(ctx, fx, "fx-tachi-spin", L.color, BLADE.spin, dirAng(fx));
    flashStage(L.color);
    arena.burst(ctx.x, ctx.y, fx.kind, 12);
  });

  arena.registerShot("levelup", (fx, ctx) => {
    const L = levelOf(fx.unitId);
    arena.spawnFx("fx-tachi-levelup", ctx.x, ctx.y, fx.kind, { "--color": L.color }, overRoot());
    arena.burst(ctx.x, ctx.y, fx.kind, 12);
  });
})();
