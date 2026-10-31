// 巡回的常驻皮肤。
// patrol：标识物在膛时底座实心琥珀菱形；在外时（后端标记「巡回标识物在外」）
//         折成 class patrol-out——底座空心、球体降饱和脉冲。
// patrolmark：警标方框自旋，沿实际轨迹留下锁定当时角度的空心方框残影。
//             旋转、残影淡出都按对局时间走；内芯箭头仍指向飞行方向。
window.lookFX = window.lookFX || {};

const PATROL_MARK_OUT = "巡回标识物在外";

window.lookFX.patrol = {
  unmount(el) {
    el?.querySelector(":scope > .patrol-base")?.remove();
  },
  tick(el, u) {
    if (!el || !el.classList.contains("look-patrol")) return;
    if (!el.querySelector(":scope > .patrol-base")) {
      const base = document.createElement("span");
      base.className = "patrol-base";
      base.innerHTML = '<span class="patrol-base-gem"></span>';
      el.appendChild(base);
    }
    const out = (u.marks || []).some((m) => m && m.kind === PATROL_MARK_OUT);
    el.classList.toggle("patrol-out", out);
  },
};

const PATROLMARK_SVG = `<svg viewBox="0 0 64 64" aria-hidden="true">
  <path fill="rgba(255,171,46,0.16)" stroke="#ffab2e" stroke-width="5" stroke-linejoin="miter"
    d="M32 7 L57 32 L32 57 L7 32 Z"/>
</svg>`;

const PATROLMARK_ARROW_SVG = `<svg viewBox="0 0 64 64" aria-hidden="true">
  <polygon fill="#ffd28a" points="28,21 28,43 48,32"/>
</svg>`;

window.lookFX.patrolmark = {
  guide(u, ctx) {
    const node = ctx.ensureGuide(`patrolmark-track-${u.id}`, "patrolmark-track");
    const time = ctx.time || 0;
    const speed = Math.hypot(u.vx || 0, u.vy || 0);
    const heat = Math.max(0, Math.min(1, speed * 0.014 / 26));
    const rate = 80 + heat * 220;
    const lifetime = 0.14;
    if (!node._patrolmark || time < node._patrolmark.time) {
      node._patrolmark = { time, angle: 0, rate, speed, x: u.x, y: u.y, samples: [] };
      node.innerHTML = `<svg aria-hidden="true">${Array.from({ length: 6 }, () =>
        '<path class="patrolmark-echo" d="M0 -25 25 0 0 25 -25 0Z" vector-effect="non-scaling-stroke"/>'
      ).join("")}</svg>`;
    }
    const data = node._patrolmark;
    const dt = time - data.time;
    if (dt > 0) {
      // 断帧或瞬移后重新留痕，避免把不连续的位置当成飞行轨迹。
      const jump = Math.hypot(u.x - data.x, u.y - data.y);
      if (dt > lifetime * 2 || jump > Math.max(speed, data.speed) * dt * 1.8 + u.radius * 2) data.samples = [];
      data.angle = (data.angle + (data.rate + rate) * 0.5 * dt) % 360;
      data.samples = data.samples.filter(p => time - p.at < lifetime);
      const last = data.samples[0];
      if (speed > 1 && (!last || Math.hypot(u.x - last.x, u.y - last.y) >= u.radius * 0.85)) {
        data.samples.unshift({ x: u.x, y: u.y, angle: data.angle, at: time });
        data.samples.length = Math.min(data.samples.length, 8);
      }
    }
    Object.assign(data, { time, rate, speed, x: u.x, y: u.y });
    const echoes = data.samples.filter(p => {
      const distance = Math.hypot(u.x - p.x, u.y - p.y);
      return distance >= u.radius * 0.55 && distance <= 130;
    }).slice(0, 6);
    node.style.setProperty("--kind", arena.kindColor(u.kind));
    node.querySelectorAll(".patrolmark-echo").forEach((path, i) => {
      const p = echoes[i];
      path.style.display = p ? "" : "none";
      if (!p) return;
      const fade = 1 - (time - p.at) / lifetime;
      const size = u.radius * 2 * 1.68 / 64 * ctx.scale * (0.64 + fade * 0.3);
      const [x, y] = arena.screenPos(p.x, p.y, ctx.scale, ctx.cx, ctx.cy);
      path.setAttribute("transform", `translate(${x.toFixed(2)} ${y.toFixed(2)}) rotate(${p.angle.toFixed(2)}) scale(${size.toFixed(4)})`);
      path.style.opacity = (fade * 0.48).toFixed(4);
    });
    ctx.seenGuides.add(node.id);
  },
  unmount(el) {
    el?.querySelector(":scope > .patrolmark-art")?.remove();
  },
  tick(el, u, ctx = {}) {
    if (!el || !el.classList.contains("look-patrolmark")) return;
    let art = el.querySelector(":scope > .patrolmark-art");
    if (!art) {
      art = document.createElement("span");
      art.className = "patrolmark-art";
      const spin = document.createElement("span");
      spin.className = "patrolmark-spin";
      spin.innerHTML = PATROLMARK_SVG;
      const arrow = document.createElement("span");
      arrow.className = "patrolmark-arrow";
      arrow.innerHTML = PATROLMARK_ARROW_SVG;
      art.appendChild(spin);
      art.appendChild(arrow);
      el.appendChild(art);
    }
    const sp = Math.hypot(u.vx || 0, u.vy || 0);
    const heat = Math.max(0, Math.min(1, sp * 0.014 / 26));
    const data = document.getElementById(`patrolmark-track-${u.id}`)?._patrolmark;
    art.style.setProperty("--mark-heat", heat.toFixed(3));
    art.style.setProperty("--mark-rotation", `${(data ? data.angle : (ctx.time || 0) * 80).toFixed(2)}deg`);
    // 屏幕 y 轴向下，vy 取反才是画面方向。
    const ang = Math.atan2(-(u.vy || 0), u.vx || 0);
    art.style.setProperty("--fly-ang", `${ang.toFixed(3)}rad`);
  },
};
