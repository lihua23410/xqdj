window.lookFX = window.lookFX || {};
window.thresholdLv = window.thresholdLv || {};

function thresholdText(x) {
  const n = Number(x);
  if (!Number.isFinite(n)) return "3";
  return String(Math.floor(n));
}

window.lookFX.threshold = {
  unmount(el) {
    el.querySelector(":scope > .threshold-read")?.remove();
  },
  tick(el, u) {
    const id = u && u.id;
    const raw = id != null && window.thresholdLv[id] != null ? window.thresholdLv[id] : 3;
    let node = el.querySelector(":scope > .threshold-read");
    if (!node) {
      node = document.createElement("div");
      node.className = "threshold-read";
      el.appendChild(node);
    }
    node.textContent = thresholdText(raw);
  },
};

arena.registerShot("threshold-read", (fx) => {
  if (fx && fx.unitId != null) window.thresholdLv[fx.unitId] = Number(fx.amount) || 0;
});
