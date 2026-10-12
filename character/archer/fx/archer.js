// 弓箭皮肤：球身上一张弓，朝着面朝方向。
// 朝向优先用速度方向（球在动）；站定蓄力时速度是 0，就用 face FX 记下的角度。
// 拉弓状态由 shot.js 的 draw/volley/cancel 三个 FX 驱动。
window.lookFX = window.lookFX || {};
window.archerSt = window.archerSt || {};

function archerEnsure(el, cls, html) {
  let n = el.querySelector(`:scope > .${cls}`);
  if (!n) {
    n = document.createElement("div");
    n.className = cls;
    if (html) n.innerHTML = html;
    el.appendChild(n);
  }
  return n;
}

// archerFaceDeg 世界向量 → CSS 旋转角（度）：弓的素材是朝正东画的，所以 0° = 正东。
// 世界 y 朝上，CSS 的 y 朝下、rotate 又是顺时针，因此屏幕角 = atan2(-vy, vx)：
//   正东 (1,0) → 0°、正北 (0,1) → -90°、正西 (-1,0) → 180°、正南 (0,-1) → +90°。
// 这和 app.js 画扇环用的 faceAng = atan2(-faceY, faceX) 是同一套口径。
// 零向量返回 null（调用方自己决定"沿用上一帧"还是"朝东"）。
window.archerFaceDeg = function (vx, vy) {
  const x = Number(vx) || 0;
  const y = Number(vy) || 0;
  if (Math.hypot(x, y) < 1e-6) return null;
  return (Math.atan2(-y, x) * 180) / Math.PI;
};

// 在动就跟速度；站定（蓄力/后摇）沿用 face FX 记下的角
function archerFacingDeg(u, st) {
  const deg = window.archerFaceDeg(u && u.vx, u && u.vy);
  if (deg !== null) {
    st.face = deg;
    return deg;
  }
  return Number.isFinite(st.face) ? st.face : 0;
}

window.lookFX.archer = {
  unmount(el) {
    el.querySelector(":scope > .archer-bow")?.remove();
  },
  tick(el, u) {
    const st = (window.archerSt[u.id] = window.archerSt[u.id] || { face: 0, draw: 0 });
    const bow = archerEnsure(el, "archer-bow", '<i class="limb"></i><i class="string"></i>');
    bow.style.setProperty("--face", `${archerFacingDeg(u, st)}deg`);
    bow.classList.toggle("drawn", !!st.draw);
  },
};
