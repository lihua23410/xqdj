const fxRoot = document.getElementById("fx");
let prevHP = new Map();
let dashGhostAt = new Map();

const FALLBACK_COLOR = "#3dd6c6";
const ENGINE_FX = new Set([
  "hurt",
  "heal",
  "swap",
  "wall-spawn",
  "wall-fade",
  "wall",
  "impact",
  "hit",
  "faction",
]);

function lookOf(kind) {
  const looks = (state && state.looks) || {};
  return looks[kind] || {};
}

function kindColor(kind) {
  const fac = factionById[kind];
  if (fac && fac.color) return fac.color;
  return lookOf(kind).color || FALLBACK_COLOR;
}

function paintKind(el, kind) {
  applyLookFX(el, kind);
  el.style.setProperty("--kind", kindColor(kind));
}

const loadedPackFiles = new Set();

function fxID(name) {
  if (typeof name !== "string" || !name) return "";
  const m = name.match(/^(?:\/fx\/)?([a-zA-Z0-9_-]+)(?:\.css)?$/);
  return m ? m[1] : "";
}

function packBaseFromScript() {
  const src = document.currentScript && document.currentScript.src;
  if (!src) return "";
  try {
    const path = new URL(src, location.origin).pathname;
    const m = path.match(/^(\/ball\/[^/]+)\//);
    return m ? m[1] : "";
  } catch {
    return "";
  }
}

window.shotFX = window.shotFX || {};

function registerShot(name, fn) {
  const base = packBaseFromScript();
  if (!base || typeof fn !== "function") return;
  window.shotFX[base] = window.shotFX[base] || {};
  window.shotFX[base][name] = fn;
}

const factions = [];
const factionById = {};

function registerFactions(list) {
  const base = packBaseFromScript();
  if (!base || !Array.isArray(list)) return;
  for (const item of list) {
    if (!item || !item.id) continue;
    const icon = item.icon || (item.file ? `${base}/${item.file}` : "");
    const entry = { id: item.id, icon, color: item.color || "" };
    const prev = factionById[item.id];
    if (prev) {
      factions[factions.indexOf(prev)] = entry;
    } else {
      factions.push(entry);
    }
    factionById[item.id] = entry;
  }
}

window.arena = {
  lookOf,
  kindColor,
  screenPos,
  spawnFx,
  burst,
  spawnGhostFrom,
  registerShot,
  registerFactions,
};

function ensurePacks() {
  const packs = (state && state.packs) || {};
  for (const pack of Object.values(packs)) {
    const base = pack.base || "";
    for (const file of pack.files || []) {
      const url = `${base}/${file}`;
      if (loadedPackFiles.has(url)) continue;
      loadedPackFiles.add(url);
      if (file.endsWith(".css")) {
        const link = document.createElement("link");
        link.rel = "stylesheet";
        link.href = url;
        document.head.appendChild(link);
      } else if (file.endsWith(".js")) {
        const script = document.createElement("script");
        script.src = url;
        script.async = false;
        document.head.appendChild(script);
      }
    }
  }
}

function applyLookFX(el, kind) {
  const want = new Set();
  for (const name of lookOf(kind).fx || []) {
    const id = fxID(name);
    if (id) want.add(id);
  }
  for (const c of [...el.classList]) {
    if (!c.startsWith("look-")) continue;
    const id = c.slice("look-".length);
    if (want.has(id)) continue;
    el.classList.remove(c);
    const hook = window.lookFX && window.lookFX[id];
    if (hook && typeof hook.unmount === "function") hook.unmount(el);
  }
  for (const id of want) el.classList.add("look-" + id);
}

function runLookFX(el, u, ctx) {
  const hooks = window.lookFX;
  if (!hooks) return;
  for (const name of lookOf(u.kind).fx || []) {
    const id = fxID(name);
    const hook = id && hooks[id];
    if (hook && typeof hook.tick === "function") hook.tick(el, u, ctx);
  }
}

function runLookGuides(scale, cx, cy, seen) {
  const hooks = window.lookFX;
  if (!hooks) return;
  const ctx = {
    scale,
    cx,
    cy,
    units: state.units || [],
    ensureGuide,
    placeSeg,
    seenGuides: seen,
  };
  for (const u of state.units || []) {
    for (const name of lookOf(u.kind).fx || []) {
      const hook = hooks[fxID(name)];
      if (hook && typeof hook.guide === "function") hook.guide(u, ctx);
    }
  }
}

function fillFactionHud(root, u) {
  if (!root) return;
  root.innerHTML = "";
  const nowFac = u && u.faction && factionById[u.faction];
  if (!nowFac || !nowFac.icon) return;
  const now = document.createElement("img");
  now.className = "now";
  now.src = nowFac.icon;
  now.alt = u.faction;
  root.appendChild(now);
  if (!u.seen) return;
  const got = new Set(u.seen || []);
  for (const f of factions) {
    if (!f.icon) continue;
    const pip = document.createElement("img");
    pip.className = `pip${got.has(f.id) ? " on" : ""}`;
    pip.src = f.icon;
    pip.alt = f.id;
    root.appendChild(pip);
  }
}

function placeFactionBadge(u, sx, sy, r) {
  const badgeId = `fac-${u.id}`;
  const pipsId = `pips-${u.id}`;
  const nowFac = u.role === "fighter" && u.faction && factionById[u.faction];
  if (!nowFac || !nowFac.icon) {
    document.getElementById(badgeId)?.remove();
    document.getElementById(pipsId)?.remove();
    return;
  }
  let badge = document.getElementById(badgeId);
  if (!badge) {
    badge = document.createElement("img");
    badge.id = badgeId;
    badge.className = "faction-badge";
    unitsEl.appendChild(badge);
  }
  badge.src = nowFac.icon;
  badge.alt = u.faction;
  badge.style.left = `${sx}px`;
  badge.style.top = `${sy}px`;
  if (!u.seen) {
    document.getElementById(pipsId)?.remove();
    return;
  }
  let pips = document.getElementById(pipsId);
  if (!pips) {
    pips = document.createElement("div");
    pips.id = pipsId;
    pips.className = "faction-pips";
    unitsEl.appendChild(pips);
  }
  const got = new Set(u.seen || []);
  pips.innerHTML = "";
  for (const f of factions) {
    if (!f.icon) continue;
    const img = document.createElement("img");
    img.src = f.icon;
    img.alt = f.id;
    img.className = got.has(f.id) ? "on" : "";
    pips.appendChild(img);
  }
  pips.style.left = `${sx}px`;
  pips.style.top = `${sy - r - 16}px`;
}

function fillMarksHud(root, u) {
  if (!root) return;
  root.innerHTML = "";
  const marks = (u && u.marks) || [];
  for (const m of marks) {
    if (!m || m.stacks <= 0) continue;
    const wrap = document.createElement("span");
    wrap.className = "status-mark";
    wrap.title = m.kind || "";
    if (m.icon) {
      const img = document.createElement("img");
      img.src = m.icon;
      img.alt = m.kind || "";
      wrap.appendChild(img);
    }
    const n = document.createElement("b");
    n.textContent = String(m.stacks);
    wrap.appendChild(n);
    root.appendChild(wrap);
  }
}

function placeMarks(u, sx, sy, r) {
  const id = `marks-${u.id}`;
  const marks = u.marks || [];
  if (!marks.length) {
    document.getElementById(id)?.remove();
    return;
  }
  let root = document.getElementById(id);
  if (!root) {
    root = document.createElement("div");
    root.id = id;
    root.className = "status-marks";
    unitsEl.appendChild(root);
  }
  root.innerHTML = "";
  fillMarksHud(root, u);
  root.style.left = `${sx + r + 10}px`;
  root.style.top = `${sy}px`;
}

function screenPos(x, y, scale, cx, cy) {
  return [cx + x * scale, cy - y * scale];
}

function dmgPop(amount) {
  const amt = Math.max(0, Number(amount) || 0);
  const t = Math.min(1, amt / 48);
  return {
    "--dmg-size": `${(14 + t * 30).toFixed(1)}px`,
    "--dmg-dur": `${(0.5 + t * 1.3).toFixed(2)}s`,
    "--dmg-rise": `${(-56 - t * 72).toFixed(0)}px`,
  };
}

function spawnFx(cls, x, y, kind, extra = {}, root = fxRoot) {
  if (!root) return null;
  const el = document.createElement("div");
  el.className = `fx ${cls}`;
  el.style.left = `${x}px`;
  el.style.top = `${y}px`;
  el.style.color = kindColor(kind);
  for (const [k, v] of Object.entries(extra)) el.style.setProperty(k, v);
  root.appendChild(el);
  el.addEventListener("animationend", (ev) => {
    if (ev.target !== el) return;
    el.remove();
  });
  return el;
}

function spawnGhostFrom(el, kind, layer, extraClass = "") {
  const root = layer || fxRoot;
  if (!root || !el) return;
  const g = document.createElement("div");
  g.className = `ghost${extraClass ? ` ${extraClass}` : ""}`;
    if (el.classList.contains("semi")) g.classList.add("semi");
    if (el.classList.contains("glow")) g.classList.add("glow");
  g.style.setProperty("--kind", kindColor(kind));
  g.style.setProperty("--face-ang", el.style.getPropertyValue("--face-ang") || "0rad");
  g.style.width = el.style.width;
  g.style.height = el.style.height;
  g.style.left = el.style.left;
  g.style.top = el.style.top;
  g.style.transform = el.style.transform;
  root.appendChild(g);
  g.addEventListener("animationend", () => g.remove());
  return g;
}

function burst(x, y, kind, n = 10) {
  for (let i = 0; i < n; i++) {
    const a = (Math.PI * 2 * i) / n + Math.random() * 0.4;
    const d = 18 + Math.random() * 28;
    spawnFx("fx-spark", x, y, kind, {
      "--dx": `${Math.cos(a) * d}px`,
      "--dy": `${Math.sin(a) * d}px`,
    });
  }
}

function wallSeg(fx, scale, cx, cy) {
  const [x1, y1] = screenPos(fx.x, fx.y, scale, cx, cy);
  const [x2, y2] = screenPos(fx.vx, fx.vy, scale, cx, cy);
  const dx = x2 - x1;
  const dy = y2 - y1;
  return {
    x1, y1, x2, y2,
    mx: (x1 + x2) / 2,
    my: (y1 + y2) / 2,
    dx, dy,
    len: Math.hypot(dx, dy),
    ang: Math.atan2(dy, dx),
  };
}

function dustAlong(s, kind, n, spread) {
  for (let i = 0; i < n; i++) {
    const t = n === 1 ? 0.5 : i / (n - 1);
    const x = s.x1 + s.dx * t;
    const y = s.y1 + s.dy * t;
    const a = s.ang + Math.PI / 2 + (Math.random() - 0.5) * 0.8;
    const d = (Math.random() - 0.5) * spread;
    spawnFx("fx-wall-dust", x, y, kind, {
      "--dx": `${Math.cos(a) * d}px`,
      "--dy": `${Math.sin(a) * d}px`,
    });
  }
}

function playWallFx(fx, scale, cx, cy, fading) {
  const s = wallSeg(fx, scale, cx, cy);
  const kind = fx.kind;
  const slash = spawnFx(fading ? "fx-wall-slash out" : "fx-wall-slash", s.mx, s.my, kind, {
    "--len": `${Math.max(24, s.len)}px`,
    "--ang": `${s.ang}rad`,
  });
  if (slash) {
    slash.style.width = `${Math.max(24, s.len)}px`;
  }
  spawnFx("fx-flash", s.x1, s.y1, kind);
  spawnFx("fx-flash", s.x2, s.y2, kind);
  spawnFx("fx-ring", s.mx, s.my, kind);
  if (fading) {
    burst(s.mx, s.my, kind, 16);
    burst(s.x1, s.y1, kind, 8);
    burst(s.x2, s.y2, kind, 8);
    dustAlong(s, kind, 14, 46);
  } else {
    spawnFx("fx-shock", s.mx, s.my, kind);
    dustAlong(s, kind, 10, 22);
  }
}

function playEffects(effects, scale, cx, cy) {
  for (const fx of effects || []) {
    const [x, y] = screenPos(fx.x, fx.y, scale, cx, cy);
    const kind = fx.kind;
    const ctx = { x, y, kind, scale, cx, cy };
    if (fx.name === "heal") {
      const n = Math.round(fx.amount || 0);
      const el = spawnFx("fx-dmg heal", x, y - 12, kind, dmgPop(n));
      if (el) el.textContent = `+${n}`;
    } else if (fx.name === "faction") {
      spawnFx("fx-flash", x, y, kind);
      spawnFx("fx-ring", x, y, kind);
    } else if (fx.name === "hurt") {
      const n = Math.round(fx.amount || 0);
      const el = spawnFx("fx-dmg", x, y - 12, kind, dmgPop(n));
      if (el) el.textContent = `-${n}`;
    } else if (fx.name === "swap") {
      spawnFx("fx-swap", x, y, kind);
      spawnFx("fx-flash", x, y, kind);
      spawnFx("fx-ring", x, y, kind);
    } else if (fx.name === "wall-spawn") {
      playWallFx(fx, scale, cx, cy, false);
    } else if (fx.name === "wall-fade") {
      playWallFx(fx, scale, cx, cy, true);
    } else if (fx.name === "wall") {
      spawnFx("fx-flash", x, y, kind);
      spawnFx("fx-shock", x, y, kind);
    } else if (fx.name === "impact" || fx.name === "hit") {
      spawnFx("fx-flash", x, y, kind);
      spawnFx("fx-shock", x, y, kind);
      burst(x, y, kind, fx.name === "impact" ? 14 : 8);
    } else if (!ENGINE_FX.has(fx.name)) {
      const base = lookOf(kind).base;
      const hook = base && window.shotFX && window.shotFX[base] && window.shotFX[base][fx.name];
      if (typeof hook === "function") {
        hook(fx, ctx);
      } else if (fx.name) {
        spawnFx(`fx-${fx.name}`, x, y, kind);
      }
    }
  }
}
const lobby = document.getElementById("lobby");
const arena = document.getElementById("arena");
const hex = document.getElementById("hex");
const unitsEl = document.getElementById("units");
const overEl = document.getElementById("over");
const wallsEl = document.getElementById("walls");
const guidesEl = document.getElementById("guides");
const banner = document.getElementById("banner");
const kinds0 = document.getElementById("kinds-0");
const kinds1 = document.getElementById("kinds-1");

let state = { phase: "select", slots: ["", ""], kinds: [], units: [] };
let hexR = 280;

const wsProto = location.protocol === "https:" ? "wss" : "ws";
const ws = new WebSocket(`${wsProto}://${location.host}/ws`);

function send(obj) {
  if (ws.readyState === WebSocket.OPEN) {
    ws.send(JSON.stringify(obj));
  }
}

ws.onmessage = (ev) => {
  state = JSON.parse(ev.data);
  hexR = state.hexRadius || 280;
  render();
};

document.getElementById("btn-start").onclick = () => send({ type: "start" });
document.getElementById("btn-pause").onclick = () => {
  if (state.phase === "paused") send({ type: "start" });
  else send({ type: "pause" });
};
document.getElementById("btn-end").onclick = () => send({ type: "end" });

let lobbyKey = "";

function renderLobby() {
  const kinds = state.kinds || [];
  const key = `${(state.slots || []).join("\0")}\n${kinds.join("\0")}`;
  if (key === lobbyKey) {
    return;
  }
  lobbyKey = key;
  for (const [el, slot] of [
    [kinds0, 0],
    [kinds1, 1],
  ]) {
    el.innerHTML = "";
    for (const kind of kinds) {
      const b = document.createElement("button");
      b.textContent = kind;
      b.type = "button";
      b.dataset.slot = String(slot);
      paintKind(b, kind);
      if (state.slots && state.slots[slot] === kind) b.classList.add("active");
      b.onclick = () => send({ type: "select", slot, kind });
      el.appendChild(b);
    }
    const slotRoot = el.closest(".slot");
    if (slotRoot) {
      paintKind(slotRoot, state.slots[slot]);
    }
  }
}

function placeSeg(el, x1, y1, x2, y2, kind) {
  const dx = x2 - x1;
  const dy = y2 - y1;
  el.style.width = `${Math.hypot(dx, dy)}px`;
  el.style.left = `${(x1 + x2) / 2}px`;
  el.style.top = `${(y1 + y2) / 2}px`;
  el.style.setProperty("--ang", `${Math.atan2(dy, dx)}rad`);
  el.style.setProperty("--kind", kindColor(kind));
}

function ensureGuide(id, cls) {
  let el = document.getElementById(id);
  if (!el) {
    el = document.createElement("div");
    el.id = id;
    el.className = `guide ${cls}`;
    guidesEl.appendChild(el);
  }
  return el;
}

function renderGuides(fighters, scale, cx, cy) {
  if (!guidesEl) return;
  const seen = new Set();
  const wallers = (fighters || []).filter((u) => (lookOf(u.kind).wallGuide || 0) > 0);
  const drawn = new Set();
  for (const w of wallers) {
    const enemy = (fighters || []).find((u) => u.id !== w.id);
    if (!enemy) continue;
    const pair = w.id < enemy.id ? `${w.id}-${enemy.id}` : `${enemy.id}-${w.id}`;
    if (drawn.has(pair)) continue;
    drawn.add(pair);
    const [ax, ay] = screenPos(w.x, w.y, scale, cx, cy);
    const [bx, by] = screenPos(enemy.x, enemy.y, scale, cx, cy);
    const link = ensureGuide(`guide-link-${pair}`, "link");
    placeSeg(link, ax, ay, bx, by, w.kind);
    seen.add(link.id);

    const dx = enemy.x - w.x;
    const dy = enemy.y - w.y;
    const n = Math.hypot(dx, dy);
    if (n < 1e-6) continue;
    const px = -dy / n;
    const py = dx / n;
    const half = lookOf(w.kind).wallGuide / 2;
    const mx = (w.x + enemy.x) / 2;
    const my = (w.y + enemy.y) / 2;
    const [g1x, g1y] = screenPos(mx + px * half, my + py * half, scale, cx, cy);
    const [g2x, g2y] = screenPos(mx - px * half, my - py * half, scale, cx, cy);
    const hint = ensureGuide(`guide-wall-${pair}`, "hint");
    placeSeg(hint, g1x, g1y, g2x, g2y, w.kind);
    seen.add(hint.id);
  }
  runLookGuides(scale, cx, cy, seen);
  for (const child of [...guidesEl.children]) {
    if (!seen.has(child.id)) child.remove();
  }
}

function renderWalls(scale, cx, cy) {
  if (!wallsEl) return;
  const seen = new Set();
  for (const w of state.walls || []) {
    seen.add(String(w.id));
    let el = document.getElementById(`wall-${w.id}`);
    if (!el) {
      el = document.createElement("div");
      el.id = `wall-${w.id}`;
      el.className = "wall";
      wallsEl.appendChild(el);
    }
    const [x1, y1] = screenPos(w.x1, w.y1, scale, cx, cy);
    const [x2, y2] = screenPos(w.x2, w.y2, scale, cx, cy);
    const dx = x2 - x1;
    const dy = y2 - y1;
    const len = Math.hypot(dx, dy);
    const thick = 2 * (w.radius || 6) * scale;
    el.style.width = `${len}px`;
    el.style.height = `${thick}px`;
    el.style.left = `${(x1 + x2) / 2}px`;
    el.style.top = `${(y1 + y2) / 2}px`;
    el.style.setProperty("--ang", `${Math.atan2(dy, dx)}rad`);
    el.style.setProperty("--kind", kindColor(w.kind));
  }
  for (const child of [...wallsEl.children]) {
    const id = child.id.slice("wall-".length);
    if (!seen.has(id)) child.remove();
  }
}

function renderArena() {
  const w = hex.clientWidth;
  const h = hex.clientHeight;
  const scale = w / (2 * hexR);
  const cx = w / 2;
  const cy = h / 2;
  hex.classList.toggle("hitstop", (state.hitStop || 0) > 0);

  const fighters = (state.units || []).filter((u) => u.role === "fighter");
  for (const slot of [0, 1]) {
    const f = fighters.find((u) => u.slot === slot);
    const root = document.getElementById(`fighter-${slot}`);
    const name = root.querySelector(".name");
    const num = root.querySelector(".hp-num");
    const bar = root.querySelector("i");
    const hudFac = root.querySelector(".faction-hud");
    const hudMarks = root.querySelector(".status-hud");
    if (!f) {
      name.textContent = state.slots[slot] || "";
      num.textContent = "0 / 100";
      bar.style.width = "0%";
      paintKind(root, state.slots[slot]);
      fillFactionHud(hudFac, null);
      fillMarksHud(hudMarks, null);
      continue;
    }
    name.textContent = f.kind;
    num.textContent = `${Math.round(f.hp)} / ${Math.round(f.maxHp)}`;
    bar.style.width = `${Math.max(0, (f.hp / f.maxHp) * 100)}%`;
    paintKind(root, f.kind);
    fillFactionHud(hudFac, f);
    fillMarksHud(hudMarks, f);
  }

  const pauseBtn = document.getElementById("btn-pause");
  pauseBtn.textContent = state.phase === "paused" ? "继续" : "暂停";

  renderGuides(fighters, scale, cx, cy);
  renderWalls(scale, cx, cy);
  playEffects(state.effects, scale, cx, cy);

  const seen = new Set();
  const now = performance.now();
  for (const u of state.units || []) {
    seen.add(String(u.id));
    const look = lookOf(u.kind);
    let el = document.getElementById(`u-${u.id}`);
    const layer = look.overlay && overEl ? overEl : unitsEl;
    if (!el) {
      el = document.createElement("div");
      el.id = `u-${u.id}`;
      el.className = "ball";
      layer.appendChild(el);
    } else if (el.parentNode !== layer) {
      layer.appendChild(el);
    }
    el.classList.toggle("projectile", u.role === "projectile" && !look.ring && !(u.arcSpan > 0));
    el.classList.toggle("clone", u.role === "clone");
    el.classList.toggle("semi", !!u.semi);
    const isArc = (u.arcSpan || 0) > 1e-6;
    el.classList.toggle("arc", isArc);
    el.classList.toggle("glow", !!look.glow);
    el.classList.toggle("ring", !!look.ring);
    el.classList.toggle("fighter", u.role === "fighter");
    el.classList.toggle("helper", u.role === "helper");
    const oldCut = el.querySelector(".cut");
    if (oldCut) oldCut.remove();
    el.style.setProperty("--kind", kindColor(u.kind));
    applyLookFX(el, u.kind);
    const speed = Math.hypot(u.vx || 0, u.vy || 0);
    const dashing = look.ghost > 0 && speed > look.ghost;
    el.classList.toggle("dashing", dashing);
    const r = u.radius * scale;
    el.style.width = `${r * 2}px`;
    el.style.height = `${r * 2}px`;
    const [sx, sy] = screenPos(u.x, u.y, scale, cx, cy);
    el.style.left = `${sx}px`;
    el.style.top = `${sy}px`;
    const hasFace = Math.abs(u.faceX) > 1e-9 || Math.abs(u.faceY) > 1e-9;
    const faceX = hasFace ? u.faceX : (u.vx || 1);
    const faceY = hasFace ? u.faceY : (u.vy || 0);
    const faceAng = (u.semi || isArc) ? Math.atan2(-faceY, faceX) : 0;
    el.style.setProperty("--face-ang", `${faceAng}rad`);
    if (isArc) {
      const outer = u.radius || 1;
      const inner = u.arcInner || 0;
      el.style.setProperty("--arc-span", `${u.arcSpan}rad`);
      el.style.setProperty("--arc-from", `${Math.PI / 2 - u.arcSpan / 2}rad`);
      el.style.setProperty("--arc-thick", `${Math.max(1.5, r * (1 - inner / outer))}px`);
    }
    el.style.transform = (u.semi || isArc)
      ? `translate(-50%, -50%) rotate(${faceAng}rad)`
      : "translate(-50%, -50%)";
    let ghostGap = 0;
    let ghostLayer = fxRoot;
    let ghostCls = "";
    if (dashing) {
      ghostGap = 32;
    } else if (look.trail) {
      ghostGap = 16;
      ghostLayer = overEl || fxRoot;
      ghostCls = "trail";
    }
    if (ghostGap) {
      const last = dashGhostAt.get(u.id) || 0;
      if (now - last > ghostGap) {
        dashGhostAt.set(u.id, now);
        spawnGhostFrom(el, u.kind, ghostLayer, ghostCls);
      }
    }
    runLookFX(el, u, {
      now,
      scale,
      cx,
      cy,
      fxRoot: overEl || fxRoot,
      spawnGhost: spawnGhostFrom,
    });
    const shownHP =
      u.role === "twin"
        ? fighters.find((f) => f.slot === u.slot)?.hp ?? u.hp
        : u.hp;
    const prev = prevHP.get(u.id);
    if (prev != null && shownHP < prev - 0.5) {
      burst(sx, sy, u.kind, 8);
    }
    if (u.role === "fighter" || u.role === "twin") prevHP.set(u.id, shownHP);
    let tag = document.getElementById(`hp-${u.id}`);
    if (u.role === "fighter" || u.role === "twin") {
      if (!tag) {
        tag = document.createElement("span");
        tag.id = `hp-${u.id}`;
        tag.className = "hp-float";
        unitsEl.appendChild(tag);
      }
      tag.textContent = String(Math.round(shownHP));
      tag.style.left = `${sx}px`;
      tag.style.top = `${sy - r - 4}px`;
    } else if (tag) {
      tag.remove();
    }
    const nested = el.querySelector(".hp-float");
    if (nested) nested.remove();
    placeFactionBadge(u, sx, sy, r);
    placeMarks(u, sx, sy, r);
    let ring = document.getElementById(`ring-${u.id}`);
    if (look.visionRing && u.vision > 0) {
      if (!ring) {
        ring = document.createElement("div");
        ring.id = `ring-${u.id}`;
        ring.className = "seek-ring";
        unitsEl.appendChild(ring);
      }
      const vr = u.vision * scale;
      ring.style.width = `${vr * 2}px`;
      ring.style.height = `${vr * 2}px`;
      ring.style.left = `${cx + u.x * scale}px`;
      ring.style.top = `${cy - u.y * scale}px`;
      ring.style.setProperty("--kind", kindColor(u.kind));
      const locked = (state.units || []).some(
        (o) =>
          o.id !== u.id &&
          o.role === "fighter" &&
          Math.hypot(o.x - u.x, o.y - u.y) <= u.vision
      );
      ring.classList.toggle("alert", locked);
    } else if (ring && ring.classList.contains("seek-ring")) {
      ring.remove();
      ring = null;
    }
  }
  for (const root of [unitsEl, overEl]) {
    if (!root) continue;
    for (const child of [...root.children]) {
      if (child.classList.contains("ghost")) continue;
      if (child.classList.contains("seek-ring")) {
        const id = child.id.replace(/^(ring|field)-/, "");
        if (!seen.has(id)) child.remove();
        continue;
      }
      if (child.classList.contains("faction-badge")) {
        const id = child.id.slice("fac-".length);
        if (!seen.has(id)) child.remove();
        continue;
      }
      if (child.classList.contains("faction-pips")) {
        const id = child.id.slice("pips-".length);
        if (!seen.has(id)) child.remove();
        continue;
      }
      if (child.classList.contains("status-marks")) {
        const id = child.id.slice("marks-".length);
        if (!seen.has(id)) child.remove();
        continue;
      }
      if (child.classList.contains("hp-float")) {
        const id = child.id.slice("hp-".length);
        if (!seen.has(id)) child.remove();
        continue;
      }
      const id = child.id.slice(2);
      if (!seen.has(id)) child.remove();
    }
  }

  if (state.phase === "ended") {
    banner.classList.remove("hidden");
    banner.textContent = state.winner === "平局" ? "平局" : `胜者：${state.winner}`;
  } else {
    banner.classList.add("hidden");
  }
}

function render() {
  ensurePacks();
  const inLobby = state.phase === "select";
  lobby.classList.toggle("hidden", !inLobby);
  arena.classList.toggle("hidden", inLobby);
  if (inLobby) {
    prevHP.clear();
    if (fxRoot) fxRoot.innerHTML = "";
    if (wallsEl) wallsEl.innerHTML = "";
    if (guidesEl) guidesEl.innerHTML = "";
    if (overEl) overEl.innerHTML = "";
    renderLobby();
  } else {
    lobbyKey = "";
    renderArena();
  }
}
