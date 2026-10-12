// 箭的皮肤：箭头用 fx/arrow.svg 当 mask（吃小球自己的 --kind 颜色 → 白箭），
// 拖尾自己画一条渐变长条（引擎的 Look.Trail 是每 16ms 复制一个"只有尺寸没有子元素"的
// ghost 点阵 —— 那种点点不是拖尾，所以关掉了，这里用 .archer-arrow-tail 顶替）。
// 两个子元素都按速度转（客户端 0° = +x，屏幕 y 朝下所以角度取 atan2(-vy, vx)）。
window.lookFX = window.lookFX || {};

function arrowArt(el, cls) {
  let n = el.querySelector(`:scope > .${cls}`);
  if (!n) {
    n = document.createElement("i");
    n.className = cls;
    el.appendChild(n);
  }
  return n;
}

window.lookFX.arrow = {
  unmount(el) {
    el.querySelector(":scope > .archer-arrow-art")?.remove();
    el.querySelector(":scope > .archer-arrow-tail")?.remove();
  },
  tick(el, u) {
    const tail = arrowArt(el, "archer-arrow-tail"); // 先建拖尾，箭头盖在上面
    const art = arrowArt(el, "archer-arrow-art");
    const vx = Number(u && u.vx) || 0;
    const vy = Number(u && u.vy) || 0;
    const deg = Math.hypot(vx, vy) > 1e-6 ? (Math.atan2(-vy, vx) * 180) / Math.PI : 0;
    const t = `translate(-50%, -50%) rotate(${deg}deg)`;
    tail.style.transform = t;
    art.style.transform = t;
  },
};
