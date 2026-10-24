// 巡回的常驻皮肤。
// patrol：标识物在膛时底座实心琥珀菱形；在外时（后端标记「巡回标识物在外」）
//         折成 class patrol-out——底座空心、球体降饱和脉冲。
// patrolmark：外圈警标菱环自旋，转速随速率（--mark-spin 越飞越快）；
//             内芯箭头指向飞行方向；--mark-heat 把速率折成辉光热度。
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
  unmount(el) {
    el?.querySelector(":scope > .patrolmark-art")?.remove();
  },
  tick(el, u) {
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
    const heat = Math.max(0, Math.min(1, (sp - 300) / 1200));
    art.style.setProperty("--mark-heat", heat.toFixed(3));
    art.style.setProperty("--mark-spin", `${Math.max(0.16, 0.6 - heat * 0.42).toFixed(3)}s`);
    // 屏幕 y 轴向下，vy 取反才是画面方向。
    const ang = Math.atan2(-(u.vy || 0), u.vx || 0);
    art.style.setProperty("--fly-ang", `${ang.toFixed(3)}rad`);
  },
};
