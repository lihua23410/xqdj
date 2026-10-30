window.lookFX = window.lookFX || {};

// 覆盖式像素钢盔直接构成整个可见轮廓；短盔缨可外溢，暗红仅从目缝透出。
// 球皮不参与原有火焰、火环或战斗状态计算。
const KNIGHT_SHELL_SVG = `<svg viewBox="0 0 32 32" aria-hidden="true" shape-rendering="crispEdges">
  <g class="knight-crest">
    <path fill="#3e0812" d="M15 2V0h4v-1h6v-2h5v1h2v3h-2V0h-5v2h-6v2Z"/>
    <path fill="#8f152a" d="M19 0h6v-2h4v1h2v1h-6v2h-6Z"/>
    <path fill="#b32a3e" d="M25-2h4v1h-4Z"/>
  </g>
  <path fill="#202d3b" d="M11 0h10v1h4v2h3v3h2v4h1v4h1v9h-1v3h-2v3h-4v2h-5v1h-8v-1H7v-2H3v-3H1v-4H0V11h1V7h2V4h3V2h5Z"/>
  <path fill="#879bac" d="M11 1h10v1h4v2h2v3h2v4h1v11h-1v4h-2v2h-3v2h-4v1h-8v-1H8v-2H4v-3H2v-4H1V11h1V8h2V5h3V3h4Z"/>
  <path fill="#ced8df" d="M11 2h7v9H2V9h2V6h3V4h4Z"/>
  <path fill="#f3f6f8" d="M11 2h7v1h-7v2H8v2H5V6h2V4h4Z"/>
  <path fill="#61758b" d="M20 3h4v2h2v3h2v4H18V2h2Z"/>
  <path fill="#aebdca" d="M16 2h2v9h-2Z"/>
  <path fill="#e5edf2" d="M3 11h26v1H3Z"/>
  <path fill="#080e18" d="M2 12h28v3h-3v1H5v-1H2Z"/>
  <path fill="#600010" d="M3 13h26v1h-3v1H6v-1H3Z"/>
  <path fill="#8f152a" d="M7 13h18v1H7Z"/>
  <path fill="#b32a3e" d="M11 13h9v1h-9Z"/>
  <path fill="#98a9b7" d="M3 16h26v6h-1v4h-3v3h-5v1h-8v-1H8v-2H5v-3H3Z"/>
  <path fill="#cad4dd" d="M4 17h10v11H9v-2H6v-3H4Z"/>
  <path fill="#5c7186" d="M18 16h11v6h-1v4h-3v3h-5v1h-2Z"/>
  <path fill="#e5edf2" d="M15 16h1v13h-1Z"/>
  <path fill="#7e94a7" d="M16 16h2v13h-2Z"/>
  <path fill="#33475b" d="M23 18h2v1h-2Zm0 3h2v1h-2Zm0 3h2v1h-2Z"/>
  <path fill="#edf3f6" d="M4 10h1v1H4Zm23 0h1v1h-1ZM2 17h1v2H2Zm27 0h1v2h-1Z"/>
</svg>`;

function knightShell(el) {
  if (el.querySelector(":scope > .knight-shell")) return;
  const shell = document.createElement("span");
  shell.className = "knight-shell";
  shell.setAttribute("aria-hidden", "true");
  shell.innerHTML = KNIGHT_SHELL_SVG;
  el.appendChild(shell);
}

function knightLocking(u) {
  const marks = (u && u.marks) || [];
  return marks.some((m) => m && m.kind === "锁血" && m.stacks > 0);
}

function knightHp(el) {
  const id = el && el.id && el.id.startsWith("u-") ? el.id.slice(2) : "";
  return id ? document.getElementById(`hp-${id}`) : null;
}

function knightId(el) {
  return el && el.id && el.id.startsWith("u-") ? el.id.slice(2) : "";
}

const KNIGHT_GLYPHS = "▓▒░█#%&*";
const KNIGHT_GLYPH_MAX = 4;

function knightClearGlyphs(id) {
  if (!id) return;
  document.querySelectorAll(`.hp-garble-char[data-knight="${id}"]`).forEach((n) => n.remove());
}

function knightClearLock(el) {
  if (!el) return;
  el.classList.remove("locking");
  const fire = el.querySelector(":scope > .knight-doom");
  if (fire) {
    if (fire._knightStop) fire._knightStop();
    fire.remove();
  }
  const id = knightId(el);
  const hp = knightHp(el);
  if (hp) hp.classList.remove("hp-garble");
  knightClearGlyphs(id);
}

