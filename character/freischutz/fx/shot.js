// 时间拖尾：痕迹按存在时间消退，不按长度截断。
// 初速 520 在 0.12 秒里只走出大约 60，看起来短；速度翻倍后同一段时间覆盖的路变长，痕迹才明显。
window.lookFX = window.lookFX || {};

const freiTraceLife = 0.12;
const freiTraces = (window.freiTraces = window.freiTraces || {});

function freiRecord(u, time) {
  const id = String(u.id);
  let tr = freiTraces[id];
  if (!tr) tr = freiTraces[id] = { pts: [] };
  tr.seen = time;
  const last = tr.pts[tr.pts.length - 1];
  if (last && Math.hypot(u.x - last.x, u.y - last.y) < 3 && time - last.t < 0.02) return;
  tr.pts.push({ x: u.x, y: u.y, t: time });
}

function freiPaint(id, ctx) {
  const tr = freiTraces[id];
  if (!tr) return;
  const now = ctx.time || 0;
  const pts = tr.pts.filter((p) => now - p.t < freiTraceLife);
  tr.pts = pts;
  const el = ctx.ensureGuide("frei-trace-" + id, "frei-trace");
  if (ctx.seenGuides) ctx.seenGuides.add(el.id);
  if (pts.length < 2) {
    el.innerHTML = "";
    if (pts.length === 0) delete freiTraces[id];
    return;
  }
  const segs = [];
  for (let i = 1; i < pts.length; i++) {
    const age = Math.max(0, now - pts[i].t);
    const alpha = Math.max(0, 1 - age / freiTraceLife) * 0.9;
    if (alpha < 0.02) continue;
    const [x1, y1] = screenPos(pts[i - 1].x, pts[i - 1].y, ctx.scale, ctx.cx, ctx.cy);
    const [x2, y2] = screenPos(pts[i].x, pts[i].y, ctx.scale, ctx.cx, ctx.cy);
    segs.push(
      `<line x1="${x1.toFixed(1)}" y1="${y1.toFixed(1)}" x2="${x2.toFixed(1)}" y2="${y2.toFixed(1)}" stroke="rgba(142,197,255,${alpha.toFixed(3)})" stroke-width="2" stroke-linecap="butt"/>`,
    );
  }
  el.innerHTML = segs.length ? `<svg>${segs.join("")}</svg>` : "";
}

function freiAngle(u) {
  const vx = Number(u && u.vx) || 0;
  const vy = Number(u && u.vy) || 0;
  if (Math.hypot(vx, vy) < 1e-6) return 0;
  return (Math.atan2(-vy, vx) * 180) / Math.PI;
}

window.lookFX["freischutz-shot"] = {
  tick(el, u) {
    if (!el) return;
    const base = (window.arena && arena.lookOf && arena.lookOf(u.kind).base) || "";
    let img = el.querySelector(":scope > .frei-shot-img");
    if (!img) {
      img = document.createElement("img");
      img.className = "frei-shot-img";
      img.alt = "";
      el.appendChild(img);
    }
    const src = `${base}/fx/魔弹.png`;
    if (img.getAttribute("src") !== src) img.src = src;
    img.style.transform = `translate(-50%, -50%) rotate(${freiAngle(u)}deg)`;
  },
  unmount(el) {
    el?.querySelector(":scope > .frei-shot-img")?.remove();
  },
  guide(u, ctx) {
    freiRecord(u, ctx.time || 0);
    freiPaint(String(u.id), ctx);
  },
};

window.freiSt = window.freiSt || {};

function freiFaceDeg(vx, vy) {
  const x = Number(vx) || 0;
  const y = Number(vy) || 0;
  if (Math.hypot(x, y) < 1e-6) return null;
  return (Math.atan2(-y, x) * 180) / Math.PI;
}

window.lookFX.freischutz = window.lookFX.freischutz || {};
window.lookFX.freischutz.tick = function (el, u) {
  if (!el) return;
  const st = (window.freiSt[u.id] = window.freiSt[u.id] || { face: 0 });
  const moving = freiFaceDeg(u.vx, u.vy);
  if (moving !== null) st.face = moving;
  const base = (window.arena && arena.lookOf && arena.lookOf(u.kind).base) || "";
  let img = el.querySelector(":scope > .frei-gun");
  if (!img) {
    img = document.createElement("img");
    img.className = "frei-gun";
    img.alt = "";
    el.appendChild(img);
  }
  const src = `${base}/fx/枪.png`;
  if (img.getAttribute("src") !== src) img.src = src;
  img.style.transform = `translate(-50%, -50%) rotate(${Number.isFinite(st.face) ? st.face : 0}deg)`;
  el.classList.toggle("aiming", !!st.aim);
};
window.lookFX.freischutz.unmount = function (el) {
  el?.querySelector(":scope > .frei-gun")?.remove();
};
window.lookFX.freischutz.guide = function (_u, ctx) {
  const live = new Set((ctx.units || []).filter((o) => o.kind === "中转站[Freischutz]弹").map((o) => String(o.id)));
  for (const id of Object.keys(freiTraces)) {
    if (live.has(id)) continue;
    freiPaint(id, ctx);
  }
};

arena.registerShot("face", (fx) => {
  const st = (window.freiSt[fx.unitId] = window.freiSt[fx.unitId] || { face: 0 });
  const deg = freiFaceDeg(fx.vx, fx.vy);
  if (deg !== null) st.face = deg;
});

arena.registerShot("aim", (fx) => {
  const st = (window.freiSt[fx.unitId] = window.freiSt[fx.unitId] || { face: 0 });
  st.aim = (Number(fx.amount) || 0) > 0 ? 1 : 0;
});
