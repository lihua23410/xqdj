window.lookFX = window.lookFX || {};
window.integratedTrack = window.integratedTrack || {};

const nearRadius = 185;
const fleeRadius = 100;
const fleeTrailAt = 148;
const trackSeconds = 5;
const arcFull = 270;
const ringWas = new Map();

function applyTrackFX(into, fx, time) {
  if (!fx || fx.name !== "track" || !fx.unitId) return;
  if (fx.amount > 0) {
    const remain = fx.vy > 0 ? fx.vy : trackSeconds;
    into[fx.unitId] = { id: fx.vx, until: (time || 0) + remain };
    return;
  }
  delete into[fx.unitId];
}

function syncTracks(ctx) {
  if (ctx._trackSynced) return;
  ctx._trackSynced = true;
  const next = {};
  for (const fx of ctx.effects || []) applyTrackFX(next, fx, ctx.time);
  window.integratedTrack = next;
}

function trackOf(id) {
  const t = window.integratedTrack[id];
  if (!t || typeof t !== "object") return null;
  return t;
}

function placeRing(ctx, u, id, cls, radius) {
  const g = ctx.ensureGuide(id, cls);
  const d = radius * 2 * ctx.scale;
  g.style.width = `${d}px`;
  g.style.height = `${d}px`;
  g.style.left = `${ctx.cx + u.x * ctx.scale}px`;
  g.style.top = `${ctx.cy - u.y * ctx.scale}px`;
  const engaged = (ctx.units || []).some((o) => {
    if (!o || o.id === u.id || o.slot === u.slot) return false;
    if (o.role !== "fighter" && !o.mortal) return false;
    return Math.hypot(o.x - u.x, o.y - u.y) <= radius;
  });
  const prev = ringWas.get(id);
  g.classList.toggle("alert", engaged);
  if (prev === false && engaged) {
    g.classList.remove("flash");
    void g.offsetWidth;
    g.classList.add("flash");
  }
  ringWas.set(id, engaged);
  ensureTicks(g);
  if (ctx.seenGuides) ctx.seenGuides.add(g.id);
}

function ensureTicks(g) {
  if (g.querySelector(":scope > .ticks")) return;
  const svg = document.createElementNS("http://www.w3.org/2000/svg", "svg");
  svg.setAttribute("class", "ticks");
  svg.setAttribute("viewBox", "-100 -100 200 200");
  const minor = document.createElementNS("http://www.w3.org/2000/svg", "circle");
  minor.setAttribute("class", "minor");
  minor.setAttribute("r", "98");
  minor.setAttribute("stroke-dasharray", "1.3 24.8");
  const major = document.createElementNS("http://www.w3.org/2000/svg", "circle");
  major.setAttribute("class", "major");
  major.setAttribute("r", "96");
  major.setAttribute("stroke-dasharray", "1.8 73.7");
  for (const c of [minor, major]) {
    c.setAttribute("cx", "0");
    c.setAttribute("cy", "0");
  }
  svg.append(minor, major);
  g.appendChild(svg);
}

function placeArc(ctx, u) {
  const lock = trackOf(u.id);
  if (!lock) return;
  const remain = (lock.until - (ctx.time || 0)) / trackSeconds;
  if (remain <= 0) return;
  const target = (ctx.units || []).find((o) => o.id === lock.id);
  if (!target) return;
  const m = ctx.ensureGuide(`guide-mark-${u.id}`, "integrated-mark");
  const span = arcFull * Math.min(remain, 1);
  const mark = Math.max(target.radius || 18, 8) * 3.2 * 2 * ctx.scale;
  m.style.width = `${mark}px`;
  m.style.height = `${mark}px`;
  m.style.left = `${ctx.cx + target.x * ctx.scale}px`;
  m.style.top = `${ctx.cy - target.y * ctx.scale}px`;
  m.style.setProperty("--arc", `${span}deg`);
  if (ctx.seenGuides) ctx.seenGuides.add(m.id);
}

window.lookFX.integrated = {
  tick(el, u, ctx) {
    if (!el || !u || !ctx || !ctx.spawnGhost) return;
    if (Math.hypot(u.vx || 0, u.vy || 0) <= fleeTrailAt) return;
    const now = ctx.now || 0;
    if (now - (el._fleeAt || 0) < 16) return;
    el._fleeAt = now;
    const g = ctx.spawnGhost(el, u.kind, ctx.fxRoot, "trail");
    if (g) g.classList.add("integrated-ghost");
  },
  guide(u, ctx) {
    if (!u || u.role !== "fighter" || !ctx || !ctx.ensureGuide) return;
    syncTracks(ctx);
    sweepTrails(ctx);
    placeRing(ctx, u, `guide-near-${u.id}`, "integrated-ring", nearRadius);
    placeRing(ctx, u, `guide-flee-${u.id}`, "integrated-flee", fleeRadius);
    placeArc(ctx, u);
  },
};