const KNIGHT_DOOM_W = 18;
const KNIGHT_DOOM_H = 22;
// 火焰传播改编自 https://github.com/filipedeschamps/doom-fire-algorithm
// 许可全文见同目录 doom-fire-LICENSE（Copyright (c) 2019 Filipe Deschamps）。
const KNIGHT_DOOM_COLORS = [
  "#090003",
  "#220008",
  "#3c000d",
  "#600010",
  "#780017",
  "#8f152a",
  "#9f1f34",
  "#b32a3e",
];

// 对齐钢盔 32px 网格：目缝宽 26px，火根落在 y=14，火苗只向上绘制。
const KNIGHT_LOCK_W = 26;
const KNIGHT_LOCK_H = 18;

function knightIgnite(el) {
  if (el.querySelector(":scope > .knight-doom")) return;
  const canvas = document.createElement("canvas");
  canvas.className = "knight-doom";
  canvas.width = KNIGHT_LOCK_W;
  canvas.height = KNIGHT_LOCK_H;
  const ctx = canvas.getContext("2d");
  const cells = new Uint8Array(KNIGHT_LOCK_W * KNIGHT_LOCK_H);
  const last = KNIGHT_LOCK_H - 1;
  for (let x = 0; x < KNIGHT_LOCK_W; x++) cells[last * KNIGHT_LOCK_W + x] = KNIGHT_DOOM_COLORS.length - 1;
  let alive = true;
  canvas._knightStop = () => { alive = false; };
  el.appendChild(canvas);
  const step = () => {
    if (!alive || !canvas.isConnected) return;
    knightDoomStep(cells, KNIGHT_LOCK_W, KNIGHT_LOCK_H);
    const img = ctx.createImageData(KNIGHT_LOCK_W, KNIGHT_LOCK_H);
    for (let i = 0; i < cells.length; i++) {
      const x = i % KNIGHT_LOCK_W;
      const y = (i / KNIGHT_LOCK_W) | 0;
      const hex = KNIGHT_DOOM_COLORS[cells[i]] || KNIGHT_DOOM_COLORS[0];
      const n = parseInt(hex.slice(1), 16);
      const nx = (x + 0.5) / KNIGHT_LOCK_W * 2 - 1;
      const ny = (y + 0.5) / KNIGHT_LOCK_H;
      const halfWidth = 0.55 + 0.45 * ny;
      const fade = Math.max(0, Math.min(1, (halfWidth - Math.abs(nx)) * 8)) * Math.min(1, ny / 0.22);
      const o = i * 4;
      img.data[o] = (n >> 16) & 255;
      img.data[o + 1] = (n >> 8) & 255;
      img.data[o + 2] = n & 255;
      img.data[o + 3] = cells[i] ? Math.round(255 * fade) : 0;
    }
    ctx.putImageData(img, 0, 0);
    requestAnimationFrame(step);
  };
  requestAnimationFrame(step);
}

function knightDoomStep(cells, w, h) {
  const last = h - 1;
  for (let y = 0; y < last; y++) {
    for (let x = 0; x < w; x++) {
      const src = (y + 1) * w + x;
      const decay = Math.random() < 0.4 ? 1 : 0;
      const dstX = Math.min(w - 1, Math.max(0, x + Math.floor(Math.random() * 3) - 1));
      const next = cells[src] - decay;
      cells[y * w + dstX] = next > 0 ? next : 0;
    }
  }
}

function knightDoomRGB(level) {
  const hex = KNIGHT_DOOM_COLORS[Math.max(0, Math.min(KNIGHT_DOOM_COLORS.length - 1, level | 0))];
  return parseInt(hex.slice(1), 16);
}

function knightDoomPut(img, i, level, alpha) {
  if (alpha <= 0 || level <= 0) return;
  const n = knightDoomRGB(level);
  const o = i * 4;
  img.data[o] = (n >> 16) & 255;
  img.data[o + 1] = (n >> 8) & 255;
  img.data[o + 2] = n & 255;
  img.data[o + 3] = Math.round(255 * Math.min(1, alpha));
}

function knightDoomDisc(n) {
  const c = document.createElement("canvas");
  c.width = n;
  c.height = n;
  const ctx = c.getContext("2d");
  const img = ctx.createImageData(n, n);
  const cx = (n - 1) / 2;
  for (let y = 0; y < n; y++) {
    for (let x = 0; x < n; x++) {
      const dx = (x - cx) / (cx || 1);
      const dy = (y - cx) / (cx || 1);
      const r2 = dx * dx + dy * dy;
      if (r2 > 1) continue;
      const fade = 1 - Math.max(0, (r2 - 0.28) / 0.72);
      const hot = Math.max(1, (KNIGHT_DOOM_COLORS.length - 1) - Math.floor(Math.sqrt(r2) * 5));
      knightDoomPut(img, y * n + x, hot, fade);
    }
  }
  ctx.putImageData(img, 0, 0);
  return c;
}

