window.lookFX = window.lookFX || {};
window.prisonerChain = window.prisonerChain || {};
window.prisonerDrops = window.prisonerDrops || {};
window.prisonerShock = window.prisonerShock || {};

// 与 Go 的 chainLen 对齐：短于 350 是松弛，到 350 才绷直。
// 旧版前端把门槛写成 250，链子会在还差 100 的时候就“假绷直”。
const CHAIN_LEN = 350;
// 绷直时约每 15px 一节；松弛时链节沿垂弧均匀铺开，弧长保持链条长度。
const CHAIN_LINKS = 24;
const CHAIN_TAUT = 1.5;
// 松弛到极限时链子向下垂的最大屏幕像素。太深会垂出场边，这里封顶。
const CHAIN_SAG = 90;
const CHAIN_SAMPLES = 48;
// 与 CSS .guide.prisoner-chain 的 height 保持一致；避免每帧读 clientHeight 触发布局。
const CHAIN_GUIDE_H = 20;
// 每条链一张 canvas：560×240 足够覆盖 350 长链在任意弦向下的 90px 垂弧。
const CHAIN_CANVAS_W = 560;
const CHAIN_CANVAS_H = 240;
const LINK_SPRITE_W = 34;
const LINK_SPRITE_H = 20;

function prisonerPackBase(u) {
  const look = (window.arena && typeof window.arena.lookOf === "function" && window.arena.lookOf(u.kind)) || {};
  return look.base || "/ball/囚徒";
}

function ensureArt(el, cls, src) {
  let img = el.querySelector(`:scope > .${cls}`);
  if (!img) {
    img = document.createElement("img");
    img.className = cls;
    img.alt = "";
    img.src = src;
    el.appendChild(img);
  }
  return img;
}

// 整条链画在 guide 内的一张 canvas 上：guide 只负责平移/旋转，canvas 里按
// 局部坐标画 24 个预渲染链节 + 电弧。不再每帧改 24 个 DOM 节点的 left/top/transform。
//
// 在弦的局部坐标系里采样垂链：x 沿囚徒→钩爪，y 是屏幕向下。
// 控制点放在弦中点下方 2×sag，二次曲线中点正好下移 sag；绷直时 control=mid，退化成直线。
function chainCurve(px, py, hx, hy, reach, taut) {
  const mx = (px + hx) / 2;
  const my = (py + hy) / 2;
  const dx = hx - px;
  const dy = hy - py;
  const ang = Math.atan2(dy, dx);
  const slack = Math.max(0, Math.min(1, 1 - reach / CHAIN_LEN));
  const sag = taut || slack <= 0 ? 0 : CHAIN_SAG * slack;
  const cx = mx;
  const cy = my + sag * 2;
  const cos = Math.cos(ang);
  const sin = Math.sin(ang);
  const pts = [];
  for (let i = 0; i <= CHAIN_SAMPLES; i++) {
    const t = i / CHAIN_SAMPLES;
    const it = 1 - t;
    const x = it * it * px + 2 * it * t * cx + t * t * hx;
    const y = it * it * py + 2 * it * t * cy + t * t * hy;
    pts.push({
      x: (x - mx) * cos + (y - my) * sin,
      y: -(x - mx) * sin + (y - my) * cos,
    });
  }
  let total = 0;
  const cum = [0];
  for (let i = 1; i < pts.length; i++) {
    total += Math.hypot(pts[i].x - pts[i - 1].x, pts[i].y - pts[i - 1].y);
    cum.push(total);
  }
  for (let i = 0; i < pts.length; i++) {
    const a = pts[Math.max(0, i - 1)];
    const b = pts[Math.min(pts.length - 1, i + 1)];
    pts[i].ang = Math.atan2(b.y - a.y, b.x - a.x);
  }
  return { pts, cum, total };
}