const shotPaths = new Map();

function ensureShot(el) {
  let shape = el.querySelector(":scope > .integrated-shot-shape");
  if (shape) return shape;
  shape = document.createElement("div");
  shape.className = "integrated-shot-shape";
  const nose = document.createElement("i");
  nose.className = "nose";
  shape.append(nose);
  el.appendChild(shape);
  return shape;
}

function trailLayer(ctx) {
  const root = ctx && ctx.fxRoot;
  if (!root) return null;
  let svg = root.querySelector(":scope > .integrated-trails");
  if (svg) return svg;
  svg = document.createElementNS("http://www.w3.org/2000/svg", "svg");
  svg.setAttribute("class", "integrated-trails");
  root.appendChild(svg);
  return svg;
}

function orangeAt(t) {
  const from = [168, 72, 16];
  const to = [240, 160, 80];
  const c = from.map((v, i) => Math.round(v + (to[i] - v) * t));
  return `rgb(${c[0]}, ${c[1]}, ${c[2]})`;
}

function dropTrail(id) {
  shotPaths.delete(id);
  document.querySelector(`#itrail-${id}`)?.remove();
}

function sweepTrails(ctx) {
  if (!ctx || ctx._trailSwept) return;
  ctx._trailSwept = true;
  const alive = new Set((ctx.units || []).map((o) => o.id));
  for (const id of shotPaths.keys()) {
    if (!alive.has(id)) dropTrail(id);
  }
}

function paintTrail(el, u, ctx, tracking) {
  if (!tracking) {
    dropTrail(u.id);
    return;
  }
  const svg = trailLayer(ctx);
  if (!svg) return;
  const x = parseFloat(el.style.left);
  const y = parseFloat(el.style.top);
  if (!Number.isFinite(x) || !Number.isFinite(y)) return;
  const ang = parseFloat(el.style.getPropertyValue("--shot-ang")) || 0;
  const back = parseFloat(el.style.getPropertyValue("--shot-r")) || 6;
  const hx = x - Math.cos(ang) * back * 0.85;
  const hy = y - Math.sin(ang) * back * 0.85;
  let pts = shotPaths.get(u.id);
  if (!pts) pts = [];
  const last = pts[pts.length - 1];
  if (!last || Math.hypot(hx - last.x, hy - last.y) >= 2.5) pts.push({ x: hx, y: hy });
  if (pts.length > 18) pts.splice(0, pts.length - 18);
  shotPaths.set(u.id, pts);
  let g = svg.querySelector(`#itrail-${u.id}`);
  if (pts.length < 2) {
    g?.remove();
    return;
  }
  if (!g) {
    g = document.createElementNS("http://www.w3.org/2000/svg", "g");
    g.id = `itrail-${u.id}`;
    svg.appendChild(g);
  }
  g.replaceChildren();
  for (let i = 1; i < pts.length; i++) {
    const seg = document.createElementNS("http://www.w3.org/2000/svg", "line");
    const t = i / (pts.length - 1);
    const a = pts[i - 1];
    const b = pts[i];
    seg.setAttribute("x1", a.x);
    seg.setAttribute("y1", a.y);
    seg.setAttribute("x2", b.x);
    seg.setAttribute("y2", b.y);
    seg.setAttribute("stroke", orangeAt(t));
    seg.setAttribute("stroke-opacity", String(0.05 + 0.52 * t));
    seg.setAttribute("stroke-width", "1.4");
    seg.setAttribute("stroke-linecap", "round");
    g.appendChild(seg);
  }
}

window.lookFX["integrated-shot"] = {
  tick(el, u, ctx) {
    if (!el || !u) return;
    ensureShot(el);
    const r = (u.radius || 6) * (ctx && ctx.scale ? ctx.scale : 1);
    el.style.setProperty("--shot-r", `${r}px`);
    const vx = u.vx || 0;
    const vy = u.vy || 0;
    if (vx * vx + vy * vy > 1e-6) {
      el.style.setProperty("--shot-ang", `${Math.atan2(-vy, vx)}rad`);
    }
    const owner = trackOf(u.ownerId);
    const tracking = !!(owner && (ctx.units || []).some((o) => o.id === u.ownerId));
    el.classList.toggle("tracking", tracking);
    sweepTrails(ctx);
    paintTrail(el, u, ctx, tracking);
  },
  unmount(el) {
    el.querySelector(":scope > .integrated-shot-shape")?.remove();
    el.classList.remove("tracking");
    const id = Number(String(el.id || "").replace(/^u-/, ""));
    if (id) dropTrail(id);
  },
};

arena.registerShot("track", (fx, ctx) => {
  applyTrackFX(window.integratedTrack, fx, ctx && ctx.time);
});