function knightDoomStrip(cols) {
  const rows = 3;
  const c = document.createElement("canvas");
  c.width = Math.max(4, cols);
  c.height = rows;
  const ctx = c.getContext("2d");
  const img = ctx.createImageData(c.width, rows);
  for (let x = 0; x < c.width; x++) {
    const edge = Math.min(x, c.width - 1 - x) / Math.max(1, c.width * 0.12);
    const along = Math.min(1, edge);
    for (let y = 0; y < rows; y++) {
      const mid = 1 - Math.abs(y - 1) * 0.45;
      const hot = y === 1 ? KNIGHT_DOOM_COLORS.length - 2 : KNIGHT_DOOM_COLORS.length - 4;
      knightDoomPut(img, y * c.width + x, hot, along * mid);
    }
  }
  ctx.putImageData(img, 0, 0);
  return c;
}

function knightDoomBurst(n) {
  const c = document.createElement("canvas");
  c.width = n;
  c.height = n;
  const ctx = c.getContext("2d");
  const cells = new Uint8Array(n * n);
  const last = n - 1;
  for (let x = 0; x < n; x++) cells[last * n + x] = KNIGHT_DOOM_COLORS.length - 1;
  for (let step = 0; step < n; step++) {
    for (let y = 0; y < last; y++) {
      for (let x = 0; x < n; x++) {
        const src = (y + 1) * n + x;
        const decay = Math.random() < 0.45 ? 1 : 0;
        const dstX = Math.min(n - 1, Math.max(0, x + Math.floor(Math.random() * 3) - 1));
        const next = cells[src] - decay;
        cells[y * n + dstX] = next > 0 ? next : 0;
      }
    }
  }
  const img = ctx.createImageData(n, n);
  for (let i = 0; i < cells.length; i++) {
    const x = i % n;
    const y = (i / n) | 0;
    const nx = (x + 0.5) / n * 2 - 1;
    const ny = (y + 0.5) / n;
    const r2 = (nx / 0.9) * (nx / 0.9) + ((ny - 0.55) / 0.5) * ((ny - 0.55) / 0.5);
    let fade = r2 >= 1 ? 0 : 1 - Math.max(0, (r2 - 0.3) / 0.7);
    fade *= Math.min(1, ny / 0.18);
    knightDoomPut(img, i, cells[i], fade);
  }
  ctx.putImageData(img, 0, 0);
  return c;
}

function knightPix(x, y, kind, w, h, canvas, extra) {
  const el = arena.spawnFx("fx-knight-pix", x, y, kind, extra || {});
  if (!el) return null;
  el.style.width = `${w}px`;
  el.style.height = `${h}px`;
  canvas.className = "knight-pix";
  el.appendChild(canvas);
  return el;
}

function knightGlyph(el, hp) {
  const now = performance.now();
  const last = hp._knightGarbleAt || 0;
  if (now - last < 90) return;
  const id = knightId(el);
  if (!id || !hp.parentNode) return;
  const live = hp.parentNode.querySelectorAll(`.hp-garble-char[data-knight="${id}"]`);
  if (live.length >= KNIGHT_GLYPH_MAX) return;
  hp._knightGarbleAt = now;
  const span = document.createElement("span");
  span.className = "hp-garble-char";
  span.dataset.knight = id;
  span.textContent = KNIGHT_GLYPHS[Math.floor(Math.random() * KNIGHT_GLYPHS.length)];
  span.style.color = KNIGHT_DOOM_COLORS[3 + Math.floor(Math.random() * 4)];
  const x = parseFloat(hp.style.left) || 0;
  const y = parseFloat(hp.style.top) || 0;
  span.style.left = `${x + (Math.random() * 16 - 8)}px`;
  span.style.top = `${y + (Math.random() * 10 - 6)}px`;
  hp.parentNode.appendChild(span);
  span.addEventListener("animationend", () => span.remove());
}

