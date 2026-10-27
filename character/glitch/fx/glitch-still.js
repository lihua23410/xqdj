window.lookFX = window.lookFX || {};
window.lookFX["glitch-still"] = {
  max: 28,
  modes: [
    "screen",
    "overlay",
    "difference",
    "exclusion",
    "lighter",
    "hard-light",
    "color-dodge",
    "multiply",
  ],
  unmount(el) {
    el?.querySelector(":scope > .look-glitch-slices")?.remove();
    if (el) delete el.dataset.glitchStill;
  },
  tick(el) {
    if (!el || !el.classList.contains("look-glitch-still")) return;
    if (el.dataset.glitchStill === "1") return;
    let host = el.querySelector(":scope > .look-glitch-slices");
    if (!host) {
      host = document.createElement("div");
      host.className = "look-glitch-slices";
      for (let n = 0; n < this.max; n++) host.appendChild(document.createElement("i"));
      el.appendChild(host);
    }
    const kids = host.children;
    let y = 0;
    let i = 0;
    while (y < 100 && i < this.max) {
      const h = 1.5 + Math.random() * 12;
      const gap = Math.random() < 0.45 ? Math.random() * 10 : 0;
      const slice = kids[i++];
      const bottom = Math.max(0, 100 - y - h);
      slice.style.display = "block";
      slice.style.clipPath = `inset(${y.toFixed(2)}% 0 ${bottom.toFixed(2)}% 0)`;
      slice.style.transform = `translate(${((Math.random() - 0.5) * 36).toFixed(1)}px, 0)`;
      slice.style.mixBlendMode = this.modes[(Math.random() * this.modes.length) | 0];
      slice.style.opacity = String(0.65 + Math.random() * 0.35);
      slice.style.background =
        Math.random() < 0.22 ? (Math.random() < 0.5 ? "#0ff" : "#f0f") : "var(--kind)";
      y += h + gap;
    }
    for (; i < this.max; i++) kids[i].style.display = "none";
    el.dataset.glitchStill = "1";
  },
};
