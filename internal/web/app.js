const fxRoot = document.getElementById("fx");
let prevHP = new Map();
let dashGhostAt = new Map();

const KIND_COLORS = {
  原型机_近战: "var(--kind-melee)",
  原型机_远程: "var(--kind-ranged)",
  分身者: "var(--kind-doppel)",
  分身: "var(--kind-doppel)",
  筑墙者: "var(--kind-waller)",
  无下限术士: "var(--kind-twin)",
  无下限: "var(--kind-twin-half)",
  紫弹: "var(--kind-violet)",
  子弹: "var(--kind-ranged)",
};

function kindColor(kind) {
  return KIND_COLORS[kind] || "var(--kind-melee)";
}

function screenPos(x, y, scale, cx, cy) {
  return [cx + x * scale, cy - y * scale];
}

function spawnFx(cls, x, y, kind, extra = {}) {
  if (!fxRoot) return null;
  const el = document.createElement("div");
  el.className = `fx ${cls}`;
  el.style.left = `${x}px`;
  el.style.top = `${y}px`;
  el.style.color = kindColor(kind);
  for (const [k, v] of Object.entries(extra)) el.style.setProperty(k, v);
  fxRoot.appendChild(el);
  el.addEventListener("animationend", () => el.remove());
  return el;
}

function spawnGhostFrom(el, kind, layer, extraClass = "") {
  const root = layer || fxRoot;
  if (!root || !el) return;
  const g = document.createElement("div");
  g.className = `ghost${extraClass ? ` ${extraClass}` : ""}`;
  if (el.classList.contains("semi")) g.classList.add("semi");
  g.style.setProperty("--kind", kindColor(kind));
  g.style.setProperty("--face-ang", el.style.getPropertyValue("--face-ang") || "0rad");
  g.style.width = el.style.width;
  g.style.height = el.style.height;
  g.style.left = el.style.left;
  g.style.top = el.style.top;
  g.style.transform = el.style.transform;
  root.appendChild(g);
  g.addEventListener("animationend", () => g.remove());
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
    if (fx.name === "dash") {
      spawnFx("fx-shock", x, y, kind);
      spawnFx("fx-ring", x, y, kind);
      burst(x, y, kind, 12);
    } else if (fx.name === "split") {
      spawnFx("fx-flash", x, y, kind);
      spawnFx("fx-shock", x, y, kind);
      spawnFx("fx-ring", x, y, kind);
      burst(x, y, kind, 14);
    } else if (fx.name === "shot") {
      spawnFx("fx-flash", x, y, kind);
      const ang = Math.atan2(-(fx.vy || 0), fx.vx || 1) * (180 / Math.PI);
      const beam = spawnFx("fx-beam", x, y, kind);
      beam.style.transform = `translate(0, -50%) rotate(${ang}deg)`;
      if (kind === "紫弹") {
        spawnFx("fx-shock", x, y, kind);
        spawnFx("fx-ring", x, y, kind);
        burst(x, y, kind, 16);
      }
    } else if (fx.name === "hurt") {
      const el = spawnFx("fx-dmg", x, y - 12, kind);
      if (el) el.textContent = `-${Math.round(fx.amount || 0)}`;
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
      b.style.setProperty("--kind", kindColor(kind));
      if (state.slots && state.slots[slot] === kind) b.classList.add("active");
      b.onclick = () => send({ type: "select", slot, kind });
      el.appendChild(b);
    }
    const slotRoot = el.closest(".slot");
    if (slotRoot) slotRoot.style.setProperty("--kind", kindColor(state.slots[slot]));
  }
}