function knightArcCanvas(el, u) {
  let c = el.querySelector(":scope > .knight-arc-pix");
  if (!c) {
    c = document.createElement("canvas");
    c.className = "knight-arc-pix";
    c.width = 32;
    c.height = 32;
    el.appendChild(c);
  }
  const size = c.width;
  const ctx = c.getContext("2d");
  const span = u.arcSpan > 0 ? u.arcSpan : (Math.PI * 2) / 3;
  const from = Math.PI / 2 - span / 2;
  const outer = size / 2 - 0.5;
  const innerRatio = u.radius > 0 ? (u.arcInner || u.radius * 0.9) / u.radius : 0.9;
  const inner = outer * innerRatio;
  const img = ctx.createImageData(size, size);
  const cx = (size - 1) / 2;
  const tick = (performance.now() / 80) | 0;
  for (let y = 0; y < size; y++) {
    for (let x = 0; x < size; x++) {
      const dx = x - cx;
      const dy = y - cx;
      const rr = Math.hypot(dx, dy);
      if (rr < inner || rr > outer) continue;
      let css = Math.atan2(dy, dx) + Math.PI / 2;
      let rel = css - from;
      rel %= Math.PI * 2;
      if (rel < 0) rel += Math.PI * 2;
      if (rel > span) continue;
      const edge = Math.min(rel, span - rel) / (span * 0.16);
      const angFade = Math.min(1, edge);
      const t = (rr - inner) / Math.max(0.001, outer - inner);
      const radFade = Math.min(1, Math.min(t, 1 - t) / 0.4);
      const hot = 3 + ((x * 3 + y * 5 + tick) % 4);
      knightDoomPut(img, y * size + x, hot, angFade * radFade);
    }
  }
  ctx.putImageData(img, 0, 0);
}

window.lookFX["knight-arc"] = {
  unmount(el) {
    el.querySelector(":scope > .knight-arc-pix")?.remove();
  },
  tick(el, u) {
    knightArcCanvas(el, u || {});
  },
};

const KNIGHT_RING_N = 48;
const KNIGHT_RING_DUR = 6;

function knightClearRing(el) {
  if (!el) return;
  const ring = el.querySelector(":scope > .knight-ring");
  if (!ring) return;
  if (ring._knightStop) ring._knightStop();
  ring.remove();
}

function knightEnsureRing(el) {
  let canvas = el.querySelector(":scope > .knight-ring");
  if (canvas) return canvas;
  canvas = document.createElement("canvas");
  canvas.className = "knight-ring";
  canvas.width = KNIGHT_RING_N;
  canvas.height = KNIGHT_RING_N;
  const cells = new Uint8Array(KNIGHT_DOOM_W * KNIGHT_DOOM_H);
  const last = KNIGHT_DOOM_H - 1;
  for (let x = 0; x < KNIGHT_DOOM_W; x++) cells[last * KNIGHT_DOOM_W + x] = KNIGHT_DOOM_COLORS.length - 1;
  canvas._cells = cells;
  canvas._fill = 0;
  canvas._alive = true;
  canvas._knightStop = () => { canvas._alive = false; };
  el.appendChild(canvas);
  const step = () => {
    if (!canvas._alive || !canvas.isConnected) return;
    try {
      knightRingPaint(canvas, canvas._fill || 0);
    } catch (err) {
      // #region agent log
      fetch("http://127.0.0.1:7695/ingest/ebd3d2b1-55a0-4ce8-b18e-b4c5f3e8e896", { method: "POST", headers: { "Content-Type": "application/json", "X-Debug-Session-Id": "bd0da0" }, body: JSON.stringify({ sessionId: "bd0da0", hypothesisId: "D", location: "knight.js:ring-raf", message: "ring paint threw", data: { error: String(err && err.message || err), fill: canvas._fill || 0 }, timestamp: Date.now() }) }).catch(() => {});
      // #endregion
      throw err;
    }
    requestAnimationFrame(step);
  };
  requestAnimationFrame(step);
  return canvas;
}

function knightRingRel(ang, sweep) {
  if (sweep <= 0) return -1;
  const start = -Math.PI / 2;
  let rel = ang - start;
  rel %= Math.PI * 2;
  if (rel < 0) rel += Math.PI * 2;
  return rel <= sweep ? rel : -1;
}

