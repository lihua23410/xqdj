// All floor geometry comes from the match. The renderer never advances combat.
(function () {
  const hooks = window.lookFX = window.lookFX || {};
  const base = document.currentScript?.dataset.pack || "/ball/外科手术";
  const voices = new Map(), lastSound = new Map();
  let soundTime = -Infinity;
  const sounds = { "surgery-cut": ["cut.wav", .12], "surgery-stitch": ["stitch.wav", .1], "surgery-burst": ["burst.wav", .2] };
  for (const [name, [file, gap]] of Object.entries(sounds)) {
    window.arena.registerShot(name, (fx, ctx = {}) => {
      if (ctx.visualOnly) return;
      const time = ctx.time || 0;
      if (time < soundTime) { for (const pool of voices.values()) for (const v of pool) { v.stop(); v.reset(); } lastSound.clear(); }
      soundTime = time;
      if (time - (lastSound.get(name) ?? -Infinity) < gap) return;
      lastSound.set(name, time);
      let pool = voices.get(name);
      if (!pool) {
        pool = Array.from({ length: 3 }, () => {
          const audio = new Audio(`${base}/fx/${file}`); audio.preload = "auto";
          return window.arena.voice(audio, { mode: window.arena.SOUND_MODE.NORMAL, priority: 215, volume: 1 });
        });
        pool.next = 0; voices.set(name, pool);
      }
      const voice = pool[pool.next++ % pool.length]; voice.stop(); voice.reset(); voice.play();
    });
  }
  window.arena.registerShot("surgery-state", () => {});

  // Surgical cap, pleated mask and gown seams stay inside the circular body.
  // There is no separate head, face, torso, arm or leg silhouette.
  const body = `<svg viewBox="0 0 100 100" aria-hidden="true">
    <defs><clipPath id="surgery-disc"><circle cx="50" cy="50" r="48"/></clipPath></defs>
    <g clip-path="url(#surgery-disc)">
      <circle cx="50" cy="50" r="48" fill="#76b5ab"/>
      <path d="M4 35Q10 1 50 2 92 3 97 35L83 42 17 41Z" fill="#4e9b8e"/>
      <path d="M10 30Q49 18 91 30M25 9l5 15M61 4l-2 18M83 14l-5 12" fill="none" stroke="#b3dbcb" stroke-width="1.7" opacity=".8"/>
      <path d="M7 35Q49 27 94 35" fill="none" stroke="#255d58" stroke-width="3"/>
      <path d="M16 42 3 39M84 42l13-3M16 68 3 75M84 68l13 7" stroke="#deeeeb" stroke-width="3" fill="none"/>
      <path d="M15 40Q50 34 85 40L82 68Q50 83 18 68Z" fill="#c0e0e3" stroke="#537e87" stroke-width="1.4"/>
      <path d="M22 47Q51 42 79 47M22 55Q51 51 79 55M23 63Q50 61 77 63" fill="none" stroke="#74a9b5" stroke-width="2"/>
      <path d="M20 42Q50 37 80 42M24 49l49-1M24 57l49 1" fill="none" stroke="#ecf6f2" stroke-width="1.1"/>
      <path d="m26 81 24 14 24-14M50 95v7M17 87l5 7m56-7-5 10" fill="none" stroke="#245e59" stroke-width="2"/>
      <path d="m29 82 21 9 21-9" fill="none" stroke="#b1d7c9" stroke-width="1.1"/>
    </g><circle cx="50" cy="50" r="48" fill="none" stroke="#cee9df" stroke-width="1.5"/>
  </svg><span class="surgery-meter" role="meter" aria-label="大出血进度" aria-valuemin="0" aria-valuemax="200"><i></i></span>`;
  const scalpel = `<svg viewBox="-60 -10 72 20" aria-hidden="true">
    <path d="M-56-3-18-3-10-1-10 2-55 3Q-60 3-60 0T-56-3Z" fill="#9aaeb4" stroke="#e8f0eb" stroke-width=".7"/>
    <path d="M-53-1h24M-50 1h17" stroke="#475e65" stroke-width=".6"/>
    <path d="M-16-2-1-4 10-1Q7 5-8 4L-16 2Z" fill="#cbd9dd" stroke="#576c74" stroke-width=".8"/>
    <path d="M-10 3Q3 5 10-1" fill="none" stroke="#f5fcfa" stroke-width="1.2"/>
    <path d="M-25-2v4m-3-4v4m-3-4v4m-3-4v4m-3-4v4m-3-4v4" stroke="#41575e" stroke-width=".5"/>
  </svg>`;
  const needle = `<svg viewBox="0 0 100 100" aria-hidden="true">
    <path d="M69 46 82 50 69 54M70 47l-9-5m9 11-9 5" stroke="#718d90" stroke-width="1.4" fill="none"/>
    <path d="M82 50C78 34 99 31 98 46 98 52 94 57 89 58" fill="none" stroke="#f1f3ec" stroke-width="1.6"/>
    <path d="M89 58q-2 15-17 8t-18 8" fill="none" stroke="#202d30" stroke-width=".8"/>
  </svg>`;

  function mount(el, cls, html) {
    let node = el.querySelector(`:scope > .${cls}`);
    if (!node) { node = document.createElement("span"); node.className = cls; node.innerHTML = html; el.appendChild(node); }
    return node;
  }
  function direction(u) { return Math.atan2(-(u.vy || u.faceY || 0), u.vx || u.faceX || 1); }
  function state(id) { return document.getElementById(`surgery-floor-${id}`)?._surgery; }
  function hash(a, b) { let n = Math.imul(a + 131, 374761393) ^ Math.imul(b + 31, 668265263); return ((Math.imul(n ^ n >>> 13, 1274126177) ^ n >>> 16) >>> 0) / 4294967295; }
  function at(w, t) { return { x: w.a.x + (w.b.x - w.a.x) * t, y: w.a.y + (w.b.y - w.a.y) * t }; }
  function closedAt(w, t) { return w.closed[Math.min(w.closed.length - 1, Math.floor(t * w.closed.length))]; }

  function incision(c, w) {
    const drawn = Math.max(0, Math.min(1, w.drawn));
    if (!drawn) return;
    const dx = w.b.x - w.a.x, dy = w.b.y - w.a.y, len = Math.hypot(dx, dy);
    if (!len) return;
    const nx = -dy / len, ny = dx / len;
    const n = Math.max(12, Math.ceil(len / 4));
    const width = t => {
      const taper = Math.pow(Math.max(0, Math.sin(Math.PI * t / drawn)), .48);
      return (closedAt(w, Math.min(.9999, t)) ? .65 : 5.5 + Math.sin(t * 29 + w.id) * .45) * taper;
    };
    function ribbon(mult, fill) {
      c.beginPath();
      for (let side = 1; side >= -1; side -= 2) {
        for (let j = 0; j <= n; j++) {
          const t = drawn * (side > 0 ? j : n - j) / n, p = at(w, t), r = width(t) * mult;
          const x = p.x + nx * r * side, y = p.y + ny * r * side;
          if (side === 1 && j === 0) c.moveTo(x, y); else c.lineTo(x, y);
        }
      }
      c.closePath(); c.fillStyle = fill; c.fill();
    }
    ribbon(1.6, "rgba(83,22,37,.36)");
    ribbon(1.23, "#c27f79");
    ribbon(1, "#982b3f");
    ribbon(.67, "#551528");
    ribbon(.23, "#2a101c");
    for (const side of [-1, 1]) {
      c.beginPath();
      for (let i = 0; i <= n; i++) {
        const t = drawn * i / n, p = at(w, t), r = width(t) * 1.11;
        const x = p.x + nx * r * side, y = p.y + ny * r * side;
        if (!i) c.moveTo(x, y); else c.lineTo(x, y);
      }
      c.strokeStyle = side > 0 ? "#e4b6a3" : "#9b5e59"; c.lineWidth = .75; c.stroke();
    }
    for (let i = 0; i < w.points.length; i++) {
      const p = w.points[i], t = i / (w.points.length - 1);
      if (t > drawn) continue;
      const pierced = (i && w.closed[i - 1]) || w.closed[i];
      if (pierced) {
        c.fillStyle = "#44212b"; c.beginPath(); c.ellipse(p.x, p.y, 1.1, .7, 0, 0, Math.PI * 2); c.fill();
      }
      if (i && w.closed[i - 1]) {
        const q = w.points[i - 1];
        c.beginPath(); c.moveTo(q.x, q.y); c.lineTo(p.x, p.y); c.strokeStyle = "#161e22"; c.lineWidth = 1.1; c.stroke();
        c.strokeStyle = "#bbc4b9"; c.lineWidth = .3; c.stroke();
      }
    }
    if (w.bleeding) {
      for (let i = 0; i < 18; i++) {
        const t = hash(w.id, i), p = at(w, t);
        if (closedAt(w, t)) continue;
        const spread = (hash(i, w.id + 31) * 2 - 1) * 16;
        c.fillStyle = "rgba(104,13,35,.45)";
        c.beginPath(); c.ellipse(p.x + nx * spread, p.y + ny * spread, 1 + hash(i, 7) * 2, .7 + hash(i, 11), 0, 0, Math.PI * 2); c.fill();
      }
    }
  }

  function paintFloor(st, ctx) {
    const data = st.data;
    if (!data) return;
    const pts = (data.wounds || []).flatMap(w => [w.a, w.b, ...w.points]).concat(data.warning || []);
    if (!pts.length) { st.el.style.display = "none"; return; }
    st.el.style.display = "block";
    const minX = Math.min(...pts.map(p => p.x)) - 20, maxX = Math.max(...pts.map(p => p.x)) + 20;
    const minY = Math.min(...pts.map(p => p.y)) - 20, maxY = Math.max(...pts.map(p => p.y)) + 20;
    const width = maxX - minX, height = maxY - minY, dpr = Math.min(2, 3000 / Math.max(width, height));
    const canvas = st.canvas, cw = Math.ceil(width * dpr), ch = Math.ceil(height * dpr);
    if (canvas.width !== cw || canvas.height !== ch) { canvas.width = cw; canvas.height = ch; }
    const c = canvas.getContext("2d"); c.setTransform(1, 0, 0, 1, 0, 0); c.clearRect(0, 0, cw, ch);
    c.setTransform(dpr, 0, 0, -dpr, -minX * dpr, maxY * dpr);
    for (const w of data.wounds || []) incision(c, w);
    if (data.warning) {
      const [a, b] = data.warning;
      c.beginPath(); c.moveTo(a.x, a.y); c.lineTo(b.x, b.y);
      c.strokeStyle = "rgba(223,228,209,.8)"; c.lineWidth = 1; c.setLineDash([5, 5]); c.stroke(); c.setLineDash([]);
      c.beginPath(); c.arc(a.x, a.y, 8, 0, Math.PI * 2); c.strokeStyle = "#d2e6df"; c.stroke();
    }
    Object.assign(st.el.style, { left: `${ctx.cx + minX * ctx.scale}px`, top: `${ctx.cy - maxY * ctx.scale}px`, width: `${width * ctx.scale}px`, height: `${height * ctx.scale}px` });
  }

  hooks.surgery = {
    guide(u, ctx) {
      const id = `surgery-floor-${u.id}`, el = ctx.ensureGuide(id, "surgery-floor"); ctx.seenGuides.add(id);
      let st = el._surgery;
      if (!st) { const canvas = document.createElement("canvas"); el.appendChild(canvas); st = el._surgery = { el, canvas, time: -Infinity }; }
      if (ctx.time < st.time) st.data = null;
      for (const fx of ctx.effects || []) if (fx.name === "surgery-state" && fx.unitId === u.id) st.data = fx.data;
      st.time = ctx.time; paintFloor(st, ctx);
    },
    tick(el, u) {
      const art = mount(el, "surgery-body", body), data = state(u.id)?.data;
      const capacity = data?.capacity || 200;
      const charge = Math.max(0, Math.min(capacity, data?.charge || 0));
      art.style.setProperty("--surgery-charge", `${charge / capacity * 100}%`);
      art.querySelector(".surgery-meter").setAttribute("aria-valuemax", capacity);
      art.querySelector(".surgery-meter").setAttribute("aria-valuenow", charge.toFixed(0));
      el.classList.toggle("surgery-immune", !!data?.immune);
    },
    unmount(el) { el.querySelector(":scope > .surgery-body")?.remove(); el.classList.remove("surgery-immune"); },
  };
  hooks["surgery-scalpel"] = { tick(el, u) { mount(el, "surgery-scalpel-art", scalpel).style.transform = `rotate(${direction(u)}rad)`; }, unmount(el) { el.querySelector(".surgery-scalpel-art")?.remove(); } };
  hooks["surgery-needle"] = { tick(el, u) { mount(el, "surgery-needle-art", needle).style.transform = `rotate(${direction(u)}rad)`; }, unmount(el) { el.querySelector(".surgery-needle-art")?.remove(); } };
  hooks["surgery-blood"] = { tick(el, u) { mount(el, "surgery-blood-art", "").style.transform = `rotate(${direction(u)}rad)`; }, unmount(el) { el.querySelector(".surgery-blood-art")?.remove(); } };
})();
