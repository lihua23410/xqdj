// 静电魔球的常驻皮肤：电荷（后端「静电充能」标记 0~10 层）折成 --charge，
// 球皮的辉光随之变亮；满电加脉冲，提示对手它随时要放。
window.lookFX = window.lookFX || {};

window.lookFX.staticball = {
  unmount(el) {
    el?.querySelector(":scope > .static-skin")?.remove();
  },
  guide(u, ctx) {
    const id = `static-warning-${u.id}`;
    let guide = document.getElementById(id);
    let warning = guide?.staticWarning;
    for (const fx of ctx.effects) {
      if (fx.kind !== u.kind || fx.unitId !== u.id) continue;
      if (fx.name === "static-warn") warning = fx;
      if (fx.name === "static-bolt" || fx.name === "static-cancel") warning = null;
    }
    if (!warning?.data || ctx.time < warning.data.started || ctx.time > warning.data.at + 0.05) return;
    guide = ctx.ensureGuide(id, "static-warning");
    guide.staticWarning = warning;
    const [x1, y1] = arena.screenPos(warning.x, warning.y, ctx.scale, ctx.cx, ctx.cy);
    const [x2, y2] = arena.screenPos(warning.vx, warning.vy, ctx.scale, ctx.cx, ctx.cy);
    ctx.placeSeg(guide, x1, y1, x2, y2, u.kind);
    guide.style.height = `${warning.data.width * ctx.scale}px`;
    const progress = (ctx.time - warning.data.started) / (warning.data.at - warning.data.started);
    guide.style.setProperty("--static-warning-progress", String(Math.min(1, Math.max(0, progress))));
    ctx.seenGuides.add(id);
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
