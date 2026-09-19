window.lookFX = window.lookFX || {};
window.engineHeat = window.engineHeat || {};

window.lookFX.engine = {
  unmount(el) {
    if (!el) return;
    el.querySelector(":scope > .engine-embers")?.remove();
    el.style.removeProperty("--engine-core");
    el.style.removeProperty("--engine-weak");
    el.style.removeProperty("--engine-frail");
    el.removeAttribute("data-engine-frail");
  },
  tick(el, u, ctx) {
    if (!el || !el.classList.contains("look-engine")) return;
    const st = window.engineHeat[u.id] || { gear: 1, weak: 0, frail: false };
    const gear = Math.max(1, Math.min(6, st.gear || 1));
    el.style.setProperty("--engine-core", String(gear / 6));
    el.style.setProperty("--engine-weak", String(st.frail ? 0 : Math.max(0, Math.min(1, st.weak || 0))));
    el.style.setProperty("--engine-frail", st.frail ? "1" : "0");
    el.dataset.engineFrail = st.frail ? "1" : "0";

    let host = el.querySelector(":scope > .engine-embers");
    if (!host) {
      host = document.createElement("div");
      host.className = "engine-embers";
      el.appendChild(host);
    }
    const scale = (ctx && ctx.scale) || 1;
    const vis = Math.max(0, u.vision || 0);
    const from = Math.max(10, (u.radius || 18) * scale);
    host.style.setProperty("--from", `${from}px`);
    host.style.setProperty("--reach", `${Math.max(from + 8, vis * scale)}px`);
    host.style.setProperty("--r", `${Math.max(36, vis * scale * 2)}px`);
    if (vis < 1) return;
    const now = (ctx && ctx.now) || performance.now();
    const gap = st.frail ? 900 : 720;
    if (now - (host._last || 0) < gap) return;
    host._last = now;
    const wave = document.createElement("i");
    wave.className = "engine-ripple";
    host.appendChild(wave);
    wave.addEventListener("animationend", () => wave.remove());
    const n = 3 + Math.min(3, gear);
    const spin = Math.random() * 360;
    for (let i = 0; i < n; i++) {
      const p = document.createElement("i");
      const jitter = (Math.random() - 0.5) * 16;
      p.className = i % 3 === 0 ? "engine-ember is-spark" : "engine-ember";
      p.style.setProperty("--ang", `${spin + (360 / n) * i + jitter}deg`);
      p.style.setProperty("--s", String(0.65 + Math.random() * 0.7));
      p.style.setProperty("--delay", `${Math.random() * 0.07}s`);
      host.appendChild(p);
      p.addEventListener("animationend", () => p.remove());
    }
  },
};
