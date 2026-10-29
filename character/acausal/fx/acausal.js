window.lookFX = window.lookFX || {};
window.acausalSpec = window.acausalSpec || { bins: [], pick: -1, n: 48 };
window.acausalDraws = window.acausalDraws || {};

function acausalSourceKind(u) {
  for (const m of (u && u.marks) || []) {
    if (!m || !m.kind || m.kind === "非因果") continue;
    const look = lookOf(m.kind);
    if (look && (look.color || (look.fx && look.fx.length))) return m.kind;
  }
  return "";
}

function acausalWear(el, u, ctx) {
  const kind = acausalSourceKind(u) || "非因果";
  paintKind(el, kind);
  for (const name of lookOf(kind).fx || []) {
    const id = fxID(name);
    if (!id || id === "acausal" || id === "acausal-ghost") continue;
    el.classList.add("look-" + id);
    const hook = window.lookFX && window.lookFX[id];
    if (hook && typeof hook.tick === "function") hook.tick(el, Object.assign({}, u, { kind }), ctx);
  }
  el.style.opacity = "0.32";
}

function paintGhostHP(u, ctx) {
  for (const g of ctx.units || []) {
    if (g.ownerId !== u.id) continue;
    if (g.kind !== "非因果战斗机影" && g.kind !== "非因果活随从影" && g.kind !== "非因果自身影") continue;
    const el = ctx.ensureGuide("acausal-hp-" + g.id, "hp-float");
    if (ctx.seenGuides) ctx.seenGuides.add(el.id);
    arena.drawHealthNumber(el, g, ctx);
  }
}

function paintAcausalChart(u, ctx) {
  const el = ctx.ensureGuide("acausal-spec-" + u.id, "acausal-spec");
  if (ctx.seenGuides) ctx.seenGuides.add(el.id);
  const bins = window.acausalSpec.bins || [];
  const n = bins.length || 1;
  const w = 112;
  const plot = 28;
  const h = 42;
  const base = plot - 1;
  let max = 0;
  for (const v of bins) if (v > max) max = v;
  let d = `M 0 ${base}`;
  for (let i = 0; i < n; i++) {
    const x = n <= 1 ? 0 : (i / (n - 1)) * w;
    const y = max > 0 ? base - ((bins[i] || 0) / max) * (plot - 6) : base;
    d += ` L ${x.toFixed(2)} ${y.toFixed(2)}`;
  }
  d += ` L ${w} ${base} Z`;
  const amt = Number(window.acausalSpec.pickAmt) || 0;
  let mark = "";
  if (window.acausalSpec.pick >= 0 && max > 0) {
    const x = Math.max(0, Math.min(w, (amt / max) * w));
    mark = `<line x1="${x.toFixed(2)}" y1="1" x2="${x.toFixed(2)}" y2="${base}" stroke="#fff" stroke-width="1.4"/>`;
  }
  const right = max >= 10 ? String(Math.round(max)) : max.toFixed(1);
  el.innerHTML =
    `<svg viewBox="0 0 ${w} ${h}" width="${w}" height="${h}">` +
    `<path d="${d}" fill="rgba(215,196,255,0.35)" stroke="#efe7ff" stroke-width="1.2"/>` +
    mark +
    `<text x="0" y="${h - 1}" fill="#efe7ff" font-size="9">0</text>` +
    `<text x="${w}" y="${h - 1}" fill="#efe7ff" font-size="9" text-anchor="end">${right}</text>` +
    `</svg>`;
  const r = (u.radius || 18) * ctx.scale;
  const [sx, sy] = screenPos(u.x, u.y, ctx.scale, ctx.cx, ctx.cy);
  el.style.left = `${sx}px`;
  el.style.top = `${sy - r - 8}px`;
}

