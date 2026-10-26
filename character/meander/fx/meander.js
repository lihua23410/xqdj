// 迷走前端：给迷径补两层“迷”的东西——球后的一小段迷影，和两条环交叉处的岔路菱形。
// 只画视觉，不改任何判定；全部放在场地图层里被场地裁切。
window.lookFX = window.lookFX || {};

const MEANDER_RING = "迷走环线";
const CROSS_GAP = 8; // 与引擎 maze.crossMargin 同值，世界单位；屏幕上要乘 scale

const dashSet = new WeakSet();
const trailAt = new WeakMap();
const smoothAt = new WeakMap();
let juncHost = null;
let juncNodes = [];

// 平滑跟随：服务端每拍给一个离散位置，直接写 left/top 会在两帧之间抖。
// 用一个很短的时间常数收敛到目标（τ=12ms，稳态滞后不到 1 个世界像素），
// 真实瞬移（换环/脱轨/相机跳）距离过大时直接跟，不拖影。
function smoothFollow(el, now) {
  const tx = parseFloat(el.style.left);
  const ty = parseFloat(el.style.top);
  if (!Number.isFinite(tx) || !Number.isFinite(ty)) return;
  const s = smoothAt.get(el);
  if (!s) {
    smoothAt.set(el, { x: tx, y: ty, t: now });
    return;
  }
  let dt = (now - s.t) / 1000;
  if (!(dt > 0) || dt > 0.25) dt = 1 / 60;
  s.t = now;
  if (Math.hypot(tx - s.x, ty - s.y) > 60) {
    s.x = tx;
    s.y = ty;
  } else {
    const k = 1 - Math.exp(-dt / 0.012);
    s.x += (tx - s.x) * k;
    s.y += (ty - s.y) * k;
  }
  el.style.left = `${s.x}px`;
  el.style.top = `${s.y}px`;
}

// 每条迷径的流动虚线用不同速度和方向，避免六条环像同一个齿轮。
function tuneDash(el, u) {
  if (dashSet.has(el)) return;
  dashSet.add(el);
  const seed = Number(u && u.id) || 0;
  const dur = 18 + (seed % 9) * 1.7;
  el.style.setProperty("--dash-dur", `${dur.toFixed(1)}s`);
  el.style.setProperty("--dash-dir", seed % 2 ? "reverse" : "normal");
}

function junctionHost() {
  const units = document.getElementById("units");
  if (!units) return null;
  let host = document.getElementById("meander-junctions");
  if (!host || host.parentNode !== units) {
    host = document.createElement("div");
    host.id = "meander-junctions";
    // ghost 类让 app.js 的每帧清理跳过它；具体样式由 meander.css 覆盖。
    host.className = "ghost meander-junctions";
    units.appendChild(host);
  }
  if (host !== juncHost) {
    juncHost = host;
    juncNodes = [];
  }
  return host;
}

// 取环当前“显示位置”（已过 smoothFollow），让交点贴着画面上的环走，而不是贴着服务端坐标。
function screenCircle(u) {
  const el = document.getElementById(`u-${u.id}`);
  if (!el) return null;
  const x = parseFloat(el.style.left);
  const y = parseFloat(el.style.top);
  const r = parseFloat(el.style.width) / 2;
  if (!Number.isFinite(x) || !Number.isFinite(y) || !Number.isFinite(r) || r <= 0) return null;
  return { x, y, r };
}

// 两个严格相交的圆的交点；口径同引擎 maze.circleCross（相切/太近不算）。
// 这里的坐标已经是屏幕像素，掉进 gap 的判定也用屏幕单位。
function ringCross(a, b, gap) {
  const dx = b.x - a.x;
  const dy = b.y - a.y;
  const d = Math.hypot(dx, dy);
  if (d < 1e-6) return null;
  if (d >= a.r + b.r - gap || d <= Math.abs(a.r - b.r) + gap) return null;
  const along = (a.r * a.r - b.r * b.r + d * d) / (2 * d);
  const h2 = a.r * a.r - along * along;
  if (h2 <= 1e-6) return null;
  const h = Math.sqrt(h2);
  const ux = dx / d;
  const uy = dy / d;
  const px = a.x + ux * along;
  const py = a.y + uy * along;
  return [
    [px - uy * h, py + ux * h],
    [px + uy * h, py - ux * h],
  ];
}

