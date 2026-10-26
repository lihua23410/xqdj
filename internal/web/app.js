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
  el.style.setProperty("--kind", kindColor(kind));
}

const loadedPackFiles = new Set();

function fxID(name) {
  if (typeof name !== "string" || !name) return "";
  const m = name.match(/^(?:\/fx\/)?([a-zA-Z0-9_-]+)(?:\.css)?$/);
  return m ? m[1] : "";
}

function packBaseFromScript() {
  const el = document.currentScript;
  if (el && el.dataset && el.dataset.pack) return el.dataset.pack;
  let src = (el && el.src) || "";
  if (!src) {
    const stack = String((new Error()).stack || "");
    const found = stack.match(/https?:\/\/[^)\s"']*\/ball\/[^)\s"']+/);
    if (found) src = found[0];
  }
  if (!src) return "";
  try {
    const path = decodeURIComponent(new URL(src, location.origin).pathname);
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

function ingestFactions(list, base) {
  if (!Array.isArray(list)) return;
  for (const item of list) {
    if (!item || !item.id) continue;
    const icon = item.icon || (item.file && base ? `${base}/${item.file}` : "");
    if (!icon) continue;
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

function registerFactions(list) {
  ingestFactions(list, packBaseFromScript());
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
    ingestFactions(pack.factions, base);
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
        if (base) script.dataset.pack = base;
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
    time: state.time || 0,
    effects: state.effects || [],
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
    "--dmg-size": `${(17.5 + t * 37.5).toFixed(1)}px`,
    "--dmg-dur": `${(0.8 + t).toFixed(2)}s`,
    "--dmg-rise": `${(-56 - t * 72).toFixed(0)}px`,
  };
}

function spawnFx(cls, x, y, kind, extra = {}, root = fxRoot) {
  if (!root) return null;
  const el = document.createElement("div");
  el.className = `fx ${cls}`;
  el.style.left = `${x}px`;
  el.style.top = `${y}px`;
  el.style.color = cls.split(/\s+/).includes("fx-dmg") ? "#fff" : kindColor(kind);
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
    const ctx = { x, y, kind, scale, cx, cy, time: state.time || 0 };
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
        // #region agent log
        if (fx.name === "fire-ring") {
          const now = Date.now();
          window.__knightDbgAt = window.__knightDbgAt || {};
          if (now - (window.__knightDbgAt.A2 || 0) > 500) {
            window.__knightDbgAt.A2 = now;
            fetch("http://127.0.0.1:7695/ingest/ebd3d2b1-55a0-4ce8-b18e-b4c5f3e8e896", { method: "POST", headers: { "Content-Type": "application/json", "X-Debug-Session-Id": "bd0da0" }, body: JSON.stringify({ sessionId: "bd0da0", hypothesisId: "A", location: "app.js:playEffects", message: "fire-ring fell through", data: { kind, base: lookOf(kind).base || "", hasBase: !!base }, timestamp: now }) }).catch(() => {});
          }
        }
        // #endregion
        spawnFx(`fx-${fx.name}`, x, y, kind);
      }
    }
  }
}
const lobby = document.getElementById("lobby");
// 不要叫 arena：经典脚本顶层 const 会盖住 window.arena，包里的 arena.registerShot 会变成对 DOM 节点的调用。
const arenaEl = document.getElementById("arena");
const hex = document.getElementById("hex");
const unitsEl = document.getElementById("units");
const overEl = document.getElementById("over");
const wallsEl = document.getElementById("walls");
const guidesEl = document.getElementById("guides");
const banner = document.getElementById("banner");
const kinds0 = document.getElementById("kinds-0");
const kinds1 = document.getElementById("kinds-1");
const fieldsEl = document.getElementById("fields");
const teamPage = document.body.dataset.page === "team";

let state = { phase: "select", slots: [[], []], kinds: [], units: [] };
let hexR = 280;
let follow = null;
let followHeld = false;
const hiddenBody = new Set(["无下限术士", "分身者(百战)", "蛆"]);
let hudKey = "";

// #region agent log
window.addEventListener("error", (ev) => {
  fetch("http://127.0.0.1:7695/ingest/ebd3d2b1-55a0-4ce8-b18e-b4c5f3e8e896", { method: "POST", headers: { "Content-Type": "application/json", "X-Debug-Session-Id": "bd0da0" }, body: JSON.stringify({ sessionId: "bd0da0", hypothesisId: "F", location: "app.js:onerror", message: "window error", data: { message: ev.message, filename: ev.filename, lineno: ev.lineno, colno: ev.colno, stack: ev.error && ev.error.stack }, timestamp: Date.now() }) }).catch(() => {});
});
// #endregion
const wsProto = location.protocol === "https:" ? "wss" : "ws";
const ws = new WebSocket(`${wsProto}://${location.host}/ws`);
ws.onopen = () => send({ type: "mode", team: teamPage });

function send(obj) {
  if (ws.readyState === WebSocket.OPEN) {
    ws.send(JSON.stringify(obj));
  }
}

function sideList(slot) {
  const s = state.slots && state.slots[slot];
  if (Array.isArray(s)) return s;
  if (typeof s === "string" && s) return [s];
  return [];
}

function lineup(slot) {
  const r = state.roster && state.roster[slot];
  if (r && r.length) return r;
  return sideList(slot);
}

let pendingFrame = null;
let frameScheduled = false;
let pendingCopy = null;
ws.onmessage = (ev) => {
  let msg;
  try { msg = JSON.parse(ev.data); } catch { return; }
  if (msg && msg.type === "log") {
    takeLog(msg);
    return;
  }
  pendingFrame = msg;
  if (frameScheduled) return;
  frameScheduled = true;
  requestAnimationFrame(() => {
    frameScheduled = false;
    const raw = pendingFrame;
    pendingFrame = null;
    if (raw == null) return;
    state = raw;
    hexR = state.hexRadius || 280;
    render();
  });
};

function rosterStatus(slot) {
  const edge = slot === 0 ? "左" : "右";
  return lineup(slot).map((kind, index) => {
    const f = (state.units || []).find(
      (u) => u.role === "fighter" && u.slot === slot && u.seat === index,
    );
    const name = `${edge}${index + 1} ${kind}`;
    if (!f || !(f.hp > 0)) return `${name} 倒下`;
    return `${name} ${Math.round(f.hp)}`;
  }).join("\n");
}

function copyLog(which) {
  pendingCopy = which;
  send({ type: "log" });
}
function takeLog(msg) {
  const which = pendingCopy;
  pendingCopy = null;
  if (!which) return;
  const text = which === "trace" ? (msg.trace || "") : (msg.recap || "");
  const box = document.getElementById("log-box");
  const done = () => {
    if (box) box.classList.add("hidden");
  };
  if (navigator.clipboard && navigator.clipboard.writeText) {
    navigator.clipboard.writeText(text).then(done).catch(() => showLogBox(text));
    return;
  }
  showLogBox(text);
}
function showLogBox(text) {
  const box = document.getElementById("log-box");
  if (!box) return;
  box.textContent = text;
  box.classList.remove("hidden");
  const range = document.createRange();
  range.selectNodeContents(box);
  const sel = window.getSelection();
  if (!sel) return;
  sel.removeAllRanges();
  sel.addRange(range);
}
function renderLogChrome() {
  const ready = !!state.logReady;
  const showBanner = ready && state.phase === "ended";
  const showLobby = ready && state.phase === "select";
  for (const el of document.querySelectorAll("[data-log='banner']")) {
    el.classList.toggle("hidden", !showBanner);
  }
  for (const el of document.querySelectorAll("[data-log='lobby']")) {
    el.classList.toggle("hidden", !showLobby);
  }
  if (!ready) {
    const box = document.getElementById("log-box");
    if (box) box.classList.add("hidden");
  }
}
for (const btn of document.querySelectorAll("[data-copy]")) {
  btn.addEventListener("click", () => copyLog(btn.dataset.copy));
}

document.getElementById("btn-start").onclick = () => {
  send({ type: "start" });
};
document.getElementById("btn-pause").onclick = () => {
  if (state.phase === "paused") send({ type: "start" });
  else send({ type: "pause" });
};
document.getElementById("btn-end").onclick = () => send({ type: "end" });
for (const id of ["side-0", "side-1"]) {
  const el = document.getElementById(id);
  if (el) el.addEventListener("click", onFighterBar);
}
{
  const wrap = document.querySelector(".stage-wrap");
  if (wrap) {
    wrap.addEventListener("pointerdown", onMapPointerDown);
    wrap.addEventListener("pointermove", onMapPointerMove);
    wrap.addEventListener("pointerup", onMapPointerUp);
    wrap.addEventListener("pointercancel", onMapPointerUp);
  }
}

let lobbyKey = "";

function paintFields(fields) {
  if (!fieldsEl) return;
  fieldsEl.innerHTML = "";
  for (const name of fields) {
    const b = document.createElement("button");
    b.textContent = name;
    b.type = "button";
    if ((state.field || "") === name) b.classList.add("active");
    b.onclick = () => send({ type: "field", field: name });
    fieldsEl.appendChild(b);
  }
}

function renderLobby() {
  const kinds = state.kinds || [];
  const fields = state.fields || [];
  const key = `${JSON.stringify(state.slots || [])}\n${kinds.join("\0")}\n${state.field || ""}\n${fields.join("\0")}`;
  if (key === lobbyKey) {
    return;
  }
  lobbyKey = key;
  paintFields(fields);
  paintKindGrids(kinds);
  if (teamPage) paintRosters();
  updateStartEnabled();
}

function paintKindGrids(kinds) {
  for (const [el, slot] of [
    [kinds0, 0],
    [kinds1, 1],
  ]) {
    if (!el) continue;
    el.innerHTML = "";
    const names = sideList(slot);
    const current = names[0] || "";
    const full = teamPage && names.length >= 3;
    for (const kind of kinds) {
      const b = document.createElement("button");
      b.textContent = kind;
      b.type = "button";
      b.dataset.slot = String(slot);
      paintKind(b, kind);
      if (!teamPage && current === kind) b.classList.add("active");
      if (full) b.disabled = true;
      b.onclick = () => {
        if (teamPage && sideList(slot).length >= 3) return;
        send({ type: "select", slot, kind });
      };
      el.appendChild(b);
    }
    const slotRoot = el.closest(".slot");
    if (slotRoot) {
      const accent = teamPage ? names[names.length - 1] || "" : current;
      paintKind(slotRoot, accent);
    }
  }
}

function paintRosters() {
  for (const slot of [0, 1]) {
    const host = document.getElementById(`roster-list-${slot}`);
    if (!host) continue;
    host.innerHTML = "";
    const names = sideList(slot);
    const head = host.previousElementSibling;
    if (head && head.tagName === "H2") {
      head.textContent = slot === 0 ? `左边 ${names.length}/3` : `右边 ${names.length}/3`;
    }
    if (!names.length) {
      const empty = document.createElement("p");
      empty.className = "roster-empty";
      empty.textContent = "点击角色加入";
      host.appendChild(empty);
      continue;
    }
    names.forEach((kind, index) => {
      const b = document.createElement("button");
      b.type = "button";
      b.textContent = kind;
      paintKind(b, kind);
      b.onclick = () => send({ type: "select", slot, index, kind: "" });
      host.appendChild(b);
    });
  }
}

function updateStartEnabled() {
  const btn = document.getElementById("btn-start");
  if (!btn) return;
  const ready = !teamPage || (sideList(0).length > 0 && sideList(1).length > 0);
  btn.disabled = !ready;
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

function distToSeg(px, py, x1, y1, x2, y2) {
  const dx = x2 - x1;
  const dy = y2 - y1;
  const l2 = dx * dx + dy * dy;
  if (l2 < 1e-12) return Math.hypot(px - x1, py - y1);
  let t = ((px - x1) * dx + (py - y1) * dy) / l2;
  if (t < 0) t = 0;
  else if (t > 1) t = 1;
  return Math.hypot(px - (x1 + dx * t), py - (y1 + dy * t));
}

function segSegDist(p1x, p1y, q1x, q1y, p2x, p2y, q2x, q2y) {
  const ux = q1x - p1x;
  const uy = q1y - p1y;
  const vx = q2x - p2x;
  const vy = q2y - p2y;
  const wx = p1x - p2x;
  const wy = p1y - p2y;
  const a = ux * ux + uy * uy;
  const b = ux * vx + uy * vy;
  const c = vx * vx + vy * vy;
  const d = ux * wx + uy * wy;
  const e = vx * wx + vy * wy;
  const D = a * c - b * b;
  let sN = D;
  let sD = D;
  let tN = D;
  let tD = D;
  if (D < 1e-12) {
    sN = 0;
    sD = 1;
    tN = e;
    tD = c;
  } else {
    sN = b * e - c * d;
    tN = a * e - b * d;
    if (sN < 0) {
      sN = 0;
      tN = e;
      tD = c;
    } else if (sN > sD) {
      sN = sD;
      tN = e + b;
      tD = c;
    }
  }
  if (tN < 0) {
    tN = 0;
    if (-d < 0) sN = 0;
    else if (-d > a) sN = sD;
    else {
      sN = -d;
      sD = a;
    }
  } else if (tN > tD) {
    tN = tD;
    if (-d + b < 0) sN = 0;
    else if (-d + b > a) sN = sD;
    else {
      sN = -d + b;
      sD = a;
    }
  }
  const sc = Math.abs(sN) > 1e-12 ? sN / sD : 0;
  const tc = Math.abs(tN) > 1e-12 ? tN / tD : 0;
  return Math.hypot(p1x + ux * sc - (p2x + vx * tc), p1y + uy * sc - (p2y + vy * tc));
}

function sightBlocked(self, o, units, walls) {
  for (const w of walls || []) {
    if (!w.visionBlock) continue;
    if (segSegDist(self.x, self.y, o.x, o.y, w.x1, w.y1, w.x2, w.y2) <= (w.radius || 0)) {
      return true;
    }
  }
  for (const b of units || []) {
    if (!b.visionBlock || b.id === self.id || b.id === o.id) continue;
    if (distToSeg(b.x, b.y, self.x, self.y, o.x, o.y) <= (b.radius || 0)) {
      return true;
    }
  }
  return false;
}

// 和 unit.Seek 同一口径：异边、瞄准优先度 > 0，数字越小越先，相同则最近；挡视线跳过。
function seekTarget(self, units, walls) {
  let best = null;
  let bestPri = 255;
  let bestD2 = Infinity;
  for (const o of units || []) {
    if (!o || o.id === self.id) continue;
    const pri = o.aimPriority || 0;
    if (o.slot === self.slot || pri <= 0) continue;
    if (sightBlocked(self, o, units, walls)) continue;
    const dx = o.x - self.x;
    const dy = o.y - self.y;
    const d2 = dx * dx + dy * dy;
    if (best && (pri > bestPri || (pri === bestPri && d2 >= bestD2))) continue;
    best = o;
    bestPri = pri;
    bestD2 = d2;
  }
  return best;
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
  const units = state.units || [];
  const walls = state.walls || [];
  const wallers = (fighters || []).filter((u) => (lookOf(u.kind).wallGuide || 0) > 0);
  for (const w of wallers) {
    const enemy = seekTarget(w, units, walls);
    if (!enemy) continue;
    const [ax, ay] = screenPos(w.x, w.y, scale, cx, cy);
    const [bx, by] = screenPos(enemy.x, enemy.y, scale, cx, cy);
    const link = ensureGuide(`guide-link-${w.id}`, "link");
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
    const hint = ensureGuide(`guide-wall-${w.id}`, "hint");
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
    el.classList.toggle("hard", !!w.hard);
    el.classList.toggle("square", !!w.square);
    el.classList.toggle("field", !!w.field || !!w.hard);
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
    if (w.field || w.hard) {
      el.style.setProperty("--kind", "#8b9cb3");
    } else {
      el.style.setProperty("--kind", kindColor(w.kind));
    }
  }
  for (const child of [...wallsEl.children]) {
    const id = child.id.slice("wall-".length);
    if (!seen.has(id)) child.remove();
  }
}

function fighterCard(slot, index) {
  const root = document.createElement("div");
  root.className = "fighter";
  root.id = `fighter-${slot}-${index}`;
  root.innerHTML =
    '<div class="name-row"><span class="name"></span><span class="faction-hud"></span><span class="status-hud"></span></div>' +
    '<span class="hp-num"></span><div class="hp"><i></i></div>';
  return root;
}

function renderHud(fighters) {
  const names = [lineup(0), lineup(1)];
  const key = JSON.stringify(names);
  if (key !== hudKey) {
    hudKey = key;
    for (const slot of [0, 1]) {
      const host = document.getElementById(`side-${slot}`);
      if (!host) continue;
      host.innerHTML = "";
      names[slot].forEach((kind, index) => {
        const card = fighterCard(slot, index);
        host.appendChild(card);
        paintKind(card, kind);
      });
    }
  }
  for (const slot of [0, 1]) {
    names[slot].forEach((kind, index) => {
      const root = document.getElementById(`fighter-${slot}-${index}`);
      if (!root) return;
      const f = fighters.find((u) => u.slot === slot && u.seat === index);
      const name = root.querySelector(".name");
      const num = root.querySelector(".hp-num");
      const bar = root.querySelector("i");
      const hudFac = root.querySelector(".faction-hud");
      const hudMarks = root.querySelector(".status-hud");
      if (!f) {
        name.textContent = kind;
        num.textContent = "0 / 100";
        bar.style.width = "0%";
        paintKind(root, kind);
        fillFactionHud(hudFac, null);
        fillMarksHud(hudMarks, null);
        return;
      }
      name.textContent = f.kind;
      const max = f.maxHp > 0 ? f.maxHp : 100;
      num.textContent = `${Math.round(f.hp)} / ${Math.round(max)}`;
      bar.style.width = `${Math.max(0, (f.hp / max) * 100)}%`;
      paintKind(root, f.kind);
      fillFactionHud(hudFac, f);
      fillMarksHud(hudMarks, f);
    });
  }
}

function livingFighter(slot, seat) {
  return (state.units || []).find(
    (u) => u.role === "fighter" && u.slot === slot && u.seat === seat && u.hp > 0,
  );
}

function firstLiving(slot) {
  const seats = (state.units || [])
    .filter((u) => u.role === "fighter" && u.slot === slot && u.hp > 0)
    .map((u) => u.seat)
    .sort((a, b) => a - b);
  return seats.length ? { slot, seat: seats[0] } : null;
}

function nextLiving(slot, seat) {
  const same = (state.units || [])
    .filter((u) => u.role === "fighter" && u.slot === slot && u.hp > 0 && u.seat !== seat)
    .map((u) => u.seat)
    .sort((a, b) => a - b);
  const after = same.find((s) => s > seat);
  if (after != null) return { slot, seat: after };
  if (same.length) return { slot, seat: same[0] };
  return firstLiving(slot === 0 ? 1 : 0);
}

function cameraPoint(f) {
  if (!f) return null;
  if (!hiddenBody.has(f.kind)) return { x: f.x, y: f.y };
  const mins = (state.units || []).filter(
    (u) => u.role === "minion" && u.ownerId === f.id && u.hp > 0,
  );
  if (!mins.length) return { x: f.x, y: f.y };
  let best = -Infinity;
  for (const m of mins) if (m.hp > best) best = m.hp;
  const tied = mins.filter((m) => m.hp >= best - 1e-6);
  let x = 0;
  let y = 0;
  for (const m of tied) {
    x += m.x;
    y += m.y;
  }
  return { x: x / tied.length, y: y / tied.length };
}

function syncFollow() {
  const phase = state.phase;
  if (phase === "select") {
    follow = null;
    followHeld = false;
    return null;
  }
  if (follow) {
    const cur = livingFighter(follow.slot, follow.seat);
    if (!cur) {
      follow = nextLiving(follow.slot, follow.seat);
    }
  }
  if (!follow) return null;
  return cameraPoint(livingFighter(follow.slot, follow.seat));
}

let mapDrag = null;

function isLargeField() {
  return (hexR || 0) > 320;
}

function mapWrap() {
  return hex.parentElement && hex.parentElement.parentElement;
}

function layoutLargeMap() {
  const large = isLargeField();
  const size = Math.max(2 * hexR, 1);
  if (!large) {
    hex.style.width = "";
    hex.style.height = "";
    hex.style.transform = "";
    if (overEl) {
      overEl.style.width = "";
      overEl.style.height = "";
      overEl.style.transform = "";
    }
    return false;
  }
  // 奇美拉凹外形：JS 写成 2×最远点的正方形。凸的大场皮由 CSS 定宽高。
  if (state.field === "奇美拉") {
    hex.style.width = `${size}px`;
    hex.style.height = `${size}px`;
    if (overEl) {
      overEl.style.width = `${size}px`;
      overEl.style.height = `${size}px`;
    }
  } else {
    hex.style.width = "";
    hex.style.height = "";
    if (overEl) {
      overEl.style.width = "";
      overEl.style.height = "";
    }
  }
  hex.style.transform = "";
  if (overEl) overEl.style.transform = "";
  const stage = hex.parentElement;
  if (stage) stage.style.overflow = "visible";
  return true;
}

function panLargeMapTo(worldX, worldY) {
  if (mapDrag) return;
  const rect = hex.getBoundingClientRect();
  const docX = rect.left + window.scrollX + rect.width / 2 + worldX;
  const docY = rect.top + window.scrollY + rect.height / 2 - worldY;
  window.scrollTo(docX - window.innerWidth / 2, docY - window.innerHeight / 2);
}

function onMapPointerDown(e) {
  if (!isLargeField() || e.button !== 0) return;
  const wrap = mapWrap();
  if (!wrap) return;
  e.preventDefault();
  mapDrag = {
    id: e.pointerId,
    x: e.clientX,
    y: e.clientY,
    left: window.scrollX,
    top: window.scrollY,
  };
  follow = null;
  followHeld = true;
  wrap.classList.add("dragging");
  wrap.setPointerCapture(e.pointerId);
}

function onMapPointerMove(e) {
  if (!mapDrag || e.pointerId !== mapDrag.id) return;
  window.scrollTo(
    mapDrag.left - (e.clientX - mapDrag.x),
    mapDrag.top - (e.clientY - mapDrag.y),
  );
}

function onMapPointerUp(e) {
  if (!mapDrag || e.pointerId !== mapDrag.id) return;
  mapDrag = null;
  const wrap = mapWrap();
  if (wrap) wrap.classList.remove("dragging");
}

function applyCamera(cam) {
  if (layoutLargeMap()) {
    if (cam && !mapDrag) panLargeMapTo(cam.x, cam.y);
    else if (!followHeld && !mapDrag) panLargeMapTo(0, 0);
    return 1;
  }
  const w = hex.clientWidth || 1;
  const h = hex.clientHeight || 1;
  const scale = w / (2 * hexR);
  let tx = 0;
  let ty = 0;
  if (cam) {
    tx = -cam.x * scale;
    ty = cam.y * scale;
  }
  const tf = cam ? `translate(${tx}px, ${ty}px)` : "";
  hex.style.transformOrigin = "50% 50%";
  hex.style.transform = tf;
  if (overEl) {
    overEl.style.transformOrigin = "50% 50%";
    overEl.style.transform = tf;
  }
  const stage = hex.parentElement;
  if (stage) stage.style.overflow = cam ? "hidden" : "";
  return scale;
}

function onFighterBar(ev) {
  const bar = ev.target.closest(".hp");
  const card = ev.target.closest(".fighter");
  if (!bar || !card) return;
  const m = /^fighter-(\d+)-(\d+)$/.exec(card.id);
  if (!m) return;
  const slot = Number(m[1]);
  const seat = Number(m[2]);
  if (follow && follow.slot === slot && follow.seat === seat) {
    follow = null;
    followHeld = false;
    panLargeMapTo(0, 0);
    return;
  }
  follow = { slot, seat };
  followHeld = false;
}

let introClip = null;
let introStart = 0;
let introPlayed = false;

function playIntro(scale, cx, cy) {
  if (!introClip && state.intro && state.intro.frames && state.intro.frames.length && state.intro.death >= 0) {
    introClip = state.intro;
    introStart = performance.now();
    introPlayed = false;
  }
  let layer = document.getElementById("intro-layer");
  if (!introClip || !(introClip.death >= 0) || !introClip.frames || !introClip.frames.length) {
    if (layer) layer.replaceChildren();
    return false;
  }
  const death = Number(introClip.death);
  const playDur = 0.5 + Math.max(0, death) / 10;
  const dur = playDur + 0.5;
  const elapsed = (performance.now() - introStart) / 1000;
  if (elapsed >= dur) {
    if (layer) layer.replaceChildren();
    if (!introPlayed) {
      introPlayed = true;
      send({ type: "intro" });
    }
    return false;
  }
  let t;
  if (elapsed < 0.5) t = Math.max(0, death - 0.5) + elapsed;
  else if (elapsed < playDur) t = Math.max(0, death - (elapsed - 0.5) * 10);
  else t = 0;
  let frame = introClip.frames[0];
  let best = Infinity;
  for (const f of introClip.frames) {
    const d = Math.abs(Number(f.t) - t);
    if (d < best) {
      best = d;
      frame = f;
    }
  }
  if (!layer && hex) {
    layer = document.createElement("div");
    layer.id = "intro-layer";
    hex.appendChild(layer);
  }
  if (!layer) return true;
  const bodies = frame.b || [];
  while (layer.childElementCount < bodies.length) {
    const el = document.createElement("div");
    el.className = "intro-ball";
    layer.appendChild(el);
  }
  while (layer.childElementCount > bodies.length) layer.lastChild.remove();
  bodies.forEach((b, i) => {
    const el = layer.children[i];
    const r = (b.r || 18) * scale;
    const [sx, sy] = screenPos(b.x, b.y, scale, cx, cy);
    el.style.width = `${r * 2}px`;
    el.style.height = `${r * 2}px`;
    el.style.left = `${sx}px`;
    el.style.top = `${sy}px`;
    el.style.background = kindColor(b.k);
    el.style.boxShadow = "0 0 10px " + kindColor(b.k);
    paintKind(el, b.k);
    applyLookFX(el, b.k);
    runLookFX(el, { id: "intro-" + i, kind: b.k, x: b.x, y: b.y, radius: b.r || 18, slot: b.s }, {});
  });
  return true;
}

function renderArena() {
  // 场皮外形/尺寸由各 map 私产 CSS 按 [data-field] 自绘，引擎不按 fieldShape 分支。
  if (state.field) hex.dataset.field = state.field;
  else delete hex.dataset.field;
  const cam = syncFollow();
  const scale = applyCamera(cam);
  const w = hex.clientWidth || 1;
  const h = hex.clientHeight || 1;
  const cx = w / 2;
  const cy = h / 2;
  if (playIntro(scale, cx, cy)) {
    if (unitsEl) unitsEl.style.visibility = "hidden";
    return;
  }
  if (unitsEl) unitsEl.style.visibility = "";
  hex.classList.toggle(
    "flooded",
    (state.units || []).some((u) => u.kind === "钓鱼佬场鱼塘"),
  );
  hex.classList.toggle("hitstop", (state.hitStop || 0) > 0);

  const fighters = (state.units || []).filter((u) => u.role === "fighter");
  renderHud(fighters);

  const pauseBtn = document.getElementById("btn-pause");
  pauseBtn.textContent = state.phase === "paused" ? "继续" : "暂停";

  renderGuides(fighters, scale, cx, cy);
  renderWalls(scale, cx, cy);
  playEffects(state.effects, scale, cx, cy);

  const seen = new Set();
  const now = performance.now();
  for (const u of state.units || []) {
    seen.add(String(u.id));
    if (!(u.radius > 0)) {
      const stale = document.getElementById(`u-${u.id}`);
      if (stale) stale.remove();
      const tag = document.getElementById(`hp-${u.id}`);
      if (tag) tag.remove();
      continue;
    }
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
      time: state.time || 0,
      scale,
      cx,
      cy,
      units: state.units || [],
      fxRoot: overEl || fxRoot,
      spawnGhost: spawnGhostFrom,
    });
    const shareHP = !!look.shareHp;
    const shownHP = shareHP
      ? fighters.find((f) => f.slot === u.slot)?.hp ?? u.hp
      : u.hp;
    const prev = prevHP.get(u.id);
    if (prev != null && shownHP < prev - 0.5) {
      burst(sx, sy, u.kind, 8);
    }
    const showHP = (u.role === "fighter" || shareHP) && !u.noHealthNumbers;
    if (showHP) prevHP.set(u.id, shownHP);
    let tag = document.getElementById(`hp-${u.id}`);
    if (showHP) {
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
      if (child.classList.contains("integrated-trails")) continue;
      const id = child.id.slice(2);
      if (!seen.has(id)) child.remove();
    }
  }

  if (state.phase === "ended") {
    banner.classList.remove("hidden");
    const bannerText = document.getElementById("banner-text");
    const line = state.winner === "平局" ? "平局" : `胜者：${state.winner}`;
    if (bannerText) bannerText.textContent = line;
    const left = document.getElementById("banner-left");
    const right = document.getElementById("banner-right");
    if (left) left.textContent = rosterStatus(0);
    if (right) right.textContent = rosterStatus(1);
  } else {
    banner.classList.add("hidden");
  }
}

let wasLobby = true;

function releaseLobby() {
  for (const id of ["roster-list-0", "roster-list-1", "kinds-0", "kinds-1"]) {
    const el = document.getElementById(id);
    if (el) el.replaceChildren();
  }
}

function render() {
  const inLobby = state.phase === "select";
  if (inLobby) {
    introClip = null;
    introPlayed = false;
  }
  if (inLobby && !wasLobby) {
    location.reload();
    return;
  }
  ensurePacks();
  lobby.classList.toggle("hidden", !inLobby);
  arenaEl.classList.toggle("hidden", inLobby);
  if (state.phase === "ended" || inLobby) window.youthBuff = {};
  if (inLobby) window.youthDoses = {};
  if (inLobby) {
    wasLobby = true;
    renderLobby();
  } else {
    wasLobby = false;
    if (lobbyKey !== "") {
      lobbyKey = "";
      releaseLobby();
    }
    renderArena();
  }
  renderLogChrome();
}