window.lookFX.acausal = {
  unmount(el) {
    document.getElementById("acausal-spec")?.remove();
    window.acausalSpec = { bins: [], pick: -1, n: 48 };
    el?.querySelector(":scope > canvas.acausal-skin")?.remove();
  },
  tick(el) {
    if (!el) return;
    const canvas = ensureAcausalCanvas(el, "acausal-skin", 1.65, 36);
    paintAcausalOrb(canvas, (performance.now() / 5000) * Math.PI * 2);
  },
  guide(u, ctx) {
    const spec = window.acausalSpec;
    for (const fx of ctx.effects || []) {
      if (!fx || fx.kind !== "非因果") continue;
      if (fx.name === "spec") {
        const i = fx.x | 0;
        spec.n = fx.y | 0 || spec.n;
        spec.bins[i] = fx.amount;
      }
      if (fx.name === "pick") {
        spec.pick = fx.x | 0;
        spec.pickAmt = fx.amount;
        spec.pending = { bin: spec.pick, amt: fx.amount, n: spec.n || 48 };
      }
    }
    paintAcausalChart(u, ctx);
    paintGhostHP(u, ctx);
    paintBladeTrails(u, ctx);
    const shot = (ctx.units || []).find((o) => o.kind === "非因果锁影弹" && o.ownerId === u.id);
    if (shot && spec.pending) window.acausalDraws[shot.id] = spec.pending;
    const ghosts = (ctx.units || []).filter((o) => o.kind === "非因果战斗机影" && o.ownerId === u.id);
    if (!shot || !ghosts.length) return;
    let best = ghosts[0];
    let bestD = Infinity;
    for (const g of ghosts) {
      const d = Math.hypot(g.x - shot.x, g.y - shot.y);
      if (d < bestD) {
        best = g;
        bestD = d;
      }
    }
    const line = ctx.ensureGuide("acausal-lock-" + u.id, "acausal-lock");
    line.className = "guide acausal-lock";
    if (ctx.seenGuides) ctx.seenGuides.add(line.id);
    const [x1, y1] = screenPos(shot.x, shot.y, ctx.scale, ctx.cx, ctx.cy);
    const [x2, y2] = screenPos(best.x, best.y, ctx.scale, ctx.cx, ctx.cy);
    const shift = ((performance.now() / 28) % 16).toFixed(1);
    const gid = "acausal-lock-grad-" + u.id;
    line.innerHTML =
      `<svg><defs><linearGradient id="${gid}" gradientUnits="userSpaceOnUse" x1="${x1.toFixed(1)}" y1="${y1.toFixed(1)}" x2="${x2.toFixed(1)}" y2="${y2.toFixed(1)}">` +
      `<stop offset="0" stop-color="#f6f1ff" stop-opacity="0"/>` +
      `<stop offset="0.14" stop-color="#f6f1ff" stop-opacity="0.95"/>` +
      `<stop offset="0.86" stop-color="#d7c4ff" stop-opacity="0.75"/>` +
      `<stop offset="1" stop-color="#d7c4ff" stop-opacity="0"/>` +
      `</linearGradient></defs>` +
      `<line x1="${x1.toFixed(1)}" y1="${y1.toFixed(1)}" x2="${x2.toFixed(1)}" y2="${y2.toFixed(1)}" stroke="url(#${gid})" stroke-width="5" stroke-linecap="round" stroke-dasharray="2 14" stroke-dashoffset="-${shift}" opacity="0.45"/>` +
      `<line x1="${x1.toFixed(1)}" y1="${y1.toFixed(1)}" x2="${x2.toFixed(1)}" y2="${y2.toFixed(1)}" stroke="url(#${gid})" stroke-width="1.6" stroke-linecap="round" stroke-dasharray="6 10" stroke-dashoffset="-${shift}"/>` +
      `</svg>`;
  },
};

window.acausalTrails = window.acausalTrails || {};

function pushBladePoint(o) {
  const id = String(o.id);
  let tr = window.acausalTrails[id];
  if (!tr) tr = window.acausalTrails[id] = { pts: [] };
  tr.alive = true;
  tr.fadeStart = 0;
  const last = tr.pts[tr.pts.length - 1];
  if (!last || Math.hypot(o.x - last.x, o.y - last.y) > 5) {
    tr.pts.push({ x: o.x, y: o.y });
    if (tr.pts.length > 42) tr.pts.shift();
  }
}

