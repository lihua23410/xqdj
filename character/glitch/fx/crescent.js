window.lookFX = window.lookFX || {};
window.lookFX.crescent = {
  unmount(el) {
    el?.querySelector(":scope > .crescent-blade")?.remove();
  },
  tick(el, u) {
    if (!el || !el.classList.contains("look-crescent")) return;
    let blade = el.querySelector(":scope > .crescent-blade");
    if (!blade) {
      blade = document.createElement("i");
      blade.className = "crescent-blade";
      el.appendChild(blade);
    }
    const ang = Math.atan2(-(u.vy || 0), u.vx || 1);
    blade.style.setProperty("--crescent-ang", `${ang}rad`);
  },
};
