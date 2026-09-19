window.lookFX = window.lookFX || {};

const HAMMER_LOOK_SVG = `<svg viewBox="0 0 64 48" xmlns="http://www.w3.org/2000/svg" aria-hidden="true">
  <rect x="3" y="20" width="34" height="8" rx="2.2" fill="#6a4424" stroke="#2a180c" stroke-width="1.3"/>
  <line x1="8" y1="24" x2="34" y2="24" stroke="#8a6238" stroke-width="1" opacity="0.55"/>
  <rect x="32" y="8" width="26" height="32" rx="2" fill="#8a5a30" stroke="#2a180c" stroke-width="1.4"/>
  <line x1="38" y1="10" x2="38" y2="38" stroke="#5a3418" stroke-width="1.1" opacity="0.55"/>
  <line x1="46" y1="10" x2="46" y2="38" stroke="#c4a070" stroke-width="1" opacity="0.35"/>
  <rect x="33" y="9" width="24" height="5" rx="1" fill="#c4a070" opacity="0.22"/>
  <rect x="30" y="21" width="8" height="7" rx="1" fill="#5a3418" stroke="#2a180c" stroke-width="1"/>
</svg>`;

const NAIL_LOOK_SVG = `<svg viewBox="0 0 88 12" xmlns="http://www.w3.org/2000/svg" aria-hidden="true">
  <rect x="1" y="2" width="9" height="8" fill="#9a9a9a" stroke="#2a2a2a" stroke-width="0.9"/>
  <rect x="2.2" y="3.2" width="6.6" height="2.2" fill="#e8e8e8" opacity="0.7"/>
  <rect x="2.2" y="6.4" width="6.6" height="2.2" fill="#5a5a5a" opacity="0.45"/>
  <path d="M9 4.6 H68 V7.4 H9 Z" fill="#b8b8b8" stroke="#2a2a2a" stroke-width="0.8"/>
  <path d="M12 5.15 H66" stroke="#f0f0f0" stroke-width="0.7" opacity="0.7"/>
  <path d="M12 6.85 H66" stroke="#606060" stroke-width="0.55" opacity="0.55"/>
  <polygon points="68,4.4 68,7.6 87,6" fill="#d0d0d0" stroke="#2a2a2a" stroke-width="0.8"/>
  <path d="M70 5.3 L84 6 L70 6.7" fill="#fff" opacity="0.35"/>
</svg>`;

const STAND_SVG = `<svg viewBox="0 0 64 88" xmlns="http://www.w3.org/2000/svg" aria-hidden="true">
  <path d="M24 54 L17 84" stroke="#7a5828" stroke-width="5.2" stroke-linecap="round"/>
  <path d="M40 54 L47 84" stroke="#8a6430" stroke-width="5.2" stroke-linecap="round"/>
  <path d="M22 62 L14 80" stroke="#c4a060" stroke-width="1.1" opacity="0.7"/>
  <path d="M42 62 L50 80" stroke="#c4a060" stroke-width="1.1" opacity="0.7"/>
  <ellipse cx="32" cy="46" rx="13" ry="17" fill="#c4a060" stroke="#5a3a18" stroke-width="1.2"/>
  <path d="M24 36 Q32 46 24 58" fill="none" stroke="#8a6230" stroke-width="0.8"/>
  <path d="M40 36 Q32 46 40 58" fill="none" stroke="#8a6230" stroke-width="0.8"/>
  <path d="M28 34 Q32 48 30 60" fill="none" stroke="#d8bc78" stroke-width="0.7" opacity="0.8"/>
  <path d="M20 42 L6 34" stroke="#b08a48" stroke-width="4.4" stroke-linecap="round"/>
  <path d="M44 42 L58 34" stroke="#b08a48" stroke-width="4.4" stroke-linecap="round"/>
  <path d="M8 36 L4 30" stroke="#d4b878" stroke-width="1.1"/>
  <path d="M56 36 L60 30" stroke="#d4b878" stroke-width="1.1"/>
  <ellipse cx="32" cy="20" rx="10.5" ry="11.5" fill="#d4b878" stroke="#5a3a18" stroke-width="1.2"/>
  <path d="M24 18 H40" stroke="#4a3018" stroke-width="1.6"/>
  <path d="M23 22 H41" stroke="#6a4a22" stroke-width="0.8" opacity="0.7"/>
  <path d="M27 18 L29 21" stroke="#1a1008" stroke-width="1.5" stroke-linecap="round"/>
  <path d="M37 18 L35 21" stroke="#1a1008" stroke-width="1.5" stroke-linecap="round"/>
  <path d="M28 25 L36 25" stroke="#1a1008" stroke-width="1.1" stroke-dasharray="1.6 1.1"/>
  <ellipse cx="32" cy="30" rx="6.5" ry="2.6" fill="none" stroke="#4a3018" stroke-width="1.8"/>
  <rect x="27" y="36" width="10" height="15" fill="#f4ead4" stroke="#1a1208" stroke-width="0.8"/>
  <path d="M29 39 h6 M28.5 43 h7 M30 43 v6 M29 47 h6" stroke="#1a1208" stroke-width="0.9" fill="none"/>
</svg>`;

function velAng(u) {
  const dx = Number(u && u.vx);
  const dy = Number(u && u.vy);
  const x = Number.isFinite(dx) ? dx : 0;
  const y = Number.isFinite(dy) ? dy : 0;
  if (Math.hypot(x, y) < 1e-6) return null;
  return Math.atan2(-y, x);
}