// 按累计弧长取点：链节间距按“链子实际长度”均匀，而不是按两端直线均分。
function curvePoint(curve, t) {
  const target = Math.max(0, Math.min(1, t)) * curve.total;
  const { pts, cum } = curve;
  let lo = 0;
  let hi = cum.length - 1;
  while (hi - lo > 1) {
    const mid = (lo + hi) >> 1;
    if (cum[mid] <= target) lo = mid;
    else hi = mid;
  }
  const a = pts[lo];
  const b = pts[hi];
  const d = cum[hi] - cum[lo];
  const f = d > 1e-9 ? (target - cum[lo]) / d : 0;
  return {
    x: a.x + (b.x - a.x) * f,
    y: a.y + (b.y - a.y) * f,
    ang: d > 1e-9 ? Math.atan2(b.y - a.y, b.x - a.x) : a.ang,
  };
}

const chainSprites = { dpr: 0, normal: null, shock: null };

// 1x 屏也按 1.5 倍渲染，旋转后的链节才不会糊；最多 2 倍，避免填充率过高。
function chainDpr() {
  return Math.min(Math.max(window.devicePixelRatio || 1, 1.5), 2);
}

// 链节只在第一次用到时画一次，之后每帧只是 drawImage。普通/触电各一张精灵。
function linkSprite(shocked) {
  const dpr = chainDpr();
  if (chainSprites.dpr !== dpr) {
    chainSprites.dpr = dpr;
    chainSprites.normal = null;
    chainSprites.shock = null;
  }
  const key = shocked ? "shock" : "normal";
  if (chainSprites[key]) return chainSprites[key];
  const c = document.createElement("canvas");
  c.width = Math.round(LINK_SPRITE_W * dpr);
  c.height = Math.round(LINK_SPRITE_H * dpr);
  const x = c.getContext("2d");
  x.scale(dpr, dpr);
  x.translate(LINK_SPRITE_W / 2, LINK_SPRITE_H / 2);
  const ring = (w, h, lw, color) => {
    x.beginPath();
    x.ellipse(0, 0, w, h, 0, 0, Math.PI * 2);
    x.lineWidth = lw;
    x.strokeStyle = color;
    x.stroke();
  };
  x.save();
  x.translate(0.6, 1.1);
  ring(10, 5, 3.8, "rgba(0,0,0,0.3)");
  x.restore();
  ring(10, 5, 3.4, shocked ? "#0b2630" : "#2a1408");
  const g = x.createLinearGradient(-10, -6, 10, 6);
  if (shocked) {
    g.addColorStop(0, "#0b2630");
    g.addColorStop(0.18, "#1d5b6b");
    g.addColorStop(0.36, "#7fe6ff");
    g.addColorStop(0.5, "#fcf05a");
    g.addColorStop(0.66, "#32b1cc");
    g.addColorStop(0.84, "#eaffff");
    g.addColorStop(1, "#123846");
  } else {
    g.addColorStop(0, "#3a1e0c");
    g.addColorStop(0.18, "#8a5628");
    g.addColorStop(0.36, "#f7d29b");
    g.addColorStop(0.5, "#c99763");
    g.addColorStop(0.68, "#6a3c1c");
    g.addColorStop(0.84, "#e0a46a");
    g.addColorStop(1, "#4a2a14");
  }
  ring(10, 5, 2.6, g);
  x.beginPath();
  x.ellipse(0, 0, 9.2, 4.2, 0, Math.PI * 1.05, Math.PI * 1.65);
  x.lineWidth = 0.9;
  x.strokeStyle = shocked ? "rgba(220,255,255,0.7)" : "rgba(255,232,190,0.6)";
  x.stroke();
  chainSprites[key] = c;
  return c;
}

function chainCanvas(g) {
  let c = g._chainCanvas;
  if (!c || !c.isConnected) {
    c = g.querySelector(":scope > canvas.chain-paint");
    if (!c) {
      g.querySelector(":scope > .chain-links")?.remove();
      g.querySelector(":scope > .chain-current")?.remove();
      c = document.createElement("canvas");
      c.className = "chain-paint";
      c.setAttribute("aria-hidden", "true");
      c.style.position = "absolute";
      c.style.width = `${CHAIN_CANVAS_W}px`;
      c.style.height = `${CHAIN_CANVAS_H}px`;
      c.style.top = `${CHAIN_GUIDE_H / 2 - CHAIN_CANVAS_H / 2}px`;
      c.style.pointerEvents = "none";
      g.appendChild(c);
    }
    g._chainCanvas = c;
  }
  const dpr = chainDpr();
  if (c._dpr !== dpr) {
    c._dpr = dpr;
    c.width = Math.round(CHAIN_CANVAS_W * dpr);
    c.height = Math.round(CHAIN_CANVAS_H * dpr);
    c._ctx = c.getContext("2d");
    c._ctx.setTransform(dpr, 0, 0, dpr, 0, 0);
  }
  return c;
}