function knightRingPaint(canvas, fill) {
  const w = KNIGHT_DOOM_W;
  const h = KNIGHT_DOOM_H;
  const cells = canvas._cells;
  knightDoomStep(cells, w, h);
  const n = canvas.width;
  const ctx = canvas.getContext("2d");
  const img = ctx.createImageData(n, n);
  const c = (n - 1) / 2;
  const half = c || 1;
  const R = half * 0.58;
  const base = 1;
  const flameH = 6;
  const sweep = Math.max(0, Math.min(1, fill)) * Math.PI * 2;
  const ballR = half / 2.4;
  let opaque = 0;
  let inside = 0;
  let minY = n;
  let maxY = 0;
  for (let y = 0; y < n; y++) {
    for (let x = 0; x < n; x++) {
      const dx = x - c;
      const dy = y - c;
      const dist = Math.hypot(dx, dy);
      let rel = -1;
      let rise = 0;
      let fade = 0;
      if (Math.abs(dist - R) <= base) {
        rel = knightRingRel(Math.atan2(dy, dx), sweep);
        if (rel >= 0) fade = 1 - Math.abs(dist - R) / base;
      }
      if (rel < 0 && Math.abs(dx) <= R) {
        const span = Math.sqrt(Math.max(0, R * R - dx * dx));
        const rings = span === 0 ? [c] : [c - span, c + span];
        let best = Infinity;
        let srcY = 0;
        for (const ry of rings) {
          const up = ry - y;
          if (up < 0 || up > flameH || up >= best) continue;
          best = up;
          srcY = ry;
        }
        if (best <= flameH) {
          rel = knightRingRel(Math.atan2(srcY - c, dx), sweep);
          if (rel >= 0) {
            rise = best;
            fade = 1 - rise / flameH;
          }
        }
      }
      if (rel < 0 || fade <= 0) continue;
      const t = rise / flameH;
      const fy = Math.min(h - 1, Math.max(0, Math.round((1 - t) * (h - 1))));
      const fx = Math.floor((rel / (Math.PI * 2)) * w * 2) % w;
      const level = cells[fy * w + fx];
      if (level > 0) {
        opaque++;
        if (y < minY) minY = y;
        if (y > maxY) maxY = y;
        if (dist < ballR) inside++;
      }
      knightDoomPut(img, y * n + x, level, fade);
    }
  }
  ctx.putImageData(img, 0, 0);
  // #region agent log
  const now = Date.now();
  window.__knightDbgAt = window.__knightDbgAt || {};
  if (now - (window.__knightDbgAt.D || 0) > 500) {
    window.__knightDbgAt.D = now;
    fetch("http://127.0.0.1:7695/ingest/ebd3d2b1-55a0-4ce8-b18e-b4c5f3e8e896", { method: "POST", headers: { "Content-Type": "application/json", "X-Debug-Session-Id": "bd0da0" }, body: JSON.stringify({ sessionId: "bd0da0", hypothesisId: "D", location: "knight.js:knightRingPaint", message: "ring pixels", data: { fill, opaque, inside, minY, maxY, n, ballR, R }, timestamp: now }) }).catch(() => {});
  }
  // #endregion
}

function knightRingTick(el, u, ctx) {
  const id = u && u.id;
  const st = id != null && window.knightRing ? window.knightRing[id] : null;
  // #region agent log
  const now = Date.now();
  window.__knightDbgAt = window.__knightDbgAt || {};
  if (u && u.kind === "小骑士" && now - (window.__knightDbgAt.B || 0) > 500) {
    window.__knightDbgAt.B = now;
    const time = (ctx && ctx.time) || 0;
    const fill = st ? Math.max(0, Math.min(1, (time - (st.start || 0)) / KNIGHT_RING_DUR)) : null;
    fetch("http://127.0.0.1:7695/ingest/ebd3d2b1-55a0-4ce8-b18e-b4c5f3e8e896", { method: "POST", headers: { "Content-Type": "application/json", "X-Debug-Session-Id": "bd0da0" }, body: JSON.stringify({ sessionId: "bd0da0", hypothesisId: "B", location: "knight.js:knightRingTick", message: "ring tick", data: { id, idType: typeof id, found: !!st, on: st ? st.on : null, start: st ? st.start : null, readyAt: st ? st.readyAt : null, time, fill, elId: el && el.id, ringNodes: el ? el.querySelectorAll(":scope > .knight-ring").length : 0 }, timestamp: now }) }).catch(() => {});
  }
  // #endregion
  if (!st || !st.on) {
    knightClearRing(el);
    return;
  }
  const time = (ctx && ctx.time) || 0;
  const fill = Math.max(0, Math.min(1, (time - (st.start || 0)) / KNIGHT_RING_DUR));
  const canvas = knightEnsureRing(el);
  canvas._fill = fill;
}

window.lookFX.knight = {
  unmount(el) {
    el?.querySelector(":scope > .knight-shell")?.remove();
    knightClearLock(el);
    knightClearRing(el);
    const id = knightId(el);
    if (id && window.knightRing) delete window.knightRing[id];
  },
  tick(el, u, ctx) {
    knightShell(el);
    knightRingTick(el, u, ctx);
    if (!knightLocking(u)) {
      knightClearLock(el);
      return;
    }
    el.classList.add("locking");
    knightIgnite(el);
    const hp = knightHp(el);
    if (!hp) return;
    hp.classList.add("hp-garble");
    knightGlyph(el, hp);
  },
};
