// 观察者前端：单眼瞳孔追踪锁定目标，显示观测值、视线和三张弹体胶片。
window.observerSt = window.observerSt || {};
window.lookFX = window.lookFX || {};

(function () {
  function st(id) {
    let s = window.observerSt[id];
    if (!s) {
      s = { gauge: 0, records: 0, targetId: 0, tones: [], replayUntil: 0, lidUntil: 0 };
      window.observerSt[id] = s;
    }
    return s;
  }

  function baseOf(kind) {
    return (window.arena && arena.lookOf && arena.lookOf(kind).base) || "/ball/观察者";
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

  function directionTo(u, target) {
    if (!u || !target) return 0;
    const dx = target.x - u.x;
    const dy = target.y - u.y;
    if (Math.hypot(dx, dy) < 1e-6) return 0;
    return Math.atan2(-dy, dx) * 180 / Math.PI;
  }

  function pushTone(id, tone) {
    const s = st(id);
    s.tones.push(Number(tone) || 0);
    if (s.tones.length > 3) s.tones = s.tones.slice(-3);
  }

  window.lookFX.observer = {
    unmount(el) {
      if (!el) return;
      for (const cls of [".observer-eye", ".observer-gauge-wrap", ".observer-films"]) {
        el.querySelector(`:scope > ${cls}`)?.remove();
      }
    },
    tick(el, u, ctx) {
      if (!el || !el.classList.contains("look-observer")) return;
      const s = st(u.id);
      const units = (ctx && ctx.units) || [];
      const target = units.find((o) => o && o.id === s.targetId);
      const now = performance.now();
      el.classList.toggle("observer-locked", !!target);
	      el.classList.toggle("observer-armored", s.gauge >= 50);
      el.classList.toggle("observer-replaying", now < s.replayUntil);

      const eye = ensure(el, "observer-eye");
      const iris = ensure(eye, "observer-iris");
      const pupil = ensure(eye, "observer-pupil");
      const degree = directionTo(u, target);
      const rad = degree * Math.PI / 180;
      const shift = target ? 5 : 0;
      iris.style.transform = `translate(${Math.cos(rad) * shift}px, ${Math.sin(rad) * shift}px)`;
      pupil.style.transform = `translate(${Math.cos(rad) * 2}px, ${Math.sin(rad) * 2}px)`;

      const gauge = ensure(el, "observer-gauge-wrap");
      let bar = gauge.querySelector("i");
      if (!bar) {
        gauge.innerHTML = "<span>观测</span><b><i></i></b>";
        bar = gauge.querySelector("i");
      }
      bar.style.width = `${Math.max(0, Math.min(100, s.gauge))}%`;

      const films = ensure(el, "observer-films");
      for (let i = 0; i < 3; i++) {
        let card = films.querySelector(`:scope > .observer-film.f${i}`);
        if (!card) {
          card = document.createElement("div");
          card.className = `observer-film f${i}`;
          films.appendChild(card);
        }
        const active = i < s.records;
        card.classList.toggle("active", active);
        const hue = s.tones[i] ?? (170 + i * 55);
        card.style.setProperty("--hue", String(hue));
      }
    },
    guide(u, ctx) {
      if (!u || !ctx || !ctx.ensureGuide || !ctx.placeSeg || !window.arena) return;
      const s = st(u.id);
      if (!s.targetId) return;
      const target = (ctx.units || []).find((o) => o && o.id === s.targetId);
      if (!target) return;
      const [x1, y1] = arena.screenPos(u.x, u.y, ctx.scale, ctx.cx, ctx.cy);
      const [x2, y2] = arena.screenPos(target.x, target.y, ctx.scale, ctx.cx, ctx.cy);
      const g = ctx.ensureGuide(`observer-gaze-${u.id}`, "observer-gaze");
      g.className = "guide observer-gaze";
      ctx.placeSeg(g, x1, y1, x2, y2, u.kind);
      if (ctx.seenGuides) ctx.seenGuides.add(g.id);
    },
  };

window.lookFX["observer-kunai-copy"] = {
  tick(el, u) {
    if (!el) return;
    const ang = Math.atan2(-(Number(u.vy) || 0), Number(u.vx) || 1);
    el.style.setProperty("--travel", `${ang}rad`);
  },
};

  window.lookFX["observer-beam"] = {
    unmount(el) {
      el?.querySelector(":scope > .observer-beam-img")?.remove();
    },
    tick(el) {
      if (!el) return;
      ensure(el, "observer-beam-img");
    },
  };

  function spawnFilm(ctx, fx, cls, extra = {}) {
    const root = document.getElementById("fx");
    if (!root) return null;
    const node = document.createElement("div");
    node.className = `fx ${cls}`;
    node.style.left = `${ctx.x}px`;
    node.style.top = `${ctx.y}px`;
    for (const [key, value] of Object.entries(extra)) node.style.setProperty(key, value);
    root.appendChild(node);
    node.addEventListener("animationend", () => node.remove(), { once: true });
    return node;
  }

  arena.registerShot("observer-state", (fx) => {
    const s = st(fx.unitId);
    s.gauge = Number(fx.amount) || 0;
    s.records = Math.max(0, Math.min(3, Math.round(Number(fx.vx) || 0)));
    s.targetId = Math.max(0, Math.round(Number(fx.vy) || 0));
    while (s.tones.length < s.records) s.tones.push(170 + s.tones.length * 55);
    if (s.tones.length > s.records) s.tones = s.tones.slice(-s.records);
  });

  arena.registerShot("observer-record", (fx, ctx) => {
    pushTone(fx.unitId, fx.amount);
    const node = spawnFilm(ctx, fx, "fx-observer-record", { "--hue": String(Number(fx.amount) || 0) });
    if (node) node.textContent = "记录";
  });

  arena.registerShot("observer-lock", (fx, ctx) => {
    spawnFilm(ctx, fx, "fx-observer-lock");
  });

  arena.registerShot("observer-replay", (fx, ctx) => {
    const s = st(fx.unitId);
    s.replayUntil = performance.now() + 1300;
    spawnFilm(ctx, fx, "fx-observer-replay");
    arena.burst(ctx.x, ctx.y, fx.kind, 16);
  });

  arena.registerShot("observer-copy-shot", (fx, ctx) => {
    const root = document.getElementById("fx");
    if (!root) return;
    const ang = Math.atan2(-(Number(fx.vy) || 0), Number(fx.vx) || 1);
    spawnFilm(ctx, fx, "fx-observer-copy", { "--ang": `${ang}rad` });
  });

  arena.registerShot("observer-beam-fire", (fx, ctx) => {
    const ang = Math.atan2(-(Number(fx.vy) || 0), Number(fx.vx) || 1);
    spawnFilm(ctx, fx, "fx-observer-beam-fire", { "--ang": `${ang}rad` });
  });

  arena.registerShot("observer-beam-hit", (fx, ctx) => {
    arena.spawnFx("fx-flash", ctx.x, ctx.y, fx.kind);
    arena.burst(ctx.x, ctx.y, fx.kind, 8);
  });
})();