function keepAng(el, key, u, fallback) {
  const ang = velAng(u);
  if (ang == null) {
    const prev = el[key];
    return Number.isFinite(prev) ? prev : fallback;
  }
  el[key] = ang;
  return ang;
}

function nailUnits(ctx) {
  if (ctx && Array.isArray(ctx.units)) return ctx.units;
  if (typeof state !== "undefined" && state && Array.isArray(state.units)) return state.units;
  return [];
}

function foeOfNail(u, ctx) {
  let best = null;
  let bestD = Infinity;
  for (const o of nailUnits(ctx)) {
    if (!o || o.id === u.id || o.role !== "fighter" || o.slot === u.slot) continue;
    const d = Math.hypot((o.x || 0) - (u.x || 0), (o.y || 0) - (u.y || 0));
    if (d < bestD) {
      bestD = d;
      best = o;
    }
  }
  return best && bestD > 1e-4 ? { foe: best, dist: bestD } : null;
}

function nailAng(el, u, ctx) {
  let dx = Number(u && u.vx);
  let dy = Number(u && u.vy);
  if (!Number.isFinite(dx)) dx = 0;
  if (!Number.isFinite(dy)) dy = 0;
  const x = Number(u && u.x);
  const y = Number(u && u.y);
  if (Math.hypot(dx, dy) < 8) {
    const around = foeOfNail(u, ctx);
    const first = !Number.isFinite(el._nailAng);
    const posMoved = Number.isFinite(el._nailX) && Number.isFinite(x) && Number.isFinite(y) &&
      Math.hypot(x - el._nailX, y - el._nailY) > 0.5;
    if (around && around.dist > 20 && around.dist < 120 && (first || posMoved)) {
      dx = around.foe.x - u.x;
      dy = around.foe.y - u.y;
    } else if (Number.isFinite(el._nailAng)) {
      if (Number.isFinite(x)) el._nailX = x;
      if (Number.isFinite(y)) el._nailY = y;
      return el._nailAng;
    }
  }
  if (Number.isFinite(x)) el._nailX = x;
  if (Number.isFinite(y)) el._nailY = y;
  if (Math.hypot(dx, dy) < 1e-6) {
    return Number.isFinite(el._nailAng) ? el._nailAng : Math.PI / 2;
  }
  el._nailAng = Math.atan2(-dy, dx);
  return el._nailAng;
}

function ensureChild(el, cls, html) {
  let node = el.querySelector(`:scope > .${cls}`);
  if (!node) {
    node = document.createElement("span");
    node.className = cls;
    if (html) node.innerHTML = html;
    el.appendChild(node);
  }
  return node;
}

window.lookFX.hammer = {
  unmount(el) {
    el?.querySelector(":scope > .hammer-art")?.remove();
    el?.querySelector(":scope > .ushi-wrap")?.remove();
  },
  tick(el, u) {
    if (!el || !el.classList.contains("look-hammer")) return;
    el.querySelector(":scope > .ushi-wrap")?.remove();
    const art = ensureChild(el, "hammer-art");
    let spin = art.querySelector(":scope > .hammer-spin");
    if (!spin) {
      spin = document.createElement("span");
      spin.className = "hammer-spin";
      spin.innerHTML = HAMMER_LOOK_SVG;
      art.appendChild(spin);
    }
    const ang = keepAng(el, "_hammerAng", u, 0);
    art.style.setProperty("--hammer-ang", `${ang}rad`);
    art.style.transform = `translate(-50%, -50%) rotate(${ang}rad)`;
  },
};

window.lookFX.nail = {
  unmount(el) {
    el?.querySelector(":scope > .nail-art")?.remove();
    if (el) {
      el.style.removeProperty("--nail-ang");
      el.style.removeProperty("--nail-len");
      el.style.removeProperty("--nail-th");
    }
  },
  tick(el, u, ctx) {
    if (!el || !el.classList.contains("look-nail")) return;
    const art = ensureChild(el, "nail-art", NAIL_LOOK_SVG);
    const scale = (ctx && ctx.scale) || 1;
    const ang = nailAng(el, u, ctx);
    el.style.setProperty("--nail-ang", `${ang}rad`);
    el.style.setProperty("--nail-len", `${Math.max(40, 58 * scale)}px`);
    el.style.setProperty("--nail-th", `${Math.max(7, 11 * scale)}px`);
    art.style.removeProperty("transform");
  },
};

window.lookFX["hammer-doll"] = {
  unmount(el) {
    el?.querySelector(":scope > .stand-art")?.remove();
    el?.querySelector(":scope > .stand-aura")?.remove();
    el?.querySelector(":scope > .stand-hp")?.remove();
  },
  tick(el, u) {
    if (!el || !el.classList.contains("look-hammer-doll")) return;
    ensureChild(el, "stand-aura");
    ensureChild(el, "stand-art", STAND_SVG);
    const bar = ensureChild(el, "stand-hp", "<i></i>");
    const fill = bar.querySelector("i");
    const max = u.maxHp || 1;
    const hp = Math.max(0, u.hp || 0);
    if (fill) fill.style.width = `${Math.max(0, Math.min(100, (hp / max) * 100))}%`;
  },
};
