window.lookFX = window.lookFX || {};

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
    el.querySelector(":scope > .shadow-core")?.remove();
    el.querySelector(":scope > .shadow-ring")?.remove();
    el.querySelector(":scope > .shadow-streak")?.remove();
    el.classList.remove("is-dash", "is-orbit");
  },
  tick(el, u) {
    if (!el || !u) return;
    ensureShadowBody(el);
    const vx = u.vx || 0;
    const vy = u.vy || 0;
    const sp = Math.hypot(vx, vy);
    // 分段须避开巡航 170 与走位 420：巡航不当环转，走位不当冲刺。
    // 环转切向约 251–367；冲刺 ≥640（与 Look.Ghost=500 对齐）。
    const dashing = sp > 500;
    const orbiting = !dashing && sp > 200;
    el.classList.toggle("is-dash", dashing);
    el.classList.toggle("is-orbit", orbiting);
    const ang = Math.atan2(-(vy || 0), vx || 1);
    el.style.setProperty("--ang", `${ang}rad`);
  },
};
