// Pink 前端：三张脸覆盖球体，武器按形态与掉落装备切换。
// 一次性表现使用生成的火球、火箭、子弹、爆炸与斩击 PNG。
window.pinkSt = window.pinkSt || {};
window.lookFX = window.lookFX || {};

(function () {
  const FORM_NAMES = ["平淡", "大笑", "狂笑"];
  const FORM_COLORS = ["#ffd4e7", "#ff98c4", "#ff4d91"];
  const FORM_IMAGES = ["平淡.png", "大笑.png", "狂笑.png"];
  const WEAPON_NAMES = ["", "瓶", "手枪", "霰弹枪", "步枪", "油箱", "加特林", "火箭筒"];
  const WEAPON_IMAGES = ["", "瓶.png", "手枪.png", "霰弹枪.png", "步枪.png", "油箱.png", "加特林.png", "火箭筒.png"];
  const WEAPON_CLASSES = ["", "w-bottle", "w-pistol", "w-shotgun", "w-rifle", "w-fuel", "w-gatling", "w-rocket"];
  const RELOAD_TIME = 6;

  function st(id) {
    let s = window.pinkSt[id];
    if (!s) {
      s = { form: 0, weapon: 0, ammo: 0, reloading: false, reloadRemain: 0, attackFace: 0, attackUntil: 0 };
      window.pinkSt[id] = s;
    }
    return s;
  }

  function baseOf(kind) {
    return (window.arena && arena.lookOf && arena.lookOf(kind).base) || "/ball/粉红色迷人的pink";
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

  function angleOf(u) {
    const vx = Number(u && u.vx) || 0;
    const vy = Number(u && u.vy) || 0;
    if (Math.hypot(vx, vy) < 1e-6) return 0;
    return Math.atan2(-vy, vx) * 180 / Math.PI;
  }

  function dirAng(fx) {
    const vx = Number(fx && fx.vx) || 0;
    const vy = Number(fx && fx.vy) || 0;
    if (Math.hypot(vx, vy) < 1e-6) return Math.random() * Math.PI * 2;
    return Math.atan2(-vy, vx);
  }

  function overRoot() {
    return document.getElementById("fx") || document.body;
  }

  function spawnImage(ctx, fx, file, worldSize, angle = 0, cls = "pink-shot-image") {
    const root = overRoot();
    const scale = ctx.scale || 1;
    const size = Math.max(24, worldSize * scale);
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
    setTimeout(() => node.remove(), 1200);
    return node;
  }


  function lockAttackFace(fx, duration = 300) {
    const vx = Number(fx && fx.vx) || 0;
    const vy = Number(fx && fx.vy) || 0;
    if (Math.hypot(vx, vy) < 1e-6) return;
    const s = st(fx.unitId);
    s.attackFace = Math.atan2(-vy, vx) * 180 / Math.PI;
    s.attackUntil = performance.now() + duration;
  }

  function swingAxe(fx) {
    const img = document.querySelector(`#u-${fx.unitId} .pink-weapon.w-axe`);
    if (!img || typeof img.animate !== "function") return;
    img._pinkSwing?.cancel();
    img._pinkSwing = img.animate([
      { transform: "rotate(-166deg) translateX(1px) scaleX(-1)", offset: 0 },
      { transform: "rotate(-148deg) scaleX(-1)", offset: 0.24 },
      { transform: "rotate(-82deg) translateX(7px) scaleX(-1)", offset: 0.58 },
      { transform: "rotate(-52deg) scaleX(-1)", offset: 0.78 },
      { transform: "rotate(-120deg) scaleX(-1)", offset: 1 },
    ], {
      duration: 380,
      easing: "cubic-bezier(0.16, 0.72, 0.22, 1)",
    });
  }

  function forwardPoint(ctx, angle, distance) {
    const scale = ctx.scale || 1;
    return {
      ...ctx,
      x: ctx.x + Math.cos(angle) * distance * scale,
      y: ctx.y + Math.sin(angle) * distance * scale,
    };
  }

  function bloodSplash(ctx, fx, power = 1, showBurst = false) {
    const angle = dirAng(fx);
    const scale = ctx.scale || 1;
    if (showBurst) {
      const burst = spawnImage(ctx, fx, "血爆.png", 72 + power * 46, angle, "pink-image-shot pink-blood-burst");
      burst.style.zIndex = "35";
    }
    spawnImage(ctx, fx, "血雾.png", 105 + power * 62, Math.random() * Math.PI * 2, "pink-image-shot pink-blood-mist");
    spawnImage(ctx, fx, "血滴.png", 78 + power * 48, angle, "pink-image-shot pink-blood-drop");
    const drops = power >= 1.8 ? 4 : 2;
    for (let i = 0; i < drops; i++) {
      const side = (Math.random() - 0.5) * 70 * power * scale;
      const lift = (Math.random() - 0.5) * 54 * power * scale;
      spawnImage(
        { ...ctx, x: ctx.x + side, y: ctx.y + lift },
        fx, "血雨.png", 72 + power * 38, Math.random() * Math.PI * 2,
        "pink-image-shot pink-blood-rain",
      );
    }
  }

  window.lookFX.pink = {
    unmount(el) {
      if (!el) return;
      for (const cls of [".pink-face", ".pink-rig", ".pink-form-badge", ".pink-weapon-badge", ".pink-reload"]) {
        el.querySelector(`:scope > ${cls}`)?.remove();
      }
    },
    tick(el, u) {
      if (!el || !el.classList.contains("look-pink")) return;
      const s = st(u.id);
      const base = baseOf(u.kind);
      const form = Math.max(0, Math.min(2, s.form | 0));
      const weapon = Math.max(0, Math.min(7, s.weapon | 0));
      const color = FORM_COLORS[form];

      el.classList.remove("pink-plain", "pink-laugh", "pink-manic");
      el.classList.add(["pink-plain", "pink-laugh", "pink-manic"][form]);

      const face = setImage(el, "pink-face", `${base}/${FORM_IMAGES[form]}`);
      face.style.setProperty("--form-color", color);

      let badge = ensure(el, "pink-form-badge");
      badge.textContent = FORM_NAMES[form];
      badge.style.setProperty("--form-color", color);

      let rig = ensure(el, "pink-rig");
      const attackActive = performance.now() < (Number(s.attackUntil) || 0);
      const faceAngle = attackActive ? Number(s.attackFace) || 0 : angleOf(u);
      rig.style.setProperty("--face", `${faceAngle}deg`);

      let file = WEAPON_IMAGES[weapon];
      let label = WEAPON_NAMES[weapon];
      let weaponClass = WEAPON_CLASSES[weapon];
      if (weapon === 0 && form === 1) {
        file = "斧头.png";
        label = "斧头";
        weaponClass = "w-axe";
      } else if (weapon === 0 && form === 2) {
        file = "电锯.png";
        label = "电锯";
        weaponClass = "w-saw";
      } else if (weapon === 0) {
        label = "撞击";
      }

      let img = rig.querySelector(":scope > .pink-weapon");
      if (!img) {
        img = document.createElement("img");
        img.className = "pink-weapon";
        img.alt = "";
        rig.appendChild(img);
      }
      for (const cls of [...img.classList]) {
        if (cls.startsWith("w-")) img.classList.remove(cls);
      }
      if (file) {
        img.style.display = "block";
        if (img.getAttribute("src") !== `${base}/${file}`) img.src = `${base}/${file}`;
        if (weaponClass) img.classList.add(weaponClass);
      } else {
        img.style.display = "none";
      }

      let ammo = "";
      if (weapon !== 0) {
        ammo = s.ammo < 0 ? "∞" : String(Math.max(0, s.ammo | 0));
      }
      let wb = ensure(el, "pink-weapon-badge");
      wb.textContent = ammo ? `${label} ${ammo}` : label;
      wb.style.display = label ? "block" : "none";
      wb.style.setProperty("--form-color", color);

      let reload = el.querySelector(":scope > .pink-reload");
      if (s.reloading) {
        reload = ensure(el, "pink-reload");
        let bar = reload.querySelector("i");
        if (!bar) {
          reload.innerHTML = "<span>装弹</span><b><i></i></b>";
          bar = reload.querySelector("i");
        }
        const ratio = Math.max(0, Math.min(1, 1 - (Number(s.reloadRemain) || 0) / RELOAD_TIME));
        bar.style.width = `${ratio * 100}%`;
      } else if (reload) {
        reload.remove();
      }
    },
  };

  window.lookFX["pink-crowd"] = {
    unmount(el) {
      el?.querySelector(":scope > .pink-crowd-img")?.remove();
    },
    tick(el, u) {
      if (!el) return;
      setImage(el, "pink-crowd-img", `${baseOf(u.kind)}/人群.png`);
    },
  };

  window.lookFX["pink-item"] = {
    unmount(el) {
      el?.querySelector(":scope > .pink-item-img")?.remove();
    },
    tick(el, u) {
      if (!el) return;
      const index = ["瓶", "手枪", "霰弹枪", "步枪", "油箱", "加特林", "火箭筒"].indexOf(u.kind);
      const file = index >= 0 ? WEAPON_IMAGES[index + 1] : "瓶.png";
      el.classList.toggle("item-large", u.kind === "火箭筒" || u.kind === "加特林" || u.kind === "步枪");
      setImage(el, "pink-item-img", `${baseOf(u.kind)}/${file}`);
    },
  };

  window.lookFX["pink-shot"] = {
    unmount(el) {
      el?.querySelector(":scope > .pink-shot-img")?.remove();
    },
    tick(el, u) {
      if (!el) return;
      const img = setImage(el, "pink-shot-img", `${baseOf(u.kind)}/fx/子弹.png`);
      img.style.transform = `translate(-50%, -50%) rotate(${angleOf(u)}deg)`;
    },
  };

  window.lookFX["pink-rocket"] = {
    unmount(el) {
      el?.querySelector(":scope > .pink-rocket-img")?.remove();
    },
    tick(el, u) {
      if (!el) return;
      const img = setImage(el, "pink-rocket-img", `${baseOf(u.kind)}/fx/火箭弹.png`);
      img.style.transform = `translate(-50%, -50%) rotate(${angleOf(u)}deg)`;
    },
  };

  window.lookFX["pink-bottle-shot"] = {
    unmount(el) {
      el?.querySelector(":scope > .pink-bottle-img")?.remove();
    },
    tick(el, u) {
      if (!el) return;
      setImage(el, "pink-bottle-img", `${baseOf(u.kind)}/瓶.png`);
    },
  };

  window.lookFX["pink-fire"] = {
    unmount(el) {
      el?.querySelectorAll(":scope > .pink-flame-img").forEach((n) => n.remove());
    },
    tick(el, u) {
      if (!el) return;
      const base = baseOf(u.kind);
      for (let i = 0; i < 4; i++) {
        let img = el.querySelector(`:scope > .pink-flame-img.f${i}`);
        if (!img) {
          img = document.createElement("img");
          img.className = `pink-flame-img f${i}`;
          img.alt = "";
          img.src = `${base}/fx/火焰.png`;
          el.appendChild(img);
        }
      }
    },
  };

  arena.registerShot("form", (fx) => {
    st(fx.unitId).form = Math.max(0, Math.min(2, Math.round(Number(fx.amount) || 0)));
  });
  arena.registerShot("weapon", (fx) => {
    const s = st(fx.unitId);
    const next = Math.max(0, Math.min(7, Math.round(Number(fx.amount) || 0)));
    if (s.weapon !== next) {
      s.reloading = false;
      s.reloadRemain = 0;
    }
    s.weapon = next;
  });
  arena.registerShot("ammo", (fx) => {
    st(fx.unitId).ammo = Math.round(Number(fx.amount) || 0);
  });
  arena.registerShot("reload-start", (fx) => {
    const s = st(fx.unitId);
    s.reloading = true;
    s.reloadRemain = Number(fx.amount) || RELOAD_TIME;
  });
  arena.registerShot("reload", (fx) => {
    const s = st(fx.unitId);
    s.reloading = true;
    s.reloadRemain = Number(fx.amount) || 0;
  });
  arena.registerShot("reload-done", (fx) => {
    const s = st(fx.unitId);
    s.reloading = false;
    s.reloadRemain = 0;
  });

  arena.registerShot("bump", (fx, ctx) => {
    lockAttackFace(fx, 220);
    arena.spawnFx("fx-flash", ctx.x, ctx.y, fx.kind);
    arena.burst(ctx.x, ctx.y, fx.kind, 4);
  });

  arena.registerShot("melee", (fx, ctx) => {
    const amount = Number(fx.amount) || 0;
    const form = Math.floor(amount / 10);
    const hit = Math.round(amount) % 10 >= 1;
    const count = form >= 2 ? 3 : 1;
    const angle = dirAng(fx);
    lockAttackFace(fx, form === 1 ? 470 : 360);
    if (form === 1) {
      swingAxe(fx);
      const strike = forwardPoint(ctx, angle, 58);
      setTimeout(() => {
        spawnImage(strike, fx, "血斩.png", 190, angle, "pink-image-shot pink-blood-slash");
        arena.burst(strike.x, strike.y, fx.kind, 8);
        if (hit) bloodSplash(strike, fx, 1.45);
      }, 205);
      return;
    }
    for (let i = 0; i < count; i++) {
      const sawAngle = angle + (i - (count - 1) / 2) * 0.24;
      spawnImage(ctx, fx, "血斩.png", 235, sawAngle, "pink-image-shot pink-blood-slash");
    }
    if (hit) {
      bloodSplash(ctx, fx, 2.15);
    }
    arena.burst(ctx.x, ctx.y, fx.kind, 12);
  });

  arena.registerShot("shoot", (fx) => lockAttackFace(fx, 220));
  arena.registerShot("shotgun", (fx) => lockAttackFace(fx, 260));

  arena.registerShot("fuel", (fx, ctx) => {
    lockAttackFace(fx, 420);
    const angle = dirAng(fx);
    const spark = forwardPoint(ctx, angle, 54);
    spawnImage(ctx, fx, "电锯突刺.png", 270, angle, "pink-image-shot pink-saw-thrust");
    spawnImage(spark, fx, "电锯火花.png", 132, angle + Math.random() * 0.3, "pink-image-shot pink-saw-spark");
    arena.spawnFx("fx-flash", spark.x, spark.y, fx.kind);
    arena.burst(spark.x, spark.y, fx.kind, 9);
  });

  arena.registerShot("fuel-hit", (fx, ctx) => {
    lockAttackFace(fx, 300);
    const angle = dirAng(fx);
    const impact = forwardPoint(ctx, angle, 34);
    spawnImage(impact, fx, "电锯突刺.png", 245, angle, "pink-image-shot pink-saw-thrust");
    spawnImage(impact, fx, "电锯火花.png", 150, angle + 0.18, "pink-image-shot pink-saw-spark");
    spawnImage(impact, fx, "血斩.png", 250, angle, "pink-image-shot pink-blood-slash");
    bloodSplash(impact, fx, 2.0, true);
    arena.spawnFx("fx-shock", ctx.x, ctx.y, fx.kind);
  });

  arena.registerShot("bullet-hit", (fx, ctx) => {
    arena.spawnFx("fx-flash", ctx.x, ctx.y, fx.kind);
    arena.burst(ctx.x, ctx.y, fx.kind, 4);
  });
  arena.registerShot("bullet-wall", (fx, ctx) => {
    arena.spawnFx("fx-flash", ctx.x, ctx.y, fx.kind);
    arena.spawnFx("fx-wall-dust", ctx.x, ctx.y, fx.kind);
  });

  arena.registerShot("fire", (fx, ctx) => {
    spawnImage(ctx, fx, "爆炸.png", 78, Math.random() * Math.PI * 2, "pink-image-shot");
  });

  arena.registerShot("rocket-boom", (fx, ctx) => {
    const size = Number(fx.amount) || 82;
    const blast = spawnImage(ctx, fx, "火箭爆炸.png", size * 3.4, Math.random() * Math.PI * 2, "pink-image-shot pink-rocket-blast");
    blast.style.zIndex = "34";
    const fire = spawnImage(ctx, fx, "火焰.png", size * 2.7, Math.random() * Math.PI * 2, "pink-image-shot pink-rocket-fire");
    fire.style.zIndex = "33";
    arena.spawnFx("fx-blast-core", ctx.x, ctx.y, fx.kind);
    arena.spawnFx("fx-blast-wave", ctx.x, ctx.y, fx.kind, { "--r": `${size * 3 * (ctx.scale || 1)}px` }, overRoot());
    arena.spawnFx("fx-blast-rim", ctx.x, ctx.y, fx.kind, { "--r": `${size * 3.4 * (ctx.scale || 1)}px` }, overRoot());
    arena.spawnFx("fx-shock", ctx.x, ctx.y, fx.kind);
    arena.burst(ctx.x, ctx.y, fx.kind, 30);
  });

  arena.registerShot("drop", (fx, ctx) => {
    arena.spawnFx("fx-ring", ctx.x, ctx.y, fx.kind);
    arena.burst(ctx.x, ctx.y, fx.kind, 6);
  });

  arena.registerShot("crowd-spawn", (fx, ctx) => {
    arena.spawnFx("fx-flash", ctx.x, ctx.y, fx.kind);
    arena.spawnFx("fx-ring", ctx.x, ctx.y, fx.kind);
  });

  arena.registerShot("form-up", (fx, ctx) => {
    spawnImage(ctx, fx, "爆炸.png", 112, 0, "pink-image-shot");
    arena.spawnFx("fx-ring", ctx.x, ctx.y, fx.kind);
    arena.burst(ctx.x, ctx.y, fx.kind, 20);
  });
})();