// 触电时沿垂弧爬的电流：形状每 65ms 重抖一次，虚线偏移每帧推进，读起来在走。
function paintChainArc(draw, canvas, curve, ox, oy, now) {
  if (!canvas._amp || now - (canvas._at || 0) > 65) {
    canvas._at = now;
    canvas._amp = Array.from({ length: 9 }, () => 0.35 + Math.random() * 0.65);
  }
  const src = curve.pts;
  const step = Math.max(1, Math.round((src.length - 1) / 12));
  const base = [];
  for (let i = 0; i < src.length; i += step) base.push(src[i]);
  if (base[base.length - 1] !== src[src.length - 1]) base.push(src[src.length - 1]);
  draw.beginPath();
  for (let i = 0; i < base.length; i++) {
    const p = base[i];
    const nx = -Math.sin(p.ang);
    const ny = Math.cos(p.ang);
    let off = 0;
    if (i > 0 && i < base.length - 1) {
      off = (i % 2 ? 1 : -1) * (2.2 + 1.7 * canvas._amp[(i - 1) % canvas._amp.length]);
    }
    const x = p.x + nx * off + ox;
    const y = p.y + ny * off + oy;
    if (i === 0) draw.moveTo(x, y);
    else draw.lineTo(x, y);
  }
  draw.lineJoin = "round";
  draw.lineCap = "round";
  draw.strokeStyle = "rgba(50,177,204,0.14)";
  draw.lineWidth = 14;
  draw.stroke();
  draw.strokeStyle = "rgba(79,216,255,0.38)";
  draw.lineWidth = 7;
  draw.stroke();
  draw.strokeStyle = "#f2ffff";
  draw.lineWidth = 2.4;
  draw.stroke();
  draw.setLineDash([8, 14]);
  draw.lineDashOffset = -((now / 16) % 22);
  draw.strokeStyle = "#fcf05a";
  draw.lineWidth = 2.6;
  draw.stroke();
  draw.setLineDash([]);
}

function paintChain(canvas, curve, shocked, now) {
  const draw = canvas._ctx;
  const ox = CHAIN_CANVAS_W / 2;
  const oy = CHAIN_CANVAS_H / 2;
  draw.clearRect(0, 0, CHAIN_CANVAS_W, CHAIN_CANVAS_H);
  const sprite = linkSprite(shocked);
  for (let i = 0; i < CHAIN_LINKS; i++) {
    const p = curvePoint(curve, i / (CHAIN_LINKS - 1));
    draw.save();
    draw.translate(p.x + ox, p.y + oy);
    draw.rotate(p.ang + (i % 2 ? Math.PI / 2 : 0));
    draw.drawImage(sprite, -LINK_SPRITE_W / 2, -LINK_SPRITE_H / 2, LINK_SPRITE_W, LINK_SPRITE_H);
    draw.restore();
  }
  if (shocked) paintChainArc(draw, canvas, curve, ox, oy, now);
}

