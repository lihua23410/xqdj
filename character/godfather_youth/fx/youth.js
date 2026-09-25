window.lookFX = window.lookFX || {};
window.youthDoses = window.youthDoses || {};
window.youthBuff = window.youthBuff || {};

function pips(el, n) {
  let row = el.querySelector(":scope > .youth-doses");
  if (!row) {
    row = document.createElement("span");
    row.className = "youth-doses";
    for (let i = 0; i < 3; i++) {
      const pip = document.createElement("i");
      const ang = (i / 3) * Math.PI * 2;
      pip.style.left = `${50 + Math.cos(ang) * 78}%`;
      pip.style.top = `${50 + Math.sin(ang) * 78}%`;
      row.appendChild(pip);
    }
    el.appendChild(row);
  }
  const slots = row.querySelectorAll("i");
  slots.forEach((pip, i) => pip.classList.toggle("full", i < n));
}

window.lookFX.youth = {
  unmount(el) {
    const id = el && el.dataset.youthId;
    if (id) {
      delete window.youthDoses[id];
      delete window.youthBuff[id];
    }
    el?.classList.remove("youth-buff");
    el?.querySelector(":scope > .youth-doses")?.remove();
    el?.querySelector(":scope > .youth-ring")?.remove();
  },
  tick(el, u, ctx) {
    if (!el || !u) return;
    el.dataset.youthId = String(u.id);
    el.classList.add("look-youth");
    const sp = Math.hypot(u.vx || 0, u.vy || 0);
    if (sp > 400 && ctx && ctx.spawnGhost) {
      const now = ctx.now || 0;
      if (now - (el._dashAt || 0) > 28) {
        el._dashAt = now;
        const g = ctx.spawnGhost(el, u.kind, ctx.fxRoot, "youth-ghost");
        if (g) g.classList.add("youth-ghost");
      }
    }
    const n = window.youthDoses[u.id];
    pips(el, n == null ? 3 : n);
    const on = !!window.youthBuff[u.id];
    el.classList.toggle("youth-buff", on);
    let ring = el.querySelector(":scope > .youth-ring");
    if (on) {
      if (!ring) {
        ring = document.createElement("i");
        ring.className = "youth-ring";
        el.appendChild(ring);
      }
      const ang = Math.atan2(-(u.vy || 0), u.vx || 1);
      ring.style.setProperty("--ang", `${ang}rad`);
    } else if (ring) {
      ring.remove();
    }
  },
};

window.lookFX["youth-shot"] = { tick() {}, unmount() {} };

arena.registerShot("doses", (fx) => {
  if (!fx || !fx.unitId) return;
  window.youthDoses[fx.unitId] = fx.amount;
});

arena.registerShot("buff", (fx) => {
  if (!fx || !fx.unitId) return;
  window.youthBuff[fx.unitId] = fx.amount > 0;
});
