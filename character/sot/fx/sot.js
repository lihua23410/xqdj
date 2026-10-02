window.lookFX = window.lookFX || {};

window.lookFX.sot = {};

window.lookFX["sot-wine"] = {
  spin: {},
  last: {},
  tick(el, u, ctx) {
    if (!el || !el.classList.contains("look-sot-wine")) return;
    if (!el.querySelector(":scope > .sot-jar")) {
      const jar = document.createElement("i");
      jar.className = "sot-jar";
      el.appendChild(jar);
    }
    const id = u && u.id;
    if (id == null) return;
    if (!this.spin[id]) {
      const dir = Math.random() < 0.5 ? -1 : 1;
      this.spin[id] = (1.6 + Math.random() * 4.2) * dir;
    }
    const now = (ctx && ctx.now) || 0;
    const prev = this.last[id] || now;
    const dt = Math.max(0, Math.min(0.05, now - prev));
    this.last[id] = now;
    const cur = Number(el.dataset.sotAng) || 0;
    const next = cur + this.spin[id] * dt;
    el.dataset.sotAng = String(next);
    el.style.setProperty("--sot-ang", `${next}rad`);
  },
  unmount(el) {
    el?.querySelector(":scope > .sot-jar")?.remove();
  },
};

window.lookFX["sot-stain"] = {
  tick(el) {
    if (!el || !el.classList.contains("look-sot-stain")) return;
    if (el.querySelector(":scope > .sot-jar")) return;
    const jar = document.createElement("i");
    jar.className = "sot-jar";
    el.appendChild(jar);
  },
  unmount(el) {
    el?.querySelector(":scope > .sot-jar")?.remove();
  },
};
