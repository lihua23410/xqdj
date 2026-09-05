window.lookFX = window.lookFX || {};
window.lookFX.slash = {
  tick(el, u) {
    if (!el || !el.classList.contains("look-slash")) return;
    const hexEl = document.getElementById("hex");
    const scale = (hexEl ? hexEl.clientWidth : 560) / 560;
    const diam = 36 * scale; // 小球直径（px），跟半径 18 对齐
    const len = (hexEl ? hexEl.clientWidth : 560) * 2.2; // 拉满后的刀光长度
    let host = el.querySelector(":scope > .look-iai");
    if (!host) {
      host = document.createElement("div");
      host.className = "look-iai";
      for (const name of ["iai-glow", "iai-blade", "iai-core", "iai-flare"]) {
        const layer = document.createElement("i");
        layer.className = name;
        host.appendChild(layer);
      }
      el.appendChild(host);
    }
    const ang = Math.atan2(-(u.vy || 0), u.vx || 1);
    host.style.setProperty("--slash-ang", `${ang}rad`);
    host.style.setProperty("--slash-short", `${diam * 2.5}px`); // 蓄力时短刀光长度
    host.style.setProperty("--slash-len", `${len}px`);
    host.style.setProperty("--slash-max", `${diam * 5}px`); // 拉满宽度 = 5 倍球径（和判定盒一致）
  },
};
