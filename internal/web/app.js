const fxRoot = document.getElementById("fx");
let prevHP = new Map();
let dashGhostAt = new Map();

function screenPos(x, y, scale, cx, cy) {
  return [cx + x * scale, cy - y * scale];
}

function slotColor(slot) {
  return slot === 1 ? "var(--slot1)" : "var(--slot0)";
}

function spawnFx(cls, x, y, slot, extra = {}) {
  if (!fxRoot) return null;
  const el = document.createElement("div");
  el.className = `fx ${cls}`;
  el.style.left = `${x}px`;
  el.style.top = `${y}px`;
  el.style.color = slotColor(slot);
  for (const [k, v] of Object.entries(extra)) el.style.setProperty(k, v);
  fxRoot.appendChild(el);
  el.addEventListener("animationend", () => el.remove());
  return el;
}

function burst(x, y, slot, n = 10) {
  for (let i = 0; i < n; i++) {
    const a = (Math.PI * 2 * i) / n + Math.random() * 0.4;
    const d = 18 + Math.random() * 28;
    spawnFx("fx-spark", x, y, slot, {
      "--dx": `${Math.cos(a) * d}px`,
      "--dy": `${Math.sin(a) * d}px`,
    });
  }
}

function playEffects(effects, scale, cx, cy) {
  for (const fx of effects || []) {
    const [x, y] = screenPos(fx.x, fx.y, scale, cx, cy);
    if (fx.name === "dash") {
      spawnFx("fx-shock", x, y, fx.slot);
      spawnFx("fx-ring", x, y, fx.slot);
      burst(x, y, fx.slot, 12);
    } else if (fx.name === "shot") {
      spawnFx("fx-flash", x, y, fx.slot);
      const ang = Math.atan2(-(fx.vy || 0), fx.vx || 1) * (180 / Math.PI);
      const beam = spawnFx("fx-beam", x, y, fx.slot);
      beam.style.transform = `translate(0, -50%) rotate(${ang}deg)`;
    } else if (fx.name === "hurt") {
      const el = spawnFx("fx-dmg", x, y - 12, fx.slot);
      if (el) el.textContent = `-${Math.round(fx.amount || 0)}`;
    } else if (fx.name === "impact" || fx.name === "hit") {
      spawnFx("fx-flash", x, y, fx.slot);
      spawnFx("fx-shock", x, y, fx.slot);
      burst(x, y, fx.slot, fx.name === "impact" ? 14 : 8);
    }
  }
}
const lobby = document.getElementById("lobby");
const arena = document.getElementById("arena");
const hex = document.getElementById("hex");
const unitsEl = document.getElementById("units");
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

function renderLobby() {
  const kinds = state.kinds || [];
  for (const [el, slot] of [
    [kinds0, 0],
    [kinds1, 1],
  ]) {
    el.innerHTML = "";
    for (const kind of kinds) {
      const b = document.createElement("button");
      b.textContent = kind;
      b.dataset.slot = String(slot);
      if (state.slots && state.slots[slot] === kind) b.classList.add("active");
      b.onclick = () => send({ type: "select", slot, kind });
      el.appendChild(b);
    }
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
      continue;
    }
    name.textContent = f.kind;
    num.textContent = `${Math.round(f.hp)} / ${Math.round(f.maxHp)}`;
    bar.style.width = `${Math.max(0, (f.hp / f.maxHp) * 100)}%`;
  }

  const pauseBtn = document.getElementById("btn-pause");
  pauseBtn.textContent = state.phase === "paused" ? "继续" : "暂停";

  playEffects(state.effects, scale, cx, cy);

  const seen = new Set();
  const now = performance.now();
  for (const u of state.units || []) {
    seen.add(String(u.id));
    let el = document.getElementById(`u-${u.id}`);
    if (!el) {
      el = document.createElement("div");
      el.id = `u-${u.id}`;
      el.className = "ball";
      unitsEl.appendChild(el);
    }
    el.classList.toggle("projectile", u.role === "projectile");
    el.classList.toggle("slot-0", u.slot === 0);
    el.classList.toggle("slot-1", u.slot === 1);
    const speed = Math.hypot(u.vx || 0, u.vy || 0);
    const dashing = u.kind === "原型机_近战" && speed > 280;
    el.classList.toggle("dashing", dashing);
    const r = u.radius * scale;
    el.style.width = `${r * 2}px`;
    el.style.height = `${r * 2}px`;
    const [sx, sy] = screenPos(u.x, u.y, scale, cx, cy);
    el.style.left = `${sx}px`;
    el.style.top = `${sy}px`;
    el.style.transform = "translate(-50%, -50%)";
    if (dashing) {
      const last = dashGhostAt.get(u.id) || 0;
      if (now - last > 32) {
        dashGhostAt.set(u.id, now);
        const g = document.createElement("div");
        g.className = `ghost slot-${u.slot}`;
        g.style.width = el.style.width;
        g.style.height = el.style.height;
        g.style.left = el.style.left;
        g.style.top = el.style.top;
        g.style.background = getComputedStyle(el).background;
        fxRoot.appendChild(g);
        g.addEventListener("animationend", () => g.remove());
      }
    }
    const prev = prevHP.get(u.id);
    if (prev != null && u.hp < prev - 0.5) {
      burst(sx, sy, u.slot, 8);
    }
    if (u.role === "fighter") prevHP.set(u.id, u.hp);
    let tag = el.querySelector(".hp-float");
    if (u.role === "fighter") {
      if (!tag) {
        tag = document.createElement("span");
        tag.className = "hp-float";
        el.appendChild(tag);
      }
      tag.textContent = String(Math.round(u.hp));
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
      ring.classList.toggle("slot-0", u.slot === 0);
      ring.classList.toggle("slot-1", u.slot === 1);
      const locked = (state.units || []).some(
        (o) =>
          o.id !== u.id &&
          o.role === "fighter" &&
          Math.hypot(o.x - u.x, o.y - u.y) <= u.vision
      );
      ring.classList.toggle("alert", locked);
    } else if (ring) {
      ring.remove();
    }
  }
  for (const child of [...unitsEl.children]) {
    if (child.classList.contains("seek-ring")) {
      const id = child.id.slice("ring-".length);
      if (!seen.has(id)) child.remove();
      continue;
    }
    const id = child.id.slice(2);
    if (!seen.has(id)) child.remove();
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
    renderLobby();
  } else renderArena();
}