function paintBladeTrails(owner, ctx) {
  const live = new Set();
  for (const o of ctx.units || []) {
    if (o.kind === "非因果锁影弹" && o.ownerId === owner.id) {
      live.add(String(o.id));
      pushBladePoint(o);
    }
  }
  const now = performance.now();
  for (const id of Object.keys(window.acausalTrails)) {
    const tr = window.acausalTrails[id];
    if (!live.has(id) && !tr.fadeStart) tr.fadeStart = now;
    const wipe = tr.fadeStart ? Math.min(1.2, (now - tr.fadeStart) / 900) : 0;
    const el = ctx.ensureGuide("acausal-trail-" + id, "acausal-trail");
    if (ctx.seenGuides) ctx.seenGuides.add(el.id);
    const pts = tr.pts;
    if (pts.length < 2 || wipe >= 1.05) {
      el.innerHTML = "";
      if (wipe >= 1.05) delete window.acausalTrails[id];
      continue;
    }
    const segs = [];
    for (let i = 1; i < pts.length; i++) {
      const pos = i / (pts.length - 1);
      let alpha = 0.62;
      if (wipe > 0) {
        if (pos < wipe - 0.16) continue;
        if (pos < wipe) alpha = ((pos - (wipe - 0.16)) / 0.16) * 0.62;
      }
      const [x1, y1] = screenPos(pts[i - 1].x, pts[i - 1].y, ctx.scale, ctx.cx, ctx.cy);
      const [x2, y2] = screenPos(pts[i].x, pts[i].y, ctx.scale, ctx.cx, ctx.cy);
      segs.push(
        `<line x1="${x1.toFixed(1)}" y1="${y1.toFixed(1)}" x2="${x2.toFixed(1)}" y2="${y2.toFixed(1)}" stroke="rgba(232,214,255,${alpha.toFixed(3)})" stroke-width="2" stroke-linecap="butt"/>`,
      );
    }
    el.innerHTML = `<svg>${segs.join("")}</svg>`;
  }
}

const acausalOrbStrands = (() => {
  const n = 36;
  const turns = 2;
  const strands = [[], []];
  for (let s = 0; s < 2; s++) {
    const pts = strands[s];
    for (let i = 0; i < n; i++) {
      const t = i / (n - 1);
      const lat = (-70 + 140 * t) * Math.PI / 180;
      const lon = turns * Math.PI * 2 * t + s * Math.PI;
      const cl = Math.cos(lat);
      pts.push(cl * Math.sin(lon), Math.sin(lat), cl * Math.cos(lon));
    }
  }
  return strands;
})();

const acausalOrbGlow = ["246,241,255", "143,104,216"].map((rgb) => {
  const c = document.createElement("canvas");
  c.width = c.height = 32;
  const g = c.getContext("2d");
  const rad = g.createRadialGradient(16, 16, 0, 16, 16, 16);
  rad.addColorStop(0, `rgba(${rgb},1)`);
  rad.addColorStop(0.4, `rgba(${rgb},0.55)`);
  rad.addColorStop(1, `rgba(${rgb},0)`);
  g.fillStyle = rad;
  g.fillRect(0, 0, 32, 32);
  return c;
});

function ensureAcausalCanvas(el, cls, mul, fallback) {
  let canvas = el.querySelector(":scope > canvas." + cls);
  if (!canvas) {
    canvas = document.createElement("canvas");
    canvas.className = cls;
    canvas._dpr = Math.min(2, window.devicePixelRatio || 1);
    el.appendChild(canvas);
    el.style.background = "transparent";
    el.style.boxShadow = "none";
  }
  const size = Math.max(24, Math.round((el.offsetWidth || fallback) * mul));
  if (canvas._css !== size) {
    canvas._css = size;
    canvas.style.width = size + "px";
    canvas.style.height = size + "px";
    const dpr = canvas._dpr;
    canvas.width = Math.round(size * dpr);
    canvas.height = Math.round(size * dpr);
  }
  return canvas;
}

