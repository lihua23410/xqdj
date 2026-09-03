window.lookFX = window.lookFX || {};
window.lookFX.slash = {
  tick(el, u) {
    if (!el || !el.classList.contains("look-slash")) return;
    let host = el.querySelector(":scope > .look-iai");
    if (!host) {
      host = document.createElement("div");
      host.className = "look-iai";
      for (const name of ["iai-glow", "iai-blade", "iai-core", "iai-sheen"]) {
        const layer = document.createElement("i");
        layer.className = name;
        host.appendChild(layer);
      }
      el.appendChild(host);
    }
    const ang = Math.atan2(-(u.vy || 0), u.vx || 1);
    const hexEl = document.getElementById("hex");
    const len = (hexEl ? hexEl.clientWidth : 560) * 2.2;
    host.style.setProperty("--slash-ang", `${ang}rad`);
    host.style.setProperty("--slash-len", `${len}px`);
    host.style.width = `${len}px`;
  },
};
