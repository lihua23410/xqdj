// 本包只读取已有快照与 aim/leave 事件；造型和动作不改变对局状态。
(function () {
const hooks = window.lookFX = window.lookFX || {};
const local = new WeakMap();
const minions = new Set(["教父暗杀者", "教父狙击者", "教父贩卖者"]);
let lastUnits;

const cores = {
  boss: `<circle class="gf-rim" cx="50" cy="50" r="46.5"/>
    <path class="gf-command-arc" d="M72 12A44 44 0 0 1 94 48M28 88A44 44 0 0 1 7 58"/>
    <path class="gf-command-mark" d="m34 63 20-25h10M43 68 64 42"/>`,
  assassin: `<circle class="gf-rim" cx="50" cy="50" r="46.5"/>
    <path class="gf-role-accent" d="M74 13A43 43 0 0 1 93 44"/>
    <path class="gf-assassin-mark" d="m31 64 17-24m-5 28 19-28m-7 22 13-19"/>`,
  sniper: `<circle class="gf-rim" cx="50" cy="50" r="46.5"/>
    <path class="gf-focus-arc" d="M26 79A38 38 0 0 1 72 19"/>
    <path class="gf-sniper-mark" d="m33 65 30-30M47 35h16v16M33 51v14h14"/>`,
  dealer: `<circle class="gf-rim" cx="50" cy="50" r="46.5"/>
    <path class="gf-role-accent" d="M73 13A43 43 0 0 1 93 45"/>
    <path class="gf-dealer-mark" d="M33 37h26v14H45v14h20"/>`,
};
const assassinMotion = `<path class="gf-assassin-wake" d="M22 36Q-4 32-38 36M15 50H-54M22 64Q-4 68-38 64"/>`;
const sniperPulse = `<path class="gf-recoil-mark" d="m85 43 6 7-6 7"/>
  <path class="gf-shot-pulse" d="M98 50h21m-15-8 9 4m-9 12 9-4"/>`;
const dealerRelease = `<circle class="gf-drop-wave" cx="50" cy="50" r="33"/>
  <g class="gf-drop-dust"><circle cx="34" cy="96" r="2.5"/><circle cx="46" cy="105" r="1.8"/>
    <circle cx="59" cy="99" r="2.7"/><circle cx="69" cy="108" r="1.6"/></g>`;
const powder = `<svg viewBox="0 0 100 100" aria-hidden="true">
  <ellipse cx="49" cy="74" rx="38" ry="10" fill="#06080a" opacity=".35"/>
  <path fill="#eee9dd" opacity=".16" d="m8 68 17-17 22-4 27 7 20 20-13 11-27 2-29-5Z"/>
  <path fill="#d4d0c5" d="m13 69 7-9 9-4 8-7 8-2 6-5 9 3 8 7 8 4 6 8 9 5-12 8-18 3-14-3-14 2-12-6Z"/>
  <path fill="#f5f2e9" d="M17 66q3-5 13-8l8-7 8-2 6-6 8 4 6 6 10 6 5 7-12 4-15-2-10 3-15-2Z"/>
  <path fill="#fffcf3" opacity=".75" d="m31 57 7-6 8-2 6-6 6 5-8 3-6 8-8 3Zm27-3 5-3 6 5 5 2-9 3-7-2Z"/>
  <path fill="#e2ded3" d="m18 68 12 1 14 2 10-3 15 2 12-4 10 3-15 7-16 1-12-2-14 2Z"/>
  <g fill="#fffdf6"><circle cx="26" cy="64" r="1.2"/><circle cx="35" cy="58" r="1"/><circle cx="40" cy="64" r="1.3"/>
    <circle cx="47" cy="57" r="1"/><circle cx="53" cy="50" r="1.1"/><circle cx="58" cy="63" r="1.4"/>
    <circle cx="63" cy="57" r="1"/><circle cx="70" cy="65" r="1.2"/><circle cx="76" cy="70" r="1"/></g>
  <g fill="#a8a69d" opacity=".6"><circle cx="32" cy="66" r=".9"/><circle cx="44" cy="60" r=".8"/>
    <circle cx="52" cy="64" r=".9"/><circle cx="63" cy="73" r="1"/><circle cx="81" cy="69" r=".8"/></g>
  <g fill="#f3f0e7"><path d="m7 52 4-2 3 4-5 2Zm11-11 3-2 2 3-4 2Zm10 40 5-1 1 3-5 2Zm52-39 3-1 2 4-4 1Zm8 15 4-2 2 4-4 2Zm-2 25 4-1 2 3-5 2ZM43 85l3-1 2 3-4 1ZM9 76l3-2 2 3-3 2Z"/></g>
  <g fill="#fffaf0" opacity=".75"><circle cx="23" cy="48" r="1.3"/><circle cx="72" cy="35" r="1.1"/>
    <circle cx="91" cy="73" r="1.5"/><circle cx="66" cy="84" r="1.4"/><circle cx="17" cy="84" r="1.1"/>
    <circle cx="6" cy="64" r="1.2"/><circle cx="38" cy="29" r="1.1"/></g>
</svg>`;

function art(el, role, overlay) {
  if (el.querySelector(":scope > .gf-core")) return;
  const mark = document.createElement("span");
  mark.className = "gf-core";
  mark.setAttribute("aria-hidden", "true");
  mark.innerHTML = `<svg viewBox="0 0 100 100">${cores[role]}</svg>`;
  el.appendChild(mark);
  if (overlay) {
    const layer = document.createElement("span");
    layer.className = `gf-action gf-${role}-action`;
    layer.setAttribute("aria-hidden", "true");
    layer.innerHTML = `<svg viewBox="0 0 100 100">${overlay}</svg>`;
    el.appendChild(layer);
  }
}
function frame(el, u, ctx) {
  const time = Number.isFinite(ctx.time) ? ctx.time : 0;
  if (ctx.units && ctx.units !== lastUnits) {
    lastUnits = ctx.units;
    const ids = new Set(ctx.units.map(n => String(n.id)));
    for (const [id, aim] of Object.entries(window.godfatherAim || {})) {
      if (!ids.has(id) || aim.time > time + 0.001) delete window.godfatherAim[id];
    }
  }
  let s = local.get(el);
  if (!s || time < s.time || s.id !== u.id) {
    s = { id: u.id, time, shot: -Infinity, drop: -Infinity, call: -Infinity };
    local.set(el, s);
    el.classList.remove("minion-leave");
  }
  s.time = time;
  return s;
}
function pulse(now, at, duration) {
  return Math.max(0, 1 - (now - at) / duration);
}
function aimOf(u) { return window.godfatherAim && window.godfatherAim[u.id]; }
function retiring(el, u) {
  // aim 的负值由现有退场流程发出；高速兜底照顾中途进入的观众。
  const leaving = !!aimOf(u)?.leaving || Math.hypot(u.vx || 0, u.vy || 0) > 680;
  el.classList.toggle("gf-retiring", leaving);
  return leaving;
}
function newOwned(s, units, u, kind, key) {
  const current = new Set((units || []).filter(n => n.ownerId === u.ownerId && n.kind === kind).map(n => n.id));
  const fresh = s[key] && [...current].some(id => !s[key].has(id));
  s[key] = current;
  return fresh;
}
function unmount(el) {
  if (!el) return;
  el.querySelectorAll(":scope > .gf-core, :scope > .gf-action, :scope > .sniper-laser").forEach(n => n.remove());
  el.classList.remove("assassin-dash", "gf-retiring", "minion-leave");
  for (const name of [...el.style]) if (name.startsWith("--gf-")) el.style.removeProperty(name);
  local.delete(el);
}

hooks.godfather = {
  guide(u, ctx) {
    window.godfatherSFX?.observe(u, ctx);
  },
  unmount,
  tick(el, u, ctx = {}) {
    if (!el || !u) return;
    art(el, "boss");
    const s = frame(el, u, ctx);
    const members = new Set((ctx.units || []).filter(n => n.ownerId === u.id && minions.has(n.kind)).map(n => n.id));
    if (s.members && [...members].some(id => !s.members.has(id))) s.call = s.time;
    s.members = members;
    el.style.setProperty("--gf-call", pulse(s.time, s.call, 0.6).toFixed(3));
  },
};
hooks.assassin = {
  unmount,
  tick(el, u, ctx = {}) {
    if (!el || !u) return;
    art(el, "assassin", assassinMotion);
    frame(el, u, ctx);
    const leaving = retiring(el, u);
    const speed = Math.hypot(u.vx || 0, u.vy || 0);
    el.classList.toggle("assassin-dash", !leaving && speed > 400);
    if (speed > 1) el.style.setProperty("--gf-angle", `${Math.atan2(-(u.vy || 0), u.vx || 0)}rad`);
  },
};
hooks.sniper = {
  unmount,
  tick(el, u, ctx = {}) {
    if (!el || !u) return;
    art(el, "sniper", sniperPulse);
    const s = frame(el, u, ctx);
    retiring(el, u);
    if (newOwned(s, ctx.units, u, "教父狙击弹", "rounds")) s.shot = s.time;
    const flash = pulse(s.time, s.shot, 0.18);
    el.style.setProperty("--gf-shot", flash.toFixed(3));
    el.style.setProperty("--gf-recoil", `${(-flash * 7).toFixed(2)}px`);
    const st = aimOf(u);
    const beams = [...el.querySelectorAll(":scope > .sniper-laser")];
    if (st && Number.isFinite(st.tx) && Number.isFinite(st.ty)) {
      el.style.setProperty("--gf-angle", `${Math.atan2(-(st.ty - (u.y || 0)), st.tx - (u.x || 0))}rad`);
    }
    const p = st && st.p >= 0 ? Math.max(0, Math.min(1, st.p)) : 0;
    el.style.setProperty("--gf-focus", p.toFixed(3));
    if (!st || st.p < 0) {
      beams.forEach(n => n.remove());
      return;
    }
    while (beams.length < 2) {
      const beam = document.createElement("i");
      beam.className = "sniper-laser";
      el.appendChild(beam);
      beams.push(beam);
    }
    const angle = Math.atan2(-(st.ty - (u.y || 0)), st.tx - (u.x || 0));
    const spread = (1 - p) * 0.35;
    const len = Math.max(80, 560 * (ctx.scale || 1));
    [-spread, spread].forEach((offset, i) => {
      beams[i].style.setProperty("--ang", `${angle + offset}rad`);
      beams[i].style.setProperty("--len", `${len}px`);
    });
  },
};

// 狙击弹的大口径铜被甲穿甲弹。弹体朝右画，页面按速度方向旋转整枚弹。
// 黑钢弹尖 + 铜被甲 + 弹尾短曳光，不做尾翼、不做火箭式长焰；不引用外部图片。
const AP_ROUND_SVG = `<svg viewBox="0 0 168 56" xmlns="http://www.w3.org/2000/svg" aria-hidden="true">
  <defs>
    <linearGradient id="ap-wake" x1="0" y1="0" x2="1" y2="0">
      <stop offset="0" stop-color="#ff3a22" stop-opacity="0"/>
      <stop offset=".6" stop-color="#ff4a2e" stop-opacity=".18"/>
      <stop offset="1" stop-color="#ffb26a" stop-opacity=".45"/>
    </linearGradient>
    <linearGradient id="ap-copper" x1="0" y1="1" x2="0" y2="0">
      <stop offset="0" stop-color="#542a15"/>
      <stop offset=".28" stop-color="#9a5328"/>
      <stop offset=".52" stop-color="#e6a05e"/>
      <stop offset=".7" stop-color="#b96b35"/>
      <stop offset="1" stop-color="#4a2412"/>
    </linearGradient>
    <linearGradient id="ap-core" x1="0" y1="1" x2="0" y2="0">
      <stop offset="0" stop-color="#07090b"/>
      <stop offset=".45" stop-color="#303942"/>
      <stop offset=".62" stop-color="#182026"/>
      <stop offset="1" stop-color="#050607"/>
    </linearGradient>
    <linearGradient id="ap-brass" x1="0" y1="1" x2="0" y2="0">
      <stop offset="0" stop-color="#6f5120"/>
      <stop offset=".46" stop-color="#e8c46a"/>
      <stop offset="1" stop-color="#a77c2d"/>
    </linearGradient>
    <radialGradient id="ap-glow" cx=".5" cy=".5" r=".5">
      <stop offset="0" stop-color="#ff8a62" stop-opacity=".38"/>
      <stop offset="1" stop-color="#ff4a2e" stop-opacity="0"/>
    </radialGradient>
  </defs>
  <path d="M2 28 Q14 25.5 32 26.5 L32 29.5 Q14 30.5 2 28 Z" fill="url(#ap-wake)"/>
  <path d="M8 28 H36" stroke="#ffb27a" stroke-width="1.1" stroke-linecap="round" opacity=".3"/>
  <ellipse cx="32" cy="28" rx="10" ry="5" fill="url(#ap-glow)" opacity=".5"/>
  <path d="M28 22 L44 18 H116 Q140 19 154 28 Q140 37 116 38 H44 L28 34 Z" fill="url(#ap-copper)" stroke="#130d09" stroke-width="2.2" stroke-linejoin="round"/>
  <path d="M46 19.6 H114" stroke="#ffe0b0" stroke-width="1.6" stroke-linecap="round" opacity=".8"/>
  <path d="M46 36.4 H114" stroke="#3a1d0d" stroke-width="1.3" stroke-linecap="round" opacity=".75"/>
  <path d="M62 18.6 V37.4 M69 18.4 V37.6" stroke="#4a2a16" stroke-width="1.2" opacity=".6"/>
  <rect x="38" y="19.4" width="5" height="17.2" rx="1.5" fill="url(#ap-brass)" opacity=".9"/>
  <path d="M28 22 V34" stroke="#2a160c" stroke-width="2.2" stroke-linecap="round"/>
  <path d="M126 19.6 Q143 21.2 154 28 Q143 34.8 126 36.4 Z" fill="url(#ap-core)" stroke="#050607" stroke-width="1.6"/>
  <path d="M129 21.2 Q144 22.8 151 27.2" stroke="#9aa6b2" stroke-width="1.1" stroke-linecap="round" opacity=".55"/>
  <path d="M148 25.8 L154 28 L148 30.2" fill="#dfe9f2" opacity=".55"/>
  <circle cx="34" cy="28" r="1.2" fill="#ffd49a" opacity=".65"/>
  <circle cx="23" cy="27" r="0.9" fill="#ff7a4a" opacity=".4"/>
</svg>`;

window.lookFX.ap = {
  unmount(el) {
    el?.querySelectorAll(":scope > .ap-round").forEach((n) => n.remove());
  },
  tick(el, u, ctx) {
    if (!el) return;
    let round = el.querySelector(":scope > .ap-round");
    if (!round) {
      round = document.createElement("span");
      round.className = "ap-round";
      round.setAttribute("aria-hidden", "true");
      round.innerHTML = AP_ROUND_SVG;
      el.appendChild(round);
    }
    const scale = (ctx && ctx.scale) || 1;
    const len = Math.max(26, 48 * scale);
    const vx = u.vx || 0;
    const vy = u.vy || 0;
    const ang = Math.hypot(vx, vy) > 1e-6 ? Math.atan2(-vy, vx) : 0;
    round.style.setProperty("--ap-len", `${len.toFixed(2)}px`);
    round.style.setProperty("--ap-h", `${(len * 0.34).toFixed(2)}px`);
    round.style.setProperty("--ap-ang", `${(ang * 180 / Math.PI).toFixed(2)}deg`);
  },
};

hooks.dealer = {
  unmount,
  tick(el, u, ctx = {}) {
    if (!el || !u) return;
    art(el, "dealer", dealerRelease);
    const s = frame(el, u, ctx);
    retiring(el, u);
    if (newOwned(s, ctx.units, u, "教父药物", "drugs")) s.drop = s.time;
    const drop = pulse(s.time, s.drop, 0.45);
    el.style.setProperty("--gf-drop", drop.toFixed(3));
    el.style.setProperty("--gf-dust-y", `${((1 - drop) * 9).toFixed(2)}px`);
  },
};
hooks.drug = {
  unmount(el) {
    el?.classList.remove("drug-pack");
    el?.querySelector(":scope > .gf-powder")?.remove();
  },
  tick(el) {
    if (!el) return;
    el.classList.add("drug-pack");
    if (!el.querySelector(":scope > .gf-powder")) {
      const pile = document.createElement("span");
      pile.className = "gf-powder";
      pile.setAttribute("aria-hidden", "true");
      pile.innerHTML = powder;
      el.appendChild(pile);
    }
  },
};
})();