function paintAcausalOrb(canvas, ang) {
  const ctx = canvas.getContext("2d");
  const dpr = canvas._dpr;
  const size = canvas._css || 56;
  ctx.setTransform(dpr, 0, 0, dpr, 0, 0);
  ctx.clearRect(0, 0, size, size);
  const tilt = 22 * Math.PI / 180;
  const cT = Math.cos(tilt);
  const sT = Math.sin(tilt);
  const cA = Math.cos(ang);
  const sA = Math.sin(ang);
  const R = size * 0.3;
  const cx = size / 2;
  const cy = size / 2;
  for (let s = 0; s < 2; s++) {
    const src = acausalOrbStrands[s];
    const pts = [];
    for (let i = 0; i < src.length; i += 3) {
      const x0 = src[i];
      const y0 = src[i + 1];
      const z0 = src[i + 2];
      const x = x0 * cA + z0 * sA;
      const z = -x0 * sA + z0 * cA;
      const y2 = y0 * cT - z * sT;
      const z2 = y0 * sT + z * cT;
      pts.push(cx + x * R, cy - y2 * R, z2);
    }
    const n = pts.length / 3;
    const order = [];
    for (let i = 0; i < n; i++) order.push(i);
    order.sort((a, b) => pts[a * 3 + 2] - pts[b * 3 + 2]);
    const sprite = acausalOrbGlow[s];
    for (const i of order) {
      const z = pts[i * 3 + 2];
      const front = (z + 1) * 0.5;
      const dot = 8 + front * 8;
      ctx.globalAlpha = 0.4 + 0.6 * front;
      ctx.drawImage(sprite, pts[i * 3] - dot / 2, pts[i * 3 + 1] - dot / 2, dot, dot);
    }
  }
  ctx.globalAlpha = 1;
}

// 锁影弹是电弧球：弧从核伸到玻璃壳。样子参考 pattle/plasma-ball
// https://github.com/pattle/plasma-ball ，这里不跑那套粒子，只画五条弧。
function plasmaBolts(canvas) {
  if (canvas._bolts) return canvas._bolts;
  const bolts = [];
  for (let i = 0; i < 5; i++) {
    bolts.push({
      ang: (i / 5) * Math.PI * 2,
      spin: (i % 2 ? 0.9 : -0.7) * (0.8 + (i % 3) * 0.15),
      phase: i * 1.37,
    });
  }
  canvas._bolts = bolts;
  return bolts;
}

function paintAcausalPlasma(canvas, t) {
  const ctx = canvas.getContext("2d");
  const dpr = canvas._dpr;
  const size = canvas._css || 32;
  ctx.setTransform(dpr, 0, 0, dpr, 0, 0);
  ctx.clearRect(0, 0, size, size);
  const cx = size / 2;
  const cy = size / 2;
  const R = size * 0.36;
  const bolts = plasmaBolts(canvas);
  ctx.lineCap = "round";
  for (let i = 0; i < bolts.length; i++) {
    const b = bolts[i];
    const ang = b.ang + t * b.spin;
    const wob = Math.sin(t * 2.4 + b.phase) * 0.7;
    const x1 = cx + Math.cos(ang) * R * 0.16;
    const y1 = cy + Math.sin(ang) * R * 0.16;
    const x2 = cx + Math.cos(ang + wob * 0.35) * R;
    const y2 = cy + Math.sin(ang + wob * 0.35) * R;
    const nx = Math.cos(ang + Math.PI / 2);
    const ny = Math.sin(ang + Math.PI / 2);
    const mx = cx + Math.cos(ang) * R * 0.55 + nx * wob * R * 0.28;
    const my = cy + Math.sin(ang) * R * 0.55 + ny * wob * R * 0.28;
    const rgb = i % 2 ? "143,104,216" : "246,241,255";
    ctx.lineWidth = Math.max(1.2, size * 0.045);
    ctx.strokeStyle = `rgba(${rgb},0.9)`;
    ctx.beginPath();
    ctx.moveTo(x1, y1);
    ctx.quadraticCurveTo(mx, my, x2, y2);
    ctx.stroke();
    const sprite = acausalOrbGlow[i % 2];
    const tip = Math.max(8, size * 0.28);
    ctx.globalAlpha = 0.9;
    ctx.drawImage(sprite, x2 - tip / 2, y2 - tip / 2, tip, tip);
  }
  const core = Math.max(10, size * 0.34);
  ctx.globalAlpha = 1;
  ctx.drawImage(acausalOrbGlow[0], cx - core / 2, cy - core / 2, core, core);
}

window.lookFX["acausal-shot"] = {
  tick(el) {
    const canvas = ensureAcausalCanvas(el, "acausal-bolt", 2.4, 12);
    paintAcausalPlasma(canvas, performance.now() / 1000);
  },
  unmount(el) {
    el.querySelector(":scope > canvas.acausal-bolt")?.remove();
    el.style.background = "";
    el.style.boxShadow = "";
  },
};

window.lookFX["acausal-ghost"] = {
  tick(el, u, ctx) {
    acausalWear(el, u, ctx);
  },
  unmount(el) {
    el.style.opacity = "";
  },
};
