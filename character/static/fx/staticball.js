// 静电魔球的常驻皮肤：电荷（后端「静电充能」标记 0~10 层）折成 --charge，
// 球皮的辉光随之变亮；满电加脉冲，提示对手它随时要放。
window.lookFX = window.lookFX || {};

window.lookFX.staticball = {
  unmount(el) {
    el?.querySelector(":scope > .static-skin")?.remove();
  },
  tick(el, u) {
    if (!el || !el.classList.contains("look-staticball")) return;
    let stacks = 0;
    for (const m of (u && u.marks) || []) {
      if (m && m.kind === "静电充能") stacks = Math.max(stacks, m.stacks || 0);
    }
    const c = Math.min(1, stacks / 10);
    el.style.setProperty("--charge", c.toFixed(2));
    el.classList.toggle("static-full", c >= 1);
    let skin = el.querySelector(":scope > .static-skin");
    if (!skin) {
      skin = document.createElement("i");
      skin.className = "static-skin";
      el.appendChild(skin);
    }
  },
};
