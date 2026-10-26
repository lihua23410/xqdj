window.lookFX = window.lookFX || {};

function ensureHP(el) {
  let wrap = el.querySelector(":scope > .minion-hp");
  if (wrap) return wrap;
  wrap = document.createElement("span");
  wrap.className = "minion-hp";
  const i = document.createElement("i");
  wrap.appendChild(i);
  el.appendChild(wrap);
  return wrap;
}

function tickHP(el, u) {
  if (!el || !u) return;
  const wrap = ensureHP(el);
  const i = wrap.querySelector("i");
  const max = u.maxHp || 1;
  const hp = Math.max(0, u.hp || 0);
  if (i) i.style.width = `${Math.max(0, Math.min(100, (hp / max) * 100))}%`;
}

window.lookFX.godfather = {
  tick(el) {
    if (!el) return;
    el.classList.toggle("look-godfather", true);
  },
};

window.lookFX.minion = {
  unmount(el) {
    el?.querySelector(":scope > .minion-hp")?.remove();
  },
  tick(el, u) {
    tickHP(el, u);
  },
};

window.lookFX.assassin = {
  tick(el, u) {
    if (!el) return;
    const dashing = Math.hypot(u.vx || 0, u.vy || 0) > 400;
    el.classList.toggle("assassin-dash", dashing);
  },
};

window.lookFX.sniper = {
  unmount(el) {
    el?.querySelectorAll(":scope > .sniper-laser").forEach((n) => n.remove());
  },
  tick(el, u, ctx) {
    if (!el) return;
    const st = window.godfatherAim && window.godfatherAim[u.id];
    const beams = [...el.querySelectorAll(":scope > .sniper-laser")];
    if (!st || st.p < 0) {
      beams.forEach((n) => n.remove());
      return;
    }
    while (beams.length < 2) {
      const i = document.createElement("i");
      i.className = "sniper-laser";
      el.appendChild(i);
      beams.push(i);
    }
    const dx = (st.tx || 0) - (u.x || 0);
    const dy = (st.ty || 0) - (u.y || 0);
    const ang = Math.atan2(-dy, dx || 1);
    const p = Math.max(0, Math.min(1, st.p || 0));
    const spread = (1 - p) * 0.35;
    const scale = (ctx && ctx.scale) || 1;
    const len = Math.max(80, 560 * scale);
    [-spread, spread].forEach((off, i) => {
      beams[i].style.setProperty("--ang", `${ang + off}rad`);
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

window.lookFX.dealer = {};
window.lookFX.drug = {
  tick(el) {
    if (!el) return;
    el.classList.add("drug-pack");
  },
};