// 每帧重算、复用节点池：不再每 100ms 重建 DOM，也不再 10Hz 跳格。
function paintJunctions(ctx) {
  const host = junctionHost();
  if (!host) return;
  const gap = CROSS_GAP * (Number(ctx.scale) || 1);
  const groups = new Map();
  for (const u of ctx.units || []) {
    if (u.kind !== MEANDER_RING || !(u.radius > 0)) continue;
    const c = screenCircle(u);
    if (!c) continue;
    const key = String(u.ownerId != null ? u.ownerId : "0");
    if (!groups.has(key)) groups.set(key, []);
    groups.get(key).push(c);
  }

  const pts = [];
  for (const list of groups.values()) {
    if (list.length < 2) continue;
    for (let i = 0; i < list.length; i++) {
      for (let j = i + 1; j < list.length; j++) {
        const cross = ringCross(list[i], list[j], gap);
        if (!cross) continue;
        pts.push(cross[0], cross[1]);
      }
    }
  }

  while (juncNodes.length < pts.length) {
    const el = document.createElement("i");
    el.className = "meander-junction";
    host.appendChild(el);
    juncNodes.push(el);
  }
  for (let i = 0; i < juncNodes.length; i++) {
    const el = juncNodes[i];
    if (i < pts.length) {
      if (el.style.display === "none") el.style.display = "";
      el.style.transform = `translate3d(${pts[i][0]}px, ${pts[i][1]}px, 0) translate(-50%, -50%) rotate(45deg)`;
    } else if (el.style.display !== "none") {
      el.style.display = "none";
    }
  }
}

// 迷影：球每走一小段，在身后落一个缩小、变淡的自己。
function dropTrail(el, u, ctx, now) {
  const speed = Math.hypot(Number(u.vx) || 0, Number(u.vy) || 0);
  if (speed < 5) return;
  const prev = trailAt.get(el);
  if (prev && now - prev < 100) return;
  const x = parseFloat(el.style.left);
  const y = parseFloat(el.style.top);
  if (!Number.isFinite(x) || !Number.isFinite(y)) return;
  trailAt.set(el, now);
  const layer = document.getElementById("fx");
  if (!layer) return;
  const vx = Number(u.vx) || 0;
  const vy = Number(u.vy) || 0;
  const size = (parseFloat(el.style.width) || 36) * 0.72;
  const deg = Math.atan2(-vy, vx);
  const back = size * 0.7;
  const n = document.createElement("i");
  n.className = "meander-trail";
  n.style.left = `${x - Math.cos(deg) * back}px`;
  n.style.top = `${y - Math.sin(deg) * back}px`;
  n.style.width = `${size}px`;
  n.style.height = `${size}px`;
  layer.appendChild(n);
  setTimeout(() => n.remove(), 500);
}

window.lookFX.meander = {
  unmount(el) {
    trailAt.delete(el);
    smoothAt.delete(el);
  },
  tick(el, u, ctx) {
    const now = performance.now();
    if (!ctx || !(ctx.scale > 0)) return;
    smoothFollow(el, now);
    dropTrail(el, u, ctx, now);
    paintJunctions(ctx);
  },
};

window.lookFX["meander-ring"] = {
  tick(el, u) {
    smoothFollow(el, performance.now());
    tuneDash(el, u);
  },
};

window.lookFX["meander-lit"] = {
  unmount(el) {
    smoothAt.delete(el);
  },
  tick(el, u) {
    smoothFollow(el, performance.now());
    tuneDash(el, u);
  },
};

// 迷走倒下后环会消失，把留在场上的交点节点清掉。
setInterval(() => {
  const host = document.getElementById("meander-junctions");
  if (!host) return;
  const has =
    typeof state !== "undefined" &&
    Array.isArray(state.units) &&
    state.units.some((u) => u.kind === MEANDER_RING && u.radius > 0);
  if (!has && host.childElementCount) {
    host.replaceChildren();
    juncNodes = [];
    juncHost = host;
  }
}, 500);
