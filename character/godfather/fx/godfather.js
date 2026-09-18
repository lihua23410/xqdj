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

window.lookFX.dealer = {};
window.lookFX.drug = {
  tick(el) {
    if (!el) return;
    el.classList.add("drug-pack");
  },
};
