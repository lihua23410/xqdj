window.lookFX = window.lookFX || {};
window.udongeinHUD = window.udongeinHUD || {};

window.lookFX.udongein = {
  unmount(el) {
    if (!el) return;
    el.style.removeProperty("--udongein-energy");
    el.removeAttribute("data-udongein-dose");
  },
  tick(el, u) {
    if (!el) return;
    const st = window.udongeinHUD[u.id] || { energy: 5, dose: 0 };
    const e = Math.max(0, Math.min(1, (st.energy || 0) / 5));
    el.style.setProperty("--udongein-energy", String(e));
    el.dataset.udongeinDose = String(st.dose || 0);
  },
};

window.lookFX["udongein-ghost"] = {
  unmount() {},
  tick() {},
};

window.lookFX["udongein-mind"] = {
  unmount() {},
  tick() {},
};

window.lookFX["udongein-shard"] = {
  unmount(el) {
    if (!el) return;
    el.style.removeProperty("opacity");
    el.style.removeProperty("--udongein-shard-fade");
  },
  tick(el, u) {
    if (!el) return;
    const st = window.udongeinHUD[u.id] || {};
    let a = st.fade;
    if (a == null) {
      const v = Math.hypot(u.vx || 0, u.vy || 0);
      a = Math.max(0, Math.min(1, v / 200));
    }
    a = Math.max(0, Math.min(1, a));
    el.style.opacity = String(a);
    el.style.setProperty("--udongein-shard-fade", String(a));
  },
};

window.lookFX["udongein-crown"] = {
  unmount(el) {
    if (!el) return;
    el._ux = el._uy = undefined;
  },
  tick(el, u) {
    if (!el) return;
    if (el._ux == null) {
      el._ux = u.x;
      el._uy = u.y;
    }
    const dist = Math.hypot((u.x || 0) - el._ux, (u.y || 0) - el._uy);
    const r = Math.min(60, 10 + dist * 0.22);
    const s = r / Math.max(1, u.radius || 10);
    el.style.transform = `translate(-50%, -50%) scale(${s})`;
  },
};

window.lookFX["udongein-gas"] = {
  unmount() {},
  tick() {},
};

window.lookFX["udongein-beam"] = {
  unmount(el) {
    if (!el) return;
    el.querySelector(":scope > .udongein-ray")?.remove();
  },
  tick(el, u, ctx) {
    if (!el) return;
    let ray = el.querySelector(":scope > .udongein-ray");
    if (!ray) {
      ray = document.createElement("div");
      ray.className = "udongein-ray";
      el.appendChild(ray);
    }
    const scale = (ctx && ctx.scale) || 1;
    const ang = Math.atan2(-(u.vy || 0), u.vx || 1);
    ray.style.setProperty("--beam-ang", `${ang}rad`);
    ray.style.setProperty("--beam-len", `${560 * scale}px`);
  },
};
