window.lookFX = window.lookFX || {};
window.udongeinHUD = window.udongeinHUD || {};

window.lookFX.udongein = {
  unmount(el) {
    if (!el) return;
    el.style.removeProperty("--udongein-energy");
    el.removeAttribute("data-udongein-dose");
    el.removeAttribute("data-udongein-drained");
    el.querySelector(":scope > .udongein-cards")?.remove();
  },
  tick(el, u) {
    if (!el) return;
    const st = window.udongeinHUD[u.id] || { energy: 5, dose: 0, cap: 5, cards: [], fill: 0 };
    const e = Math.max(0, Math.min(1, (st.energy || 0) / 5));
    el.style.setProperty("--udongein-energy", String(e));
    el.dataset.udongeinDose = String(st.dose || 0);
    el.dataset.udongeinDrained = (st.cap || 5) < 5 ? "1" : "0";
    let host = el.querySelector(":scope > .udongein-cards");
    if (!host) {
      host = document.createElement("i");
      host.className = "udongein-cards";
      el.appendChild(host);
    }
    const cards = Array.isArray(st.cards) ? st.cards : [];
    const fill = Math.max(0, Math.min(1, st.fill || 0));
    for (let i = 0; i < 5; i++) {
      let slot = host.children[i];
      if (!slot) {
        slot = document.createElement("i");
        slot.className = "udongein-card";
        host.appendChild(slot);
      }
      const sk = cards[i] || 0;
      slot.dataset.sk = String(sk);
      slot.classList.toggle("is-on", sk > 0);
      slot.classList.toggle("is-fill", sk <= 0 && i === cards.length);
      slot.style.setProperty("--p", sk > 0 ? "100%" : i === cards.length ? `${Math.round(fill * 100)}%` : "0%");
    }
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

window.lookFX["udongein-seek"] = {
  unmount() {},
  tick() {},
};

window.lookFX["udongein-blast"] = {
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
