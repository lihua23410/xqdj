// 只读真实的盾/碎片快照与带归属的破盾事件；所有动效使用对局时间。
(function () {
  const hooks = window.lookFX = window.lookFX || {};
  const body = `<svg viewBox="0 0 100 100" aria-hidden="true">
    <path class="warden-armor-shadow" d="M50 5 87 27 95 61 67 91 28 88 6 58 15 25Z"/>
    <path class="warden-armor-light" d="M15 25 50 5 47 40 6 58Z"/>
    <circle class="warden-body-rim" cx="50" cy="50" r="45"/>
    <path class="warden-center-plate" d="m40 27 25 11 6 25-24 13-20-21Z"/>
    <path class="warden-center-edge" d="m40 27 7 23 24 13M47 50v26M15 25l13 8m58 29-12 4"/>
    <g class="warden-rivets"><circle cx="22" cy="44" r="2"/><circle cx="75" cy="31" r="2"/><circle cx="76" cy="75" r="2"/><circle cx="30" cy="78" r="2"/></g>
    <circle class="warden-body-flash" cx="50" cy="50" r="49"/>
  </svg>`;
  const thick = "M-8 -4 -3 -7 8 -3 6 5 -4 7 -8 3Z";
  const thin = "M-8 -2 6 -4 8 2 -5 4Z";
  const pulse = (time, at, duration) => {
    const age = time - at;
    return age >= 0 && age < duration ? 1 - age / duration : 0;
  };
  function mount(el, className, html) {
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
  function ownerState(id) { return document.getElementById(`warden-state-${id}`)?._warden; }
  function platePath() {
    const point = (r, a) => `${(50 + Math.sin(a) * r).toFixed(3)} ${(50 - Math.cos(a) * r).toFixed(3)}`;
    const a = 19 * Math.PI / 180;
    return `M${point(49,-a)}A49 49 0 0 1 ${point(49,a)}L${point(40,a)}A40 40 0 0 0 ${point(40,-a)}Z`;
  }
  const shield = `<svg viewBox="0 0 100 100" aria-hidden="true">
    <circle class="warden-shield-contour" cx="50" cy="50" r="48"/>
    ${Array.from({ length: 8 }, (_, i) => `<g class="warden-shield-piece" style="--sector:${i * 45}deg"><path d="${platePath()}"/></g>`).join("")}
  </svg>`;
  const burst = `<svg viewBox="-60 -60 120 120" aria-hidden="true">
    <circle class="warden-burst-halo" r="24"/>
    <g class="warden-shock-arcs"><path d="M-27 -7A28 28 0 0 1 -17 -22M8 -27A28 28 0 0 1 24 -14M27 8A28 28 0 0 1 15 24M-8 27A28 28 0 0 1 -25 13"/></g>
    ${Array.from({ length: 8 }, (_, i) => `<g class="warden-burst-piece" style="--sector:${i * 45}deg;--spin:${i % 2 ? -32 : 32}deg"><path d="M-3 -21 3 -21 2 -18 -2 -18Z"/><circle cx="5" cy="-25" r="0.65"/></g>`).join("")}
  </svg>`;

  hooks.warden = {
    guide(u, ctx) {
      const node = ctx.ensureGuide(`warden-state-${u.id}`, "warden-state");
      const time = ctx.time || 0;
      if (!node._warden || time < node._warden.time) {
        node._warden = { time, shield: null, known: false, shieldAt: -Infinity, burstAt: -Infinity, mode: "full", key: "" };
        node.innerHTML = burst;
      }
      const data = node._warden;
      const currentShield = (ctx.units || []).find(n => n.kind === "盾" && n.ownerId === u.id);
      const id = currentShield ? currentShield.id : null;
      if (id !== data.shield) {
        if (id != null && (data.known || time < 0.15)) data.shieldAt = time;
        data.shield = id;
      }
      data.known = true;
      data.time = time;
      for (const fx of ctx.effects || []) {
        if (fx.name !== "shatter" || fx.unitId !== u.id || (fx.kind !== "盾碎片" && fx.kind !== "弱化碎片")) continue;
        const mode = fx.kind === "弱化碎片" ? "weak" : "full";
        const key = `${time}:${mode}`;
        if (key === data.key) continue;
        data.key = key;
        data.burstAt = time;
        data.mode = mode;
        data.x = fx.x ?? u.x;
        data.y = fx.y ?? u.y;
      }
      const strength = pulse(time, data.burstAt, data.mode === "weak" ? 0.32 : 0.38);
      node.style.display = strength > 0 ? "block" : "none";
      node.classList.toggle("warden-burst-weak", data.mode === "weak");
      if (strength > 0) {
        const [x, y] = arena.screenPos(data.x, data.y, ctx.scale, ctx.cx, ctx.cy);
        const size = 120 * ctx.scale;
        Object.assign(node.style, { left: `${x}px`, top: `${y}px`, width: `${size}px`, height: `${size}px` });
        node.style.setProperty("--kind", arena.kindColor(data.mode === "weak" ? "弱化碎片" : "盾碎片"));
        node.style.setProperty("--warden-phase", (1 - strength).toFixed(4));
        node.style.setProperty("--warden-fade", (strength * strength).toFixed(4));
        node.style.setProperty("--warden-flash", pulse(time, data.burstAt, data.mode === "weak" ? 0.12 : 0.09).toFixed(4));
      }
      ctx.seenGuides.add(node.id);
    },
    tick(el, u, ctx = {}) {
      mount(el, "warden-body", body);
      const data = ownerState(u.id);
      const time = ctx.time || 0;
      const strong = data?.mode === "full" ? pulse(time, data.burstAt, 0.09) : 0;
      el.classList.toggle("warden-covered", data?.shield != null);
      el.style.setProperty("--warden-flash", strong.toFixed(4));
    },
    unmount(el) {
      el.querySelector(":scope > .warden-body")?.remove();
      el.classList.remove("warden-covered");
      el.style.removeProperty("--warden-flash");
    },
  };
  hooks["warden-shield"] = {
    tick(el, u, ctx = {}) {
      mount(el, "warden-shield-art", shield);
      const data = ownerState(u.ownerId);
      el.style.setProperty("--warden-assemble", (data ? pulse(ctx.time || 0, data.shieldAt, 0.28) : 0).toFixed(4));
    },
    unmount(el) {
      el.querySelector(":scope > .warden-shield-art")?.remove();
      el.style.removeProperty("--warden-assemble");
    },
  };
  hooks["warden-shard"] = {
    guide(u, ctx) {
      const node = ctx.ensureGuide(`warden-trail-${u.id}`, "warden-shard-trail");
      const time = ctx.time || 0;
      const angle = Math.atan2(-(u.vy || 0), u.vx || 0) * 180 / Math.PI;
      const weak = u.kind === "弱化碎片";
      if (!node._wardenTrail || time < node._wardenTrail.time) {
        node._wardenTrail = { time, x: u.x, y: u.y, samples: [] };
        node.innerHTML = '<svg aria-hidden="true"><path/><path/></svg>';
      }
      const data = node._wardenTrail;
      const dt = time - data.time;
      if (dt > 0) {
        const speed = Math.hypot(u.vx || 0, u.vy || 0);
        if (dt > 0.15 || Math.hypot(u.x - data.x, u.y - data.y) > speed * dt * 1.8 + u.radius * 2) data.samples = [];
        data.samples = data.samples.filter(p => time - p.at < 0.065);
        const last = data.samples[0];
        if (speed > 1 && (!last || Math.hypot(u.x - last.x, u.y - last.y) >= u.radius * 0.5)) {
          data.samples.unshift({ x: u.x, y: u.y, angle, at: time });
          data.samples.length = Math.min(data.samples.length, 4);
        }
      }
      Object.assign(data, { time, x: u.x, y: u.y });
      const echoes = data.samples.filter(p => {
        const distance = Math.hypot(u.x - p.x, u.y - p.y);
        return distance >= u.radius * 0.5 && distance < 22;
      }).slice(0, 2);
      node.style.setProperty("--kind", arena.kindColor(u.kind));
      node.querySelectorAll("path").forEach((path, i) => {
        const p = echoes[i];
        path.style.display = p ? "" : "none";
        if (!p) return;
        const fade = 1 - (time - p.at) / 0.065;
        const [x, y] = arena.screenPos(p.x, p.y, ctx.scale, ctx.cx, ctx.cy);
        path.setAttribute("d", weak ? thin : thick);
        path.setAttribute("transform", `translate(${x.toFixed(2)} ${y.toFixed(2)}) rotate(${p.angle.toFixed(2)}) scale(${(u.radius * 2 / 20 * ctx.scale * 0.9).toFixed(4)})`);
        path.style.opacity = (fade * 0.24).toFixed(4);
      });
      ctx.seenGuides.add(node.id);
    },
    tick(el, u) {
      const weak = u.kind === "弱化碎片";
      const art = mount(el, "warden-fragment", "");
      if (art.dataset.mode !== u.kind) {
        art.dataset.mode = u.kind;
        art.innerHTML = `<svg viewBox="-10 -10 20 20" aria-hidden="true"><path class="warden-fragment-face" d="${weak ? thin : thick}"/><path class="warden-fragment-facet" d="${weak ? 'M-8 -2 6 -4 1 0Z' : 'M-8 -4 -3 -7 8 -3 -1 0Z'}"/><path class="warden-fragment-edge" d="${weak ? 'M-7 -2 6 -4 7 -1' : 'M-7 -4 -3 -6 7 -3'}"/></svg>`;
      }
      el.classList.toggle("warden-weak", weak);
      art.style.transform = `rotate(${Math.atan2(-(u.vy || 0), u.vx || 0)}rad)`;
    },
    unmount(el) {
      el.querySelector(":scope > .warden-fragment")?.remove();
      el.classList.remove("warden-weak");
    },
  };
  // 持续表现由快照钩子绘制，避免旧版的通用圆环、散点叠加。
  arena.registerShot("shatter", () => {});
})();
