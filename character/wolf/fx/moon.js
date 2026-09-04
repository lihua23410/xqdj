window.lookFX = window.lookFX || {};
window.wolfMoonPhase = window.wolfMoonPhase || {};

window.lookFX.moon = {
  unmount(el) {
    el?.querySelector(":scope > .moon-face")?.remove();
    el?.querySelector(":scope > .moon-shade")?.remove();
  },
  tick(el, u) {
    if (!el || !el.classList.contains("look-moon")) return;
    let face = el.querySelector(":scope > .moon-face");
    if (!face) {
      face = document.createElement("i");
      face.className = "moon-face";
      el.appendChild(face);
    }
    let shade = el.querySelector(":scope > .moon-shade");
    if (!shade) {
      shade = document.createElement("i");
      shade.className = "moon-shade";
      el.appendChild(shade);
    }
    const phase = Math.max(0, Math.min(7, (window.wolfMoonPhase[u.slot] | 0)));
    el.dataset.moon = String(phase);
    const wax = [0, 28, 50, 78, 100, 78, 50, 28];
    const dir = phase > 4 ? 1 : -1;
    const lit = phase === 4;
    const dark = phase === 0;
    el.style.setProperty("--moon-shift", dark || lit ? "0%" : `${dir * wax[phase]}%`);
    el.style.setProperty("--moon-shade", lit ? "0" : "1");
  },
};
