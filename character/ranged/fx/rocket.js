// 小火箭：弹体长 22、宽 8，尾翼两片，尖头朝速度。尾焰长 = 速率 × 0.1，
// 从喷口向后，近白落到暖橙再淡掉。速率近 0 时不画焰，朝向留着上一次的。
window.lookFX = window.lookFX || {};

const rocketBody =
  "M11 0 L5 -4 L-5 -4 L-11 -7 L-7 -4 L-8 -2.4 L-8 2.4 L-7 4 L-11 7 L-5 4 L5 4 Z";

function rocketSvg(el) {
  let svg = el.querySelector(":scope > .rocket");
  if (svg) return svg;
  svg = document.createElementNS("http://www.w3.org/2000/svg", "svg");
  svg.setAttribute("class", "rocket");
  el.appendChild(svg);
  return svg;
}

window.lookFX.rocket = {
  unmount(el) {
    el.querySelector(":scope > .rocket")?.remove();
    delete el._rocketDeg;
  },
  tick(el, u, ctx) {
    const scale = (ctx && ctx.scale) || 1;
    const vx = Number(u && u.vx) || 0;
    const vy = Number(u && u.vy) || 0;
    const sp = Math.hypot(vx, vy);
    if (sp > 1e-3) el._rocketDeg = (Math.atan2(-vy, vx) * 180) / Math.PI;
    const deg = el._rocketDeg || 0;
    const flame = sp * 0.1;
    const show = flame > 0.5;
    const x0 = -11 - (show ? flame : 0) - 1;
    const y0 = -8;
    const w = 11 - x0;
    const h = 16;
    const svg = rocketSvg(el);
    const gid = `rf-${u && u.id != null ? u.id : "x"}`;
    const tip = -8 - flame;
    const fire = show
      ? `<defs><linearGradient id="${gid}" gradientUnits="userSpaceOnUse" x1="-8" y1="0" x2="${tip}" y2="0">` +
        `<stop offset="0" stop-color="#fff6d0"/>` +
        `<stop offset="0.4" stop-color="#f0a050"/>` +
        `<stop offset="1" stop-color="#f0a050" stop-opacity="0"/>` +
        `</linearGradient></defs>` +
        `<polygon points="-8,-2.6 ${tip},0 -8,2.6" fill="url(#${gid})"/>`
      : "";
    svg.setAttribute("viewBox", `${x0} ${y0} ${w} ${h}`);
    svg.style.width = `${w * scale}px`;
    svg.style.height = `${h * scale}px`;
    svg.style.left = `calc(50% + ${x0 * scale}px)`;
    svg.style.top = `calc(50% + ${y0 * scale}px)`;
    svg.style.transformOrigin = `${(-x0 / w) * 100}% ${(-y0 / h) * 100}%`;
    svg.style.transform = `rotate(${deg}deg)`;
    svg.innerHTML = fire + `<path d="${rocketBody}" fill="#f0a050"/>`;
  },
};
