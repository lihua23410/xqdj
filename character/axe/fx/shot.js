(function () {
  window.axeSt = window.axeSt || {};

  function stOf(id) {
    if (id == null) return null;
    window.axeSt[id] = window.axeSt[id] || { form: 0, pose: 90, phial: 0, redShield: false, redSword: false, flip: 1 };
    return window.axeSt[id];
  }

  function weapon(id) {
    const ball = document.getElementById(`u-${id}`);
    return ball && ball.querySelector(":scope > .axe-art > .axe-weapon");
  }

  // 规格：面向为正上时 0° 在正右（东），逆时针。素材朝北，就是 90°。
  // CSS rotate 顺时针且 0 为素材朝上，所以显示角 = 90 - pose。
  function poseCss(pose) {
    return 90 - Number(pose);
  }

  function prep(img) {
    if (!img || typeof gsap === "undefined") return;
    gsap.set(img, { transformOrigin: "50% 78%" });
  }

  function animDur(fx, fallback) {
    const n = Number(fx && fx.amount);
    return n > 0 ? n : fallback;
  }

  function tweenPose(id, from, to, dur, dir) {
    const img = weapon(id);
    const s = stOf(id);
    if (!s) return;
    let end = to;
    if (dir === "ccw" && to <= from) end = to + 360;
    if (dir === "cw" && to >= from) end = to - 360;
    s.pose = end;
    if (!img || typeof gsap === "undefined") return;
    prep(img);
    img._axeTween = true;
    gsap.killTweensOf(img);
    gsap.fromTo(img, { rotation: poseCss(from) }, {
      rotation: poseCss(end),
      duration: dur,
      ease: "power2.inOut",
      onComplete: () => {
        s.pose = ((end % 360) + 360) % 360;
        img._axeTween = false;
        gsap.set(img, { rotation: poseCss(s.pose) });
      },
    });
  }

  arena.registerShot("look", (fx) => {
    const s = stOf(fx.unitId);
    if (!s) return;
    s.form = fx.amount | 0;
    s.redShield = (fx.vx || 0) > 0;
    s.redSword = (fx.vy || 0) > 0;
    if (s.form === 0) {
      s.pose = 90;
      s.flip = 1;
    } else {
      s.pose = 120;
    }
  });

  arena.registerShot("pose", (fx) => {
    const s = stOf(fx.unitId);
    if (!s) return;
    s.pose = fx.amount;
    const img = weapon(fx.unitId);
    if (img && !img._axeTween) {
      prep(img);
      gsap.set(img, { rotation: poseCss(s.pose) });
    }
  });

  arena.registerShot("face", (fx) => {
    const s = stOf(fx.unitId);
    if (!s) return;
    s.face = Math.atan2(fx.vx || 0, fx.vy || 0) * (180 / Math.PI);
  });

  arena.registerShot("phial", (fx) => {
    const s = stOf(fx.unitId);
    if (s) s.phial = fx.amount | 0;
  });

  arena.registerShot("slash", (fx) => {
    const img = weapon(fx.unitId);
    const s = stOf(fx.unitId);
    if (!img || !s || typeof gsap === "undefined") return;
    prep(img);
    img._axeTween = true;
    gsap.killTweensOf(img);
    const beat = animDur(fx, 0.25);
    gsap.timeline({
      onComplete: () => {
        s.pose = 150;
        img._axeTween = false;
      },
    })
      .fromTo(img, { rotation: poseCss(150) }, { rotation: poseCss(30), duration: beat, ease: "power2.inOut" })
      .to(img, { duration: beat })
      .to(img, { rotation: poseCss(150), duration: beat, ease: "power2.inOut" });
  });

  arena.registerShot("dai", (fx) => {
    tweenPose(fx.unitId, 300, 120, animDur(fx, 0.35), "ccw");
  });

  arena.registerShot("tsui", (fx) => {
    tweenPose(fx.unitId, 120, 120, animDur(fx, 0.35), "cw");
  });

  arena.registerShot("tsui-back", (fx) => {
    tweenPose(fx.unitId, 120, 120, animDur(fx, 0.35), "ccw");
  });

  arena.registerShot("chou", (fx) => {
    tweenPose(fx.unitId, 120, 120, animDur(fx, 0.35), "cw");
  });

  arena.registerShot("flip", (fx) => {
    const img = weapon(fx.unitId);
    const s = stOf(fx.unitId);
    if (!img || !s || typeof gsap === "undefined") return;
    prep(img);
    img._axeTween = true;
    gsap.killTweensOf(img);
    const down = Number(fx.vx) > 0 ? fx.vx : 0.8;
    const up = Number(fx.vy) > 0 ? fx.vy : 0.25;
    gsap.timeline({
      onComplete: () => {
        s.flip = 1;
        img._axeTween = false;
      },
    })
      .to(img, { scaleY: -1, duration: down, ease: "power2.inOut" })
      .to(img, { scaleY: 1, duration: up, ease: "power2.inOut" });
  });

  arena.registerShot("wave", (fx, ctx) => {
    const n = Math.max(1, Math.round(fx.amount || 1));
    const scale = ctx.scale || 1;
    const slam = Math.hypot(fx.vx || 0, fx.vy || 0) || 28;
    const r = Math.max(42, 1.2 * slam * scale);
    const ux = slam > 1e-6 ? (fx.vx || 0) / slam : 1;
    const uy = slam > 1e-6 ? (fx.vy || 0) / slam : 0;
    const px = -uy * scale;
    const py = -ux * scale;
    const gap = r * 0.52;
    for (let i = 0; i < n; i++) {
      const t = n === 1 ? 0 : i - (n - 1) / 2;
      arena.spawnFx("fx-axe-wave", ctx.x + px * t * gap, ctx.y + py * t * gap, fx.kind, {
        "--r": `${r}px`,
        "--delay": `${i * 0.05}s`,
      });
    }
  });
})();
