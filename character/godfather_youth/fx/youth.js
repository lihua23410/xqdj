// 仅使用已有针剂/强化事件、弹体和速度快照，不改变青年教父的战斗状态。
(function () {
  const hooks = window.lookFX = window.lookFX || {};
  window.youthDoses = window.youthDoses || {};
  window.youthBuff = window.youthBuff || {};
  const actors = new Map();
  const shotKinds = new Set(["教父(青年)手枪弹", "教父(青年)强化弹"]);
  let lastTime = -Infinity;
  let lastUnits;

  const coreMark = `<svg viewBox="0 0 100 100" aria-hidden="true">
    <circle class="youth-rim" cx="50" cy="50" r="46.5"/>
    <path class="youth-accent" d="M75 14A44 44 0 0 1 94 48"/>
    <path class="youth-accent-dim" d="M25 87A43 43 0 0 1 12 70"/>
    <path class="youth-mark" d="M39 63 61 36"/>
    <path class="youth-mark-offset" d="m34 68 10-12m12-7 9-11"/>
    <circle class="youth-inject-flare" cx="50" cy="50" r="51"/>
  </svg>`;
  const emitter = `<svg viewBox="0 0 100 100" aria-hidden="true">
    <path class="youth-recoil-mark" d="m86 43 5 7-5 7"/>
    <path class="youth-shot-pulse" d="M98 50h15m-10-7 7 3m-7 11 7-3"/>
  </svg>`;
  const resource = `<svg viewBox="0 0 20 24" aria-hidden="true">
    <rect class="youth-cell-frame" x="3" y="3" width="14" height="18" rx="1.5"/>
    <rect class="youth-cell-fill" x="6" y="6" width="8" height="12" rx=".5"/>
  </svg>`;
  const motion = `<svg viewBox="0 0 100 100" aria-hidden="true">
    <path d="M18 37Q-7 34-39 37M12 50H-54M18 63Q-7 66-39 63"/>
  </svg>`;
  const streak = `<svg viewBox="0 0 100 36" aria-hidden="true">
    <path class="youth-streak-core" d="M17 18H84"/>
    <path class="youth-streak-edge" d="M67 18H84"/>
  </svg>`;

  function clock(ctx) {
    const now = Number.isFinite(ctx.time) ? ctx.time : 0;
    if (now < lastTime) {
      actors.clear();
      for (const id of Object.keys(window.youthDoses)) delete window.youthDoses[id];
      for (const id of Object.keys(window.youthBuff)) delete window.youthBuff[id];
      window.youthSFX?.reset();
      lastUnits = null;
    }
    lastTime = now;
    if (ctx.units && lastUnits !== ctx.units) {
      lastUnits = ctx.units;
      const alive = new Set(ctx.units.filter(u => u.kind === "教父(青年)").map(u => String(u.id)));
      for (const id of actors.keys()) if (!alive.has(String(id))) actors.delete(id);
      for (const map of [window.youthDoses, window.youthBuff]) {
        for (const id of Object.keys(map)) if (!alive.has(id)) delete map[id];
      }
    }
    return now;
  }
  function state(id) {
    let s = actors.get(id);
    if (!s) {
      s = { rounds: null, time: -Infinity, shotAt: -Infinity, doseAt: -Infinity,
        injectSoundAt: -Infinity, spent: -1, shotAngle: 0, shotBuff: false, dash: false, vx: 0, vy: 0 };
      actors.set(id, s);
    }
    return s;
  }
  function pulse(now, at, length) { return Math.max(0, 1 - (now - at) / length); }
  function observe(u, ctx) {
    const now = clock(ctx);
    const s = state(u.id);
    if (s.time === now && s.units === ctx.units) return s;
    const shots = (ctx.units || []).filter(n => n.ownerId === u.id && shotKinds.has(n.kind));
    const fresh = s.rounds ? shots.filter(n => !s.rounds.has(n.id)) : now <= 0.1 ? shots : [];
    if (fresh.length) {
      const shot = fresh.reduce((a, b) => a.id > b.id ? a : b);
      s.shotAt = now;
      s.shotAngle = Math.atan2(-(shot.vy || 0), shot.vx || 0);
      s.shotBuff = shot.kind === "教父(青年)强化弹";
      window.youthSFX?.play("pistol", now);
    }
    s.rounds = new Set(shots.map(n => n.id));
    const speed = Math.hypot(u.vx || 0, u.vy || 0);
    const dash = speed > 400;
    const reversed = dash && s.dash && (u.vx * s.vx + u.vy * s.vy) < -0.65 * speed * Math.hypot(s.vx, s.vy);
    if (dash && (s.time > -Infinity || now <= 0.1) && (!s.dash || reversed)) window.youthSFX?.play("rush", now);
    s.dash = dash;
    s.vx = u.vx || 0;
    s.vy = u.vy || 0;
    s.time = now;
    s.units = ctx.units;
    return s;
  }
  function append(el, className, html) {
    let node = el.querySelector(`:scope > .${className}`);
    if (!node) {
      node = document.createElement("span");
      node.className = className;
      node.setAttribute("aria-hidden", "true");
      node.innerHTML = html;
      el.appendChild(node);
    }
    return node;
  }
  function unmount(el) {
    if (!el) return;
    el.querySelectorAll(":scope > .youth-core, :scope > .youth-emitter, :scope > .youth-motion, :scope > .youth-doses, :scope > .youth-ring, :scope > .youth-streak").forEach(n => n.remove());
    el.classList.remove("youth-buff", "youth-moving-fast", "youth-emitter-hot", "youth-bullet-buffed");
    for (const key of [...el.style]) if (key.startsWith("--youth-")) el.style.removeProperty(key);
    delete el.dataset.youthId;
  }

  hooks.youth = {
    guide: observe,
    unmount,
    tick(el, u, ctx = {}) {
      if (!el || !u) return;
      const s = observe(u, ctx);
      el.dataset.youthId = String(u.id);
      append(el, "youth-motion", motion);
      append(el, "youth-core", coreMark);
      append(el, "youth-emitter", emitter);
      const doses = append(el, "youth-doses", `<i>${resource}</i><i>${resource}</i><i>${resource}</i>`);
      const n = window.youthDoses[u.id];
      doses.classList.toggle("youth-doses-unknown", n == null);
      doses.dataset.remaining = n == null ? "unknown" : String(n);
      doses.querySelectorAll("i").forEach((node, i) => {
        node.classList.toggle("full", n != null && i < n);
        node.classList.toggle("spent", i === s.spent);
      });
      const scale = ctx.scale || 1;
      const shot = pulse(s.time, s.shotAt, 0.095);
      const dose = pulse(s.time, s.doseAt, 0.5);
      const movingAngle = Math.atan2(-(u.vy || 0), u.vx || 0);
      el.classList.toggle("youth-buff", !!window.youthBuff[u.id]);
      el.classList.toggle("youth-moving-fast", s.dash);
      el.classList.toggle("youth-emitter-hot", s.shotBuff && shot > 0);
      el.style.setProperty("--youth-move-angle", `${movingAngle}rad`);
      el.style.setProperty("--youth-emitter-angle", `${s.time - s.shotAt < 0.2 ? s.shotAngle : movingAngle}rad`);
      el.style.setProperty("--youth-flash", shot.toFixed(3));
      el.style.setProperty("--youth-recoil", `${(-shot * 7).toFixed(2)}px`);
      el.style.setProperty("--youth-inject", dose.toFixed(3));
      el.style.setProperty("--youth-dose-w", `${Math.max(4, 5 * scale).toFixed(2)}px`);
      el.style.setProperty("--youth-dose-h", `${Math.max(5, 7 * scale).toFixed(2)}px`);
      el.style.setProperty("--youth-dose-gap", `${Math.max(1, 1.5 * scale).toFixed(2)}px`);
    },
  };
  hooks["youth-shot"] = {
    unmount,
    tick(el, u, ctx = {}) {
      if (!el || !u) return;
      append(el, "youth-streak", streak);
      // 弹体种类在出手时已确定；强化结束后，已飞出的强化弹仍保留自身造型。
      el.classList.toggle("youth-bullet-buffed", u.kind === "教父(青年)强化弹");
      el.style.setProperty("--youth-streak-angle", `${Math.atan2(-(u.vy || 0), u.vx || 0)}rad`);
      el.style.setProperty("--youth-streak-w", `${Math.max(7, 11 * (ctx.scale || 1))}px`);
    },
  };

  function injection(s, now) {
    s.doseAt = now;
    if (s.injectSoundAt !== now) {
      s.injectSoundAt = now;
      window.youthSFX?.play("inject", now);
    }
  }
  arena.registerShot("doses", (fx, ctx = {}) => {
    if (!fx || fx.unitId == null || !Number.isFinite(fx.amount)) return;
    const now = clock(ctx);
    const s = state(fx.unitId);
    const n = Math.max(0, Math.min(3, Math.round(fx.amount)));
    const previous = window.youthDoses[fx.unitId];
    window.youthDoses[fx.unitId] = n;
    if (previous != null && n < previous) {
      s.spent = n;
      injection(s, now);
    }
  });
  arena.registerShot("buff", (fx, ctx = {}) => {
    if (!fx || fx.unitId == null) return;
    const now = clock(ctx);
    const on = fx.amount > 0;
    window.youthBuff[fx.unitId] = on;
    if (on) {
      const s = state(fx.unitId);
      const n = window.youthDoses[fx.unitId];
      if (n != null && n < 3) s.spent = n;
      injection(s, now);
    }
  });
})();
