window.lookFX = window.lookFX || {};

function ensureShadowHP(el) {
  let wrap = el.querySelector(":scope > .minion-hp");
  if (wrap) return wrap;
  wrap = document.createElement("span");
  wrap.className = "minion-hp";
  wrap.appendChild(document.createElement("i"));
  el.appendChild(wrap);
  return wrap;
}

function ensureShadowBody(el) {
  let core = el.querySelector(":scope > .shadow-core");
  if (!core) {
    core = document.createElement("span");
    core.className = "shadow-core";
    el.appendChild(core);
  }
  let ring = el.querySelector(":scope > .shadow-ring");
  if (!ring) {
    ring = document.createElement("span");
    ring.className = "shadow-ring";
    el.appendChild(ring);
  }
  let streak = el.querySelector(":scope > .shadow-streak");
  if (!streak) {
    streak = document.createElement("span");
    streak.className = "shadow-streak";
    el.appendChild(streak);
  }
  return { core, ring, streak };
}

window.lookFX.shadow = {
  unmount(el) {
    if (!el) return;
    el.querySelector(":scope > .minion-hp")?.remove();
    el.querySelector(":scope > .shadow-core")?.remove();
    el.querySelector(":scope > .shadow-ring")?.remove();
    el.querySelector(":scope > .shadow-streak")?.remove();
    el.classList.remove("is-dash", "is-orbit");
  },
  tick(el, u) {
    if (!el || !u) return;
    ensureShadowBody(el);
    const wrap = ensureShadowHP(el);
    const bar = wrap.querySelector("i");
    const max = u.maxHp || 1;
    const hp = Math.max(0, u.hp || 0);
    if (bar) bar.style.width = `${Math.max(0, Math.min(100, (hp / max) * 100))}%`;

    const vx = u.vx || 0;
    const vy = u.vy || 0;
    const sp = Math.hypot(vx, vy);
    // 冲刺（围刺/叠撞）与环转切向速度分段
    const dashing = sp > 480;
    const orbiting = !dashing && sp > 140;
    el.classList.toggle("is-dash", dashing);
    el.classList.toggle("is-orbit", orbiting);
    const ang = Math.atan2(-(vy || 0), vx || 1);
    el.style.setProperty("--ang", `${ang}rad`);
  },
};
