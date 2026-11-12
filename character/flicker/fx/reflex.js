(function () {
  const api = window.arena;
  const base = document.currentScript?.dataset.pack || "/ball/闪烁";
  const pools = new Map();
  // 固定小声音池；交给通用音频管理器处理暂停、静音与导出。
  for (const [name, file, volume] of [["reflex", "reflex", .65], ["shot", "shot", .38], ["overdrive", "overdrive", .55], ["armed", "armed", .18]]) {
    api.registerShot(name, (fx, ctx = {}) => {
      if (ctx.visualOnly || !api.audio) return;
      if (!pools.has(name)) pools.set(name, { next: 0, voices: Array.from({ length: 3 }, () => api.audio(`${base}/fx/${file}.wav`, { volume, mode: api.SOUND_MODE.NORMAL })) });
      const pool = pools.get(name);
      const voice = pool.voices[pool.next++ % pool.voices.length];
      voice.stop(); voice.reset(); voice.play();
    });
  }
  api.registerShot("flicker-state", () => {});

  function phaseAt(st, time) {
    if (st?.Phase === 1 && time < st.SideUntil) return 1;
    if (st?.Phase === 2 && time <= st.FocusUntil) return 2;
    return 0;
  }
  const clamp = (n) => Math.max(0, Math.min(1, n));
  function hash01(n) {
    const x = Math.sin(n * 12.9898) * 43758.5453;
    return x - Math.floor(x);
  }
  // 侧闪留在路径上的那一笔。身上这层只换位置，不另画。
  function paintTrail(g, px, py, radius, scale, i, alpha) {
    const circle = (x, y, r, color, a, line) => {
      g.globalAlpha = a; g.strokeStyle = color; g.lineWidth = line * scale;
      g.beginPath(); g.arc(x, y, r, 0, Math.PI * 2); g.stroke();
    };
    circle(px - 2 * scale, py, radius, "#77ffe0", alpha, 1.4);
    circle(px + 2 * scale, py, radius, i % 2 ? "#ca8bff" : "#ffc477", alpha * .65);
    g.fillStyle = "#86ffe3"; g.globalAlpha = alpha * .16;
    g.beginPath(); g.arc(px, py, radius * .88, 0, Math.PI * 2); g.fill();
    g.globalAlpha = alpha;
    g.fillRect(px - radius * .55, py - 1 * scale, radius * 1.1, 2 * scale);
  }
  function paintBody(el, phase, time, radius, scale) {
    let canvas = el.querySelector(":scope > canvas.flicker-glitch");
    if (!canvas) {
      canvas = document.createElement("canvas");
      canvas.className = "flicker-glitch";
      el.appendChild(canvas);
    }
    const ball = Math.max(1, radius * 2 * scale);
    const size = ball * 3;
    const ratio = Math.min(2, window.devicePixelRatio || 1);
    if (canvas.width !== Math.ceil(size * ratio)) {
      canvas.width = canvas.height = Math.ceil(size * ratio);
      canvas.style.width = canvas.style.height = `${size}px`;
    }
    const g = canvas.getContext("2d");
    if (!g) return;
    g.setTransform(ratio, 0, 0, ratio, 0, 0);
    g.clearRect(0, 0, size, size);
    const side = phase === 1;
    const count = side ? 8 : 5;
    const beat = side ? 0.028 : 0.046;
    const n = Math.floor(Math.max(0, time) / beat);
    const r = radius * scale;
    for (let i = 0; i < count; i++) {
      if (hash01(n * 19.1 + i * 3.7) >= (side ? 0.8 : 0.64)) continue;
      const ang = hash01(n * 0.73 + i * 1.9) * Math.PI * 2;
      const dist = r * (0.28 + hash01(n * 2.2 + i) * 0.22);
      const alpha = (0.45 + hash01(n * 4.1 + i * 8) * 0.4) * (side ? 0.9 : 0.7);
      paintTrail(g, size / 2 + Math.cos(ang) * dist, size / 2 + Math.sin(ang) * dist, r, scale, i, alpha);
    }
    g.globalAlpha = 1;
  }
  function draw(canvas, u, st, ctx) {
    const width = Math.max(1, Math.ceil(ctx.cx * 2));
    const height = Math.max(1, Math.ceil(ctx.cy * 2));
    const ratio = Math.min(2, window.devicePixelRatio || 1);
    if (canvas.width !== Math.ceil(width * ratio) || canvas.height !== Math.ceil(height * ratio)) {
      canvas.width = Math.ceil(width * ratio); canvas.height = Math.ceil(height * ratio);
      canvas.style.width = `${width}px`; canvas.style.height = `${height}px`;
    }
    const g = canvas.getContext("2d");
    if (!g) return;
    g.setTransform(ratio, 0, 0, ratio, 0, 0);
    g.clearRect(0, 0, width, height);
    const scale = ctx.scale || 1, time = ctx.time || 0;
    const x = (v) => ctx.cx + v * scale;
    const y = (v) => ctx.cy - v * scale;
    const phase = phaseAt(st, time);
    const radius = (u.radius || 18) * scale;
    // 残像只取后端已走过的位置，不外推穿墙、击退或瞬移轨迹。
    for (const [i, p] of (st?.Trail || []).entries()) {
      const age = time - p.T;
      if (age < 0 || age >= .25) continue;
      paintTrail(g, x(p.X), y(p.Y), radius, scale, i, (1 - age / .25) * .65);
    }
    if (st?.Echo?.Until > time) {
      const p = st.Echo;
      paintTrail(g, x(p.X), y(p.Y), radius, scale, 1, .25 + clamp((p.Until - time) / .9) * .35);
    }
    const px = x(u.x), py = y(u.y), ring = radius + (phase === 1 ? 3 : 7) * scale;
    const charge = st?.ReadyAt > time ? clamp(1 - (st.ReadyAt - time) / (st.Cooldown || 1.4)) : 1;
    g.strokeStyle = phase ? "#acffeb" : "#6ec2b7";
    g.globalAlpha = phase ? .95 : .18 + charge * .3;
    g.lineWidth = (phase ? 1.7 : 1) * scale;
    g.beginPath(); g.arc(px, py, ring, -Math.PI / 2, -Math.PI / 2 + Math.max(.02, charge) * Math.PI * 1.85); g.stroke();
    if (phase === 1) {
      const angle = Math.atan2(-(st.ThreatY - u.y), st.ThreatX - u.x);
      g.strokeStyle = "#ffe0a0"; g.lineWidth = 3 * scale; g.globalAlpha = .9;
      g.beginPath(); g.arc(px, py, radius + 11 * scale, angle - .45, angle + .45); g.stroke();
    }
    if (phase && st?.Aim) {
      const ax = x(st.AimX), ay = y(st.AimY), r = 25 * scale;
      g.globalAlpha = phase === 2 ? .65 : .3; g.strokeStyle = "#9fffe1"; g.lineWidth = scale;
      for (let i = 0; i < 4; i++) {
        const angle = Math.PI * .25 + i * Math.PI / 2;
        g.beginPath(); g.arc(ax, ay, r, angle - .2, angle + .2); g.stroke();
      }
    }
    g.globalAlpha = 1;
  }

  window.lookFX = window.lookFX || {};
  window.lookFX.flicker = {
    guide(u, ctx) {
      const id = `flicker-state-${u.id}`;
      const node = ctx.ensureGuide(id, "flicker-scene");
      if (!node._canvas) { node._canvas = document.createElement("canvas"); node.appendChild(node._canvas); }
      if (ctx.time < (node._time ?? -Infinity)) node._state = null;
      for (const fx of ctx.effects || []) {
        if (fx.name === "flicker-state" && Number(fx.unitId) === Number(u.id)) node._state = fx.data;
      }
      node._time = ctx.time;
      ctx.seenGuides.add(id);
      draw(node._canvas, u, node._state, ctx);
    },
    tick(el, u, ctx) {
      if (!el) return;
      const st = document.getElementById(`flicker-state-${u.id}`)?._state;
      const phase = phaseAt(st, ctx.time || 0);
      el.classList.toggle("flicker-side", phase === 1);
      el.classList.toggle("flicker-focus", phase === 2);
      paintBody(el, phase, ctx.time || 0, u.radius || 18, ctx.scale || 1);
    },
    unmount(el) {
      el?.classList.remove("flicker-side", "flicker-focus");
      el?.querySelector(":scope > canvas.flicker-glitch")?.remove();
    },
  };
})();
