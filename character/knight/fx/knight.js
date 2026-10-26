window.lookFX = window.lookFX || {};

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
  "#070002",
  "#1a0006",
  "#2e000a",
  "#4a000c",
  "#5c0012",
  "#6e1020",
  "#7a1828",
  "#8a2030",
];

function knightIgnite(el) {
  if (el.querySelector(":scope > .knight-doom")) return;
  const canvas = document.createElement("canvas");
  canvas.className = "knight-doom";
  canvas.width = KNIGHT_DOOM_W;
  canvas.height = KNIGHT_DOOM_H;
  const ctx = canvas.getContext("2d");
  const cells = new Uint8Array(KNIGHT_DOOM_W * KNIGHT_DOOM_H);
  const last = KNIGHT_DOOM_H - 1;
  for (let x = 0; x < KNIGHT_DOOM_W; x++) cells[last * KNIGHT_DOOM_W + x] = KNIGHT_DOOM_COLORS.length - 1;
  let alive = true;
  canvas._knightStop = () => { alive = false; };
  el.appendChild(canvas);
  const step = () => {
    if (!alive || !canvas.isConnected) return;
    knightDoomStep(cells, KNIGHT_DOOM_W, KNIGHT_DOOM_H);
    const img = ctx.createImageData(KNIGHT_DOOM_W, KNIGHT_DOOM_H);
    for (let i = 0; i < cells.length; i++) {
      const x = i % KNIGHT_DOOM_W;
      const y = (i / KNIGHT_DOOM_W) | 0;
      const hex = KNIGHT_DOOM_COLORS[cells[i]] || KNIGHT_DOOM_COLORS[0];
      const n = parseInt(hex.slice(1), 16);
      const nx = (x + 0.5) / KNIGHT_DOOM_W * 2 - 1;
      const ny = (y + 0.5) / KNIGHT_DOOM_H;
      const ex = nx / 0.92;
      const ey = (ny - 0.58) / 0.5;
      const r2 = ex * ex + ey * ey;
      let fade = r2 >= 1 ? 0 : 1 - Math.max(0, (r2 - 0.35) / 0.65);
      fade *= Math.min(1, ny / 0.2);
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
    knightClearLock(el);
    knightClearRing(el);
    const id = knightId(el);
    if (id && window.knightRing) delete window.knightRing[id];
  },
  tick(el, u, ctx) {
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