window.lookFX.prisoner = {
  tick(el, u) {
    if (!el || !el.classList.contains("look-prisoner")) return;
    const now = performance.now();
    const until = (window.prisonerShock && window.prisonerShock[u.id]) || 0;
    const marks = (u && u.marks) || [];
    const shocked = now < until || marks.some((m) => m && m.kind === "触电" && m.stacks > 0);
    el.classList.toggle("is-shock", shocked);
  },
  guide(u, ctx) {
    if (!u || !ctx || !ctx.ensureGuide || !window.arena) return;
    const st = window.prisonerChain[u.id];
    if (st && Number.isFinite(st.hx) && Number.isFinite(st.hy)) {
      // 锚点以本帧的钩爪单位为准：effects 比 guides 晚一步处理，换钩爪那一帧
      // 若还用上一帧的 hx/hy，链子会先甩到旧锚点再弹回来。
      const hook = (ctx.units || []).find(
        (o) => o && o.kind === "囚徒钩爪" && o.ownerId === u.id && Number.isFinite(o.x),
      );
      const hx = hook ? hook.x : st.hx;
      const hy = hook ? hook.y : st.hy;
      const [x1, y1] = arena.screenPos(u.x, u.y, ctx.scale, ctx.cx, ctx.cy);
      const [x2, y2] = arena.screenPos(hx, hy, ctx.scale, ctx.cx, ctx.cy);
      const g = ctx.ensureGuide(`guide-chain-${u.id}`, "prisoner-chain");
      ctx.placeSeg(g, x1, y1, x2, y2, u.kind);

      const reach = Math.hypot(hx - u.x, hy - u.y);
      const taut = reach >= CHAIN_LEN - CHAIN_TAUT;
      const screenLen = Math.hypot(x2 - x1, y2 - y1);
      const now = performance.now();
      const curve = chainCurve(x1, y1, x2, y2, reach, taut);
      const until = (window.prisonerShock && window.prisonerShock[u.id]) || 0;
      const shocked = now < until;

      const canvas = chainCanvas(g);
      canvas.style.left = `${(screenLen / 2 - CHAIN_CANVAS_W / 2).toFixed(1)}px`;
      paintChain(canvas, curve, shocked, now);
      if (ctx.seenGuides) ctx.seenGuides.add(g.id);
    }

    const now = performance.now();
    for (const [key, d] of Object.entries(window.prisonerDrops)) {
      if (!d || now > d.until) {
        delete window.prisonerDrops[key];
        continue;
      }
      const [x, y] = arena.screenPos(d.x, d.y, ctx.scale, ctx.cx, ctx.cy);
      const g = ctx.ensureGuide(`guide-drop-${key}`, "prisoner-drop");
      g.style.left = `${x}px`;
      g.style.top = `${y}px`;
      g.style.setProperty("--kind", arena.kindColor(d.kind));
      g.dataset.kind = d.kind || "";
      if (ctx.seenGuides) ctx.seenGuides.add(g.id);
    }
  },
};

window.lookFX.cage = {
  unmount(el) {
    el?.querySelector(":scope > .cage-grid")?.remove();
  },
  tick(el) {
    if (!el || !el.classList.contains("look-cage")) return;
    if (el.querySelector(":scope > .cage-grid")) return;
    const grid = document.createElement("span");
    grid.className = "cage-grid";
    grid.innerHTML = `<svg viewBox="0 0 120 120" aria-hidden="true">
      <circle cx="60" cy="60" r="58" fill="none" stroke="#3a3a3a" stroke-width="2"/>
      <circle cx="60" cy="60" r="38" fill="none" stroke="#3a3a3a" stroke-width="1.4"/>
      <circle cx="60" cy="60" r="18" fill="none" stroke="#3a3a3a" stroke-width="1.2"/>
      <g stroke="#3a3a3a" stroke-width="1.4">
        <line x1="60" y1="2" x2="60" y2="118"/>
        <line x1="2" y1="60" x2="118" y2="60"/>
        <line x1="18" y1="18" x2="102" y2="102"/>
        <line x1="102" y1="18" x2="18" y2="102"/>
      </g>
      <g fill="none" stroke="#2a2a2a" stroke-width="1">
        <rect x="8" y="8" width="14" height="14"/>
        <rect x="98" y="8" width="14" height="14"/>
        <rect x="8" y="98" width="14" height="14"/>
        <rect x="98" y="98" width="14" height="14"/>
        <rect x="46" y="46" width="28" height="28"/>
      </g>
    </svg>`;
    el.appendChild(grid);
  },
};

window.lookFX.gallows = {
  unmount(el) {
    el?.querySelector(":scope > .gallows-art")?.remove();
  },
  tick(el, u) {
    if (!el || !el.classList.contains("look-gallows")) return;
    ensureArt(el, "gallows-art", `${prisonerPackBase(u)}/fx/gallows.png`);
  },
};

window.lookFX.chair = {
  unmount(el) {
    el?.querySelector(":scope > .chair-art")?.remove();
  },
  tick(el, u) {
    if (!el || !el.classList.contains("look-chair")) return;
    ensureArt(el, "chair-art", `${prisonerPackBase(u)}/fx/chair.svg`);
  },
};