const WALL_GUIDE_LEN = 150;

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
  const wallers = (fighters || []).filter((u) => u.kind === "筑墙者");
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
    placeSeg(link, ax, ay, bx, by, "筑墙者");
    seen.add(link.id);

    const dx = enemy.x - w.x;
    const dy = enemy.y - w.y;
    const n = Math.hypot(dx, dy);
    if (n < 1e-6) continue;
    const px = -dy / n;
    const py = dx / n;
    const half = WALL_GUIDE_LEN / 2;
    const mx = (w.x + enemy.x) / 2;
    const my = (w.y + enemy.y) / 2;
    const [g1x, g1y] = screenPos(mx + px * half, my + py * half, scale, cx, cy);
    const [g2x, g2y] = screenPos(mx - px * half, my - py * half, scale, cx, cy);
    const hint = ensureGuide(`guide-wall-${pair}`, "hint");
    placeSeg(hint, g1x, g1y, g2x, g2y, "筑墙者");
    seen.add(hint.id);
  }
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
    if (!f) {
      name.textContent = state.slots[slot] || "";
      num.textContent = "0 / 100";
      bar.style.width = "0%";
      root.style.setProperty("--kind", kindColor(state.slots[slot]));
      continue;
    }
    name.textContent = f.kind;
    num.textContent = `${Math.round(f.hp)} / ${Math.round(f.maxHp)}`;
    bar.style.width = `${Math.max(0, (f.hp / f.maxHp) * 100)}%`;
    root.style.setProperty("--kind", kindColor(f.kind));
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
    let el = document.getElementById(`u-${u.id}`);
    const layer = u.passWalls && overEl ? overEl : unitsEl;
    if (!el) {
      el = document.createElement("div");
      el.id = `u-${u.id}`;
      el.className = "ball";
      layer.appendChild(el);
    } else if (el.parentNode !== layer) {
      layer.appendChild(el);
    }
    el.classList.toggle("projectile", u.role === "projectile");
    el.classList.toggle("clone", u.role === "clone");
    el.classList.toggle("semi", !!u.semi);
    el.style.setProperty("--kind", kindColor(u.kind));
    const speed = Math.hypot(u.vx || 0, u.vy || 0);
    const dashing = u.kind === "原型机_近战" && speed > 280;
    el.classList.toggle("dashing", dashing);
    const r = u.radius * scale;
    el.style.width = `${r * 2}px`;
    el.style.height = `${r * 2}px`;
    const [sx, sy] = screenPos(u.x, u.y, scale, cx, cy);
    el.style.left = `${sx}px`;
    el.style.top = `${sy}px`;
    const faceX = u.faceX || u.vx || 1;
    const faceY = u.faceY || u.vy || 0;
    const faceAng = u.semi ? Math.atan2(-faceY, faceX) : 0;
    el.style.setProperty("--face-ang", `${faceAng}rad`);
    el.style.transform = u.semi
      ? `translate(-50%, -50%) rotate(${faceAng}rad)`
      : "translate(-50%, -50%)";
    let ghostGap = 0;
    let ghostLayer = fxRoot;
    let ghostCls = "";
    if (dashing) {
      ghostGap = 32;
    } else if (u.kind === "紫弹") {
      ghostGap = 16;
      ghostLayer = overEl || fxRoot;
      ghostCls = "trail";
    } else if (u.semi && speed > 40) {
      ghostGap = 40;
    }
    if (ghostGap) {
      const last = dashGhostAt.get(u.id) || 0;
      if (now - last > ghostGap) {
        dashGhostAt.set(u.id, now);
        spawnGhostFrom(el, u.kind, ghostLayer, ghostCls);
      }
    }
    const shownHP =
      u.role === "twin"
        ? fighters.find((f) => f.slot === u.slot)?.hp ?? u.hp
        : u.hp;
    const prev = prevHP.get(u.id);
    if (prev != null && shownHP < prev - 0.5) {
      burst(sx, sy, u.kind, 8);
    }
    if (u.role === "fighter" || u.role === "twin") prevHP.set(u.id, shownHP);
    let tag = el.querySelector(".hp-float");
    if (u.role === "fighter" || u.role === "twin") {
      if (!tag) {
        tag = document.createElement("span");
        tag.className = "hp-float";
        el.appendChild(tag);
      }
      tag.textContent = String(Math.round(shownHP));
    } else if (tag) {
      tag.remove();
    }
    let ring = document.getElementById(`ring-${u.id}`);
    if (u.kind === "原型机_近战" && u.vision > 0) {
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
    let field = document.getElementById(`field-${u.id}`);
    if (u.semi && (u.kind === "无下限术士" || u.kind === "无下限")) {
      if (!field) {
        field = document.createElement("div");
        field.id = `field-${u.id}`;
        field.className = "field-ring";
        unitsEl.appendChild(field);
      }
      field.classList.toggle("pull", u.kind === "无下限术士");
      field.classList.toggle("push", u.kind === "无下限");
      const fr = 72 * scale;
      field.style.width = `${fr * 2}px`;
      field.style.height = `${fr * 2}px`;
      field.style.left = `${sx}px`;
      field.style.top = `${sy}px`;
      field.style.setProperty("--kind", kindColor(u.kind));
    } else if (field) {
      field.remove();
    }
  }
  for (const root of [unitsEl, overEl]) {
    if (!root) continue;
    for (const child of [...root.children]) {
      if (child.classList.contains("ghost")) continue;
      if (child.classList.contains("seek-ring") || child.classList.contains("field-ring")) {
        const id = child.id.replace(/^(ring|field)-/, "");
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
